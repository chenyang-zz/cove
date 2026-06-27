package response

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/boxify/api-go/internal/xerr"
	"github.com/gin-gonic/gin"
)

const suppressErrorLogKey = "response_suppress_error_log"

type Envelope struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Data    any          `json:"data,omitempty"`
	Errors  []FieldError `json:"errors,omitempty"`
}

type FieldError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Param   string `json:"param"`
	Message string `json:"message"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Code: 0, Message: "ok", Data: data})
}

func FromError(c *gin.Context, err error) {
	status, code, message := xerr.ToHTTP(err)
	fieldErrors := validationFieldErrors(err)
	logHTTPError(c, err, status, code, message, fieldErrors)
	c.JSON(status, Envelope{Code: code, Message: message, Errors: fieldErrors})
}

func BadRequest(c *gin.Context, err error) {
	FromError(c, xerr.Validation(err))
}

func SuppressErrorLog(c *gin.Context) {
	if c != nil {
		c.Set(suppressErrorLogKey, true)
	}
}

func logHTTPError(c *gin.Context, err error, status int, code int, safeMessage string, fieldErrors []FieldError) {
	if err == nil || shouldSuppressErrorLog(c) {
		return
	}
	appErr := xerr.From(err)
	if appErr == nil {
		return
	}

	ctx := context.Background()
	method := ""
	path := ""
	if c != nil && c.Request != nil {
		ctx = c.Request.Context()
		method = c.Request.Method
		path = c.Request.URL.Path
	}

	attrs := []slog.Attr{
		slog.String("component", "transport.http.response"),
		slog.String("method", method),
		slog.String("path", path),
		slog.Int("status", status),
		slog.String("error_kind", string(appErr.Kind)),
		slog.Int("error_code", code),
		slog.String("safe_message", safeMessage),
		slog.String("error", safeErrorString(err)),
	}
	if appErr.Cause != nil {
		attrs = append(attrs, slog.String("cause", safeErrorString(appErr.Cause)))
	}
	if appErr.Location != nil {
		attrs = append(attrs,
			slog.String("error_file", appErr.Location.File),
			slog.Int("error_line", appErr.Location.Line),
			slog.String("error_func", appErr.Location.Func),
		)
	}
	if len(appErr.Fields) > 0 {
		attrs = append(attrs, slog.Any("error_fields", appErr.Fields))
	}
	if len(fieldErrors) > 0 {
		attrs = append(attrs, slog.Any("validation_errors", fieldErrors))
	}

	level := slog.LevelWarn
	if status >= http.StatusInternalServerError {
		level = slog.LevelError
	}
	slog.Default().LogAttrs(ctx, level, "http_error", attrs...)
}

func safeErrorString(err error) (text string) {
	if err == nil {
		return ""
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			text = fmt.Sprintf("%T error string panic: %v", err, recovered)
		}
	}()
	return err.Error()
}

func shouldSuppressErrorLog(c *gin.Context) bool {
	if c == nil {
		return false
	}
	value, ok := c.Get(suppressErrorLogKey)
	if !ok {
		return false
	}
	suppressed, _ := value.(bool)
	return suppressed
}
