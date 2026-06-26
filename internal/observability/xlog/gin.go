package xlog

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const ginUserIDKey = "user_id"

func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		ctx := WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()

		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
		}
		if userID, ok := c.Get(ginUserIDKey); ok {
			if id, ok := userID.(uuid.UUID); ok {
				attrs = append(attrs, slog.String("user_id", id.String()))
			}
		}
		L(c.Request.Context()).Info("http_request", attrsToArgs(attrs)...)
	}
}

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				err := xerr.Internal("系统开小差了", fmt.Errorf("%v", recovered))
				L(c.Request.Context()).Error(
					"panic_recovered",
					"panic", fmt.Sprint(recovered),
					"stack", string(debug.Stack()),
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
				)
				response.FromError(c, err)
				c.Abort()
			}
		}()
		c.Next()
	}
}

func attrsToArgs(attrs []slog.Attr) []any {
	args := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		args = append(args, attr)
	}
	return args
}

var _ = http.StatusOK
