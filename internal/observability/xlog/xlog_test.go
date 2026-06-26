package xlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/gin-gonic/gin"
)

func TestDevelopmentLoggerUsesPrettyColorOutputAndContextAttrs(t *testing.T) {
	var buf bytes.Buffer
	xlog.Configure(xlog.Config{Env: "development", Level: slog.LevelDebug, Writer: &buf, Color: true})

	ctx := xlog.WithRequestID(context.Background(), "req-123")
	xlog.Component("test").InfoContext(ctx, "hello")

	out := buf.String()
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("development output should contain ANSI color: %q", out)
	}
	if !strings.Contains(out, "req-123") || !strings.Contains(out, "hello") {
		t.Fatalf("development output missing context/message: %q", out)
	}
}

func TestProductionLoggerUsesJSON(t *testing.T) {
	var buf bytes.Buffer
	xlog.Configure(xlog.Config{Env: "production", Level: slog.LevelInfo, Writer: &buf})

	ctx := xlog.WithRequestID(context.Background(), "req-json")
	xlog.Component("json-test").InfoContext(ctx, "json hello")

	var row map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &row); err != nil {
		t.Fatalf("production output should be JSON: %v; out=%q", err, buf.String())
	}
	if row["request_id"] != "req-json" || row["component"] != "json-test" || row["msg"] != "json hello" {
		t.Fatalf("json row = %#v", row)
	}
}

func TestLoggerInfoContextIncludesUserIDFromContext(t *testing.T) {
	var buf bytes.Buffer
	xlog.Configure(xlog.Config{Env: "production", Level: slog.LevelInfo, Writer: &buf})

	ctx := xlog.WithUserID(context.Background(), "user-123")
	xlog.Component("auth").InfoContext(ctx, "verified")

	var row map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &row); err != nil {
		t.Fatalf("production output should be JSON: %v; out=%q", err, buf.String())
	}
	if row["user_id"] != "user-123" || row["component"] != "auth" {
		t.Fatalf("json row = %#v", row)
	}
}

func TestGinMiddlewareLogsRequestFields(t *testing.T) {
	var buf bytes.Buffer
	xlog.Configure(xlog.Config{Env: "production", Level: slog.LevelInfo, Writer: &buf})
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(xlog.GinMiddleware())
	router.GET("/ping", func(c *gin.Context) { c.String(201, "pong") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Request-ID", "req-mid")
	router.ServeHTTP(w, req)

	var row map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &row); err != nil {
		t.Fatalf("middleware output should be JSON: %v; out=%q", err, buf.String())
	}
	if row["request_id"] != "req-mid" || row["path"] != "/ping" || row["method"] != "GET" {
		t.Fatalf("log row missing request fields: %#v", row)
	}
	if row["status"].(float64) != 201 {
		t.Fatalf("status = %#v, want 201", row["status"])
	}
}
