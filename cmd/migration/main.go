package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/boxify/api-go/internal/config"
	"github.com/boxify/api-go/internal/infrastructure/db/migration"
)

type migrator interface {
	Up(context.Context) error
}

func main() {
	if err := run(); err != nil {
		slog.Error("database migration failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	runner, err := migration.NewRunner(migration.Config{DatabaseURL: cfg.Database.URL})
	if err != nil {
		return err
	}
	defer func() {
		if err := runner.Close(); err != nil {
			slog.Error("close migration runner", "error", err)
		}
	}()
	return runCommand(context.Background(), runner)
}

func runCommand(ctx context.Context, runner migrator) error {
	return runner.Up(ctx)
}
