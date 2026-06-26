package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/boxify/api-go/internal/app"
	"github.com/boxify/api-go/internal/config"
	"github.com/boxify/api-go/internal/infrastructure/db/postgres"
	"github.com/boxify/api-go/internal/observability/xlog"
	repositorypostgres "github.com/boxify/api-go/internal/repository/postgres"
	httptransport "github.com/boxify/api-go/internal/transport/http"
)

func main() {
	cfg := config.Load()
	xlog.Configure(xlog.Config{
		Env:   cfg.App.Env,
		Level: slog.LevelInfo,
		Color: true,
	})
	cipher, err := app.NewSecretCipher(cfg.SecretKey)
	if err != nil {
		slog.Error("create secret cipher", "error", err)
		os.Exit(1)
	}
	db, err := postgres.NewGormDB(context.Background(), postgres.Config{URL: cfg.Database.URL})
	if err != nil {
		slog.Error("connect postgres", "error", err)
		os.Exit(1)
	}
	sqlDB, err := db.DB()
	if err != nil {
		slog.Error("get postgres db", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			slog.Error("close postgres", "error", err)
		}
	}()
	userRepo := repositorypostgres.NewUserRepository(db)
	refreshTokenRepo := repositorypostgres.NewRefreshTokenRepository(db)
	router := httptransport.NewRouter(httptransport.Dependencies{
		AuthService:        app.NewAuthService(userRepo, refreshTokenRepo, cfg.JWT.Secret),
		ChatService:        app.NewChatService(),
		ModelConfigService: app.NewModelConfigService(cipher),
	})
	server := &http.Server{
		Addr:              cfg.HTTPAddr(),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	slog.Info("api server starting", "addr", cfg.HTTPAddr())
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("api server stopped", "error", err)
		os.Exit(1)
	}
}
