package xlog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Env    string
	Level  slog.Level
	Writer io.Writer
	Color  bool
}

func Configure(cfg Config) {
	writer := cfg.Writer
	if writer == nil {
		writer = os.Stdout
	}
	level := &slog.LevelVar{}
	level.Set(cfg.Level)
	if cfg.Env == "" {
		cfg.Env = os.Getenv("APP_ENV")
	}
	var handler slog.Handler
	if cfg.Env == "" || cfg.Env == "development" || cfg.Env == "dev" || cfg.Env == "local" {
		handler = newPrettyHandler(writer, &slog.HandlerOptions{Level: level}, cfg.Color)
	} else {
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: level})
	}
	handler = contextHandler{next: handler}
	slog.SetDefault(slog.New(handler))
}

type contextHandler struct {
	next slog.Handler
}

func (h contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h contextHandler) Handle(ctx context.Context, record slog.Record) error {
	record = record.Clone()
	if attrs, ok := ctx.Value(attrsKey).([]slog.Attr); ok {
		record.AddAttrs(attrs...)
	}
	return h.next.Handle(ctx, record)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{next: h.next.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{next: h.next.WithGroup(name)}
}

type prettyHandler struct {
	writer io.Writer
	opts   *slog.HandlerOptions
	color  bool
	attrs  []slog.Attr
	group  string
	mu     *sync.Mutex
}

func newPrettyHandler(writer io.Writer, opts *slog.HandlerOptions, color bool) slog.Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &prettyHandler{writer: writer, opts: opts, color: color, mu: &sync.Mutex{}}
}

func (h *prettyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if h.opts.Level == nil {
		return level >= slog.LevelInfo
	}
	return level >= h.opts.Level.Level()
}

func (h *prettyHandler) Handle(ctx context.Context, record slog.Record) error {
	var b strings.Builder
	level := record.Level.String()
	if h.color {
		level = colorizeLevel(record.Level, level)
	}
	b.WriteString(record.Time.Format(time.RFC3339))
	b.WriteByte(' ')
	b.WriteString(level)
	b.WriteByte(' ')
	b.WriteString(record.Message)
	allAttrs := append([]slog.Attr{}, h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		allAttrs = append(allAttrs, attr)
		return true
	})
	for _, attr := range allAttrs {
		attr.Value = attr.Value.Resolve()
		if attr.Equal(slog.Attr{}) {
			continue
		}
		key := attr.Key
		if h.group != "" {
			key = h.group + "." + key
		}
		b.WriteByte(' ')
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(fmt.Sprint(attr.Value.Any()))
	}
	b.WriteByte('\n')
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.writer, b.String())
	return err
}

func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	next.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &next
}

func (h *prettyHandler) WithGroup(name string) slog.Handler {
	next := *h
	if next.group == "" {
		next.group = name
	} else {
		next.group += "." + name
	}
	return &next
}

func colorizeLevel(level slog.Level, text string) string {
	const reset = "\x1b[0m"
	switch {
	case level >= slog.LevelError:
		return "\x1b[31m" + text + reset
	case level >= slog.LevelWarn:
		return "\x1b[33m" + text + reset
	case level >= slog.LevelInfo:
		return "\x1b[32m" + text + reset
	default:
		return "\x1b[36m" + text + reset
	}
}
