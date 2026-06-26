package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/boxify/api-go/internal/xerr"
	"github.com/jackc/pgx/v5"
)

func TestPoolConfigRejectsBlankURL(t *testing.T) {
	_, err := poolConfig(Config{URL: "   "})
	if err == nil {
		t.Fatal("poolConfig returned nil error")
	}
	if xerr.From(err).Kind != xerr.KindBadRequest {
		t.Fatalf("error kind = %s, want %s", xerr.From(err).Kind, xerr.KindBadRequest)
	}
}

func TestPoolConfigTrimsURLAndAppliesOverrides(t *testing.T) {
	cfg, err := poolConfig(Config{
		URL:               "  postgres://user:pass@localhost:5432/app?sslmode=disable  ",
		MinConns:          2,
		MaxConns:          8,
		MaxConnLifetime:   time.Minute,
		MaxConnIdleTime:   2 * time.Minute,
		HealthCheckPeriod: 3 * time.Minute,
	})
	if err != nil {
		t.Fatalf("poolConfig error = %v", err)
	}
	if cfg.ConnConfig.Config.Database != "app" {
		t.Fatalf("database = %q, want app", cfg.ConnConfig.Config.Database)
	}
	if cfg.MinConns != 2 || cfg.MaxConns != 8 {
		t.Fatalf("pool conns = min %d max %d, want 2/8", cfg.MinConns, cfg.MaxConns)
	}
	if cfg.MaxConnLifetime != time.Minute {
		t.Fatalf("MaxConnLifetime = %s, want 1m", cfg.MaxConnLifetime)
	}
	if cfg.MaxConnIdleTime != 2*time.Minute {
		t.Fatalf("MaxConnIdleTime = %s, want 2m", cfg.MaxConnIdleTime)
	}
	if cfg.HealthCheckPeriod != 3*time.Minute {
		t.Fatalf("HealthCheckPeriod = %s, want 3m", cfg.HealthCheckPeriod)
	}
}

func TestPoolConfigLeavesDefaultsWhenOverridesAreZero(t *testing.T) {
	defaults, err := poolConfig(Config{URL: "postgres://user:pass@localhost:5432/app?sslmode=disable"})
	if err != nil {
		t.Fatalf("default poolConfig error = %v", err)
	}
	zeroOverrides, err := poolConfig(Config{
		URL:               "postgres://user:pass@localhost:5432/app?sslmode=disable",
		MinConns:          0,
		MaxConns:          0,
		MaxConnLifetime:   0,
		MaxConnIdleTime:   0,
		HealthCheckPeriod: 0,
	})
	if err != nil {
		t.Fatalf("zero override poolConfig error = %v", err)
	}
	if zeroOverrides.MinConns != defaults.MinConns || zeroOverrides.MaxConns != defaults.MaxConns {
		t.Fatalf("zero overrides changed conns: got %d/%d want %d/%d", zeroOverrides.MinConns, zeroOverrides.MaxConns, defaults.MinConns, defaults.MaxConns)
	}
	if zeroOverrides.MaxConnLifetime != defaults.MaxConnLifetime ||
		zeroOverrides.MaxConnIdleTime != defaults.MaxConnIdleTime ||
		zeroOverrides.HealthCheckPeriod != defaults.HealthCheckPeriod {
		t.Fatal("zero duration overrides changed pgxpool defaults")
	}
}

func TestNilClientCloseIsSafe(t *testing.T) {
	var client *Client
	client.Close()
}

func TestTxRejectsNilFunction(t *testing.T) {
	err := (&Client{}).Tx(context.Background(), nil)
	if err == nil {
		t.Fatal("Tx returned nil error")
	}
	if xerr.From(err).Kind != xerr.KindBadRequest {
		t.Fatalf("error kind = %s, want %s", xerr.From(err).Kind, xerr.KindBadRequest)
	}
}

func TestClientIntegrationWhenPostgresEnvIsConfigured(t *testing.T) {
	url := os.Getenv("POSTGRES_TEST_URL")
	if url == "" {
		t.Skip("POSTGRES_TEST_URL is required")
	}

	ctx := context.Background()
	client, err := NewClient(ctx, Config{URL: url})
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}
	defer client.Close()

	if client.Pool() == nil {
		t.Fatal("Pool returned nil")
	}
	if err := client.Verify(ctx); err != nil {
		t.Fatalf("Verify error = %v", err)
	}

	table := "codex_postgres_client_test"
	if _, err := client.Pool().Exec(ctx, "DROP TABLE IF EXISTS "+table); err != nil {
		t.Fatalf("drop table before test: %v", err)
	}
	if _, err := client.Pool().Exec(ctx, "CREATE TABLE "+table+" (id TEXT PRIMARY KEY)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	t.Cleanup(func() {
		_, _ = client.Pool().Exec(context.Background(), "DROP TABLE IF EXISTS "+table)
	})

	if err := client.Tx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO "+table+" (id) VALUES ($1)", "committed")
		return err
	}); err != nil {
		t.Fatalf("Tx commit path error = %v", err)
	}
	var committed string
	if err := client.Pool().QueryRow(ctx, "SELECT id FROM "+table+" WHERE id = $1", "committed").Scan(&committed); err != nil {
		t.Fatalf("query committed row: %v", err)
	}
	if committed != "committed" {
		t.Fatalf("committed = %q, want committed", committed)
	}

	sentinel := errors.New("rollback sentinel")
	err = client.Tx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "INSERT INTO "+table+" (id) VALUES ($1)", "rolled-back"); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("Tx rollback error = %v, want sentinel", err)
	}
	var count int
	if err := client.Pool().QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE id = $1", "rolled-back").Scan(&count); err != nil {
		t.Fatalf("query rollback count: %v", err)
	}
	if count != 0 {
		t.Fatalf("rollback row count = %d, want 0", count)
	}
}

func TestNewClientRejectsBlankURL(t *testing.T) {
	_, err := NewClient(context.Background(), Config{URL: strings.Repeat(" ", 2)})
	if err == nil {
		t.Fatal("NewClient returned nil error")
	}
	if xerr.From(err).Kind != xerr.KindBadRequest {
		t.Fatalf("error kind = %s, want %s", xerr.From(err).Kind, xerr.KindBadRequest)
	}
}

func TestNewGormDBRejectsBlankURL(t *testing.T) {
	_, err := NewGormDB(context.Background(), Config{URL: strings.Repeat(" ", 2)})
	if err == nil {
		t.Fatal("NewGormDB returned nil error")
	}
	if xerr.From(err).Kind != xerr.KindBadRequest {
		t.Fatalf("error kind = %s, want %s", xerr.From(err).Kind, xerr.KindBadRequest)
	}
}
