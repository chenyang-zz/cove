package xlog

import (
	"context"
	"log/slog"
)

type contextKey string

const attrsKey contextKey = "xlog_attrs"

func With(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	existing, _ := ctx.Value(attrsKey).([]slog.Attr)
	next := make([]slog.Attr, 0, len(existing)+len(attrs))
	next = append(next, existing...)
	next = append(next, attrs...)
	return context.WithValue(ctx, attrsKey, next)
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		return ctx
	}
	return With(ctx, slog.String("request_id", requestID))
}

func WithUserID(ctx context.Context, userID string) context.Context {
	if userID == "" {
		return ctx
	}
	return With(ctx, slog.String("user_id", userID))
}

func WithComponent(ctx context.Context, component string) context.Context {
	if component == "" {
		return ctx
	}
	return With(ctx, slog.String("component", component))
}

func WithError(ctx context.Context, err error) context.Context {
	if err == nil {
		return ctx
	}
	return With(ctx, slog.String("error", err.Error()))
}

func Component(component string) *slog.Logger {
	if component == "" {
		return slog.Default()
	}
	return slog.Default().With(slog.String("component", component))
}

func L(ctx context.Context) *slog.Logger {
	logger := slog.Default()
	if ctx == nil {
		return logger
	}
	attrs, _ := ctx.Value(attrsKey).([]slog.Attr)
	args := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		args = append(args, attr)
	}
	if len(args) == 0 {
		return logger
	}
	return logger.With(args...)
}
