package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/boxify/api-go/internal/config"
	gatewayruntime "github.com/boxify/api-go/internal/gateway/runtime"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/gatewayhttp"
)

func main() {
	cfg := config.Load()
	xlog.Configure(xlog.Config{Env: cfg.App.Env, Level: slog.LevelInfo, Color: true})
	if !cfg.Gateway.Enabled {
		slog.Info("消息网关未启用")
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	svcCtx, err := svc.New(ctx, cfg)
	if err != nil {
		slog.Error("初始化网关服务失败", "错误", err)
		os.Exit(1)
	}
	defer func() { _ = svcCtx.Close(context.Background()) }()
	server := &http.Server{Addr: cfg.GatewayAddr(), Handler: gatewayhttp.NewRouter(svcCtx), ReadHeaderTimeout: 10 * time.Second}
	errCh := make(chan error, 2)
	go func() { errCh <- gatewayruntime.NewManager(svcCtx).Run(ctx) }()
	go func() {
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errCh <- err
	}()
	slog.Info("消息网关启动中", "地址", cfg.GatewayAddr())
	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			slog.Error("消息网关异常停止", "错误", err)
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}
