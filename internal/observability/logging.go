package observability

import (
	"log/slog"

	"github.com/boxify/api-go/internal/observability/xlog"
)

func ConfigureLogger(level slog.Level) {
	xlog.Configure(xlog.Config{Level: level})
}
