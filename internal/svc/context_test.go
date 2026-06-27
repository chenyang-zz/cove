package svc_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/boxify/api-go/internal/config"
	"github.com/boxify/api-go/internal/infrastructure/storage"
	"github.com/boxify/api-go/internal/svc"
)

func TestNewReturnsErrorForInvalidPostgresURL(t *testing.T) {
	cfg := config.Config{}
	cfg.Database.URL = "   "
	cfg.JWT.Secret = "test-secret"
	cfg.JWT.AccessTokenTTL = "168h"
	cfg.SecretKey = "0123456789abcdef0123456789abcdef"

	if _, err := svc.New(context.Background(), cfg); err == nil {
		t.Fatal("svc.New error = nil, want error")
	}
}

func TestNewReturnsErrorForInvalidAccessTokenTTL(t *testing.T) {
	cfg := config.Config{}
	cfg.Database.URL = "   "
	cfg.JWT.Secret = "test-secret"
	cfg.JWT.AccessTokenTTL = "not-a-duration"
	cfg.SecretKey = "0123456789abcdef0123456789abcdef"

	_, err := svc.New(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "JWT access token TTL 配置无效") {
		t.Fatalf("svc.New error = %v, want invalid ttl error", err)
	}
}

func TestCloseCanBeCalledRepeatedly(t *testing.T) {
	svcCtx := &svc.ServiceContext{Storage: storage.NewLocalStore(t.TempDir())}
	ctx := context.Background()

	if err := svcCtx.Close(ctx); err != nil {
		t.Fatalf("first Close error = %v", err)
	}
	if err := svcCtx.Close(ctx); err != nil {
		t.Fatalf("second Close error = %v", err)
	}
}

func TestServiceContextCanHoldLocalStorage(t *testing.T) {
	svcCtx := &svc.ServiceContext{Storage: storage.NewLocalStore(t.TempDir())}

	if svcCtx.Storage == nil {
		t.Fatal("storage = nil, want local store")
	}
}

func TestBuildStorageReturnsErrorForIncompleteCOSConfig(t *testing.T) {
	_, err := svc.BuildStorage(config.StorageConfig{Backend: "cos"})
	if err == nil || !strings.Contains(err.Error(), "COS 存储配置无效") {
		t.Fatalf("BuildStorage error = %v, want cos config error", err)
	}
}

func TestBuildStorageReturnsErrorForUnknownBackend(t *testing.T) {
	_, err := svc.BuildStorage(config.StorageConfig{Backend: "unknown"})
	if err == nil || !strings.Contains(err.Error(), "存储 backend 配置无效") {
		t.Fatalf("BuildStorage error = %v, want backend error", err)
	}
}

func TestBuildStorageReturnsLocalStoreByDefault(t *testing.T) {
	store, err := svc.BuildStorage(config.StorageConfig{Backend: "local", Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("BuildStorage error = %v", err)
	}
	if store == nil {
		t.Fatal("store = nil")
	}
}

func TestCloseReturnsStorageCloseError(t *testing.T) {
	want := errors.New("close storage")
	svcCtx := &svc.ServiceContext{Storage: closeStore{err: want}}

	if err := svcCtx.Close(context.Background()); !errors.Is(err, want) {
		t.Fatalf("Close error = %v, want %v", err, want)
	}
}

type closeStore struct {
	err error
}

func (s closeStore) Put(context.Context, string, []byte) error   { return nil }
func (s closeStore) Get(context.Context, string) ([]byte, error) { return nil, nil }
func (s closeStore) Delete(context.Context, string) error        { return nil }
func (s closeStore) Ping(context.Context) error                  { return nil }
func (s closeStore) Close() error                                { return s.err }
