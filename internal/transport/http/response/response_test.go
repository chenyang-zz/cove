package response

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/boxify/api-go/internal/xerr"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func TestFromErrorIncludesValidationFieldErrors(t *testing.T) {
	type createRequest struct {
		APIKey   string `json:"api_key" binding:"required"`
		Provider string `json:"provider" binding:"oneof=openai deepseek"`
	}
	registerValidationTagNames()
	err := binding.Validator.ValidateStruct(createRequest{Provider: "invalid"})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	FromError(c, xerr.Validation(err))

	var got Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal response: %v", err)
	}
	if got.Code != 40001 || got.Message != "请求参数错误" {
		t.Fatalf("envelope = %+v, want validation envelope", got)
	}
	if len(got.Errors) != 2 {
		t.Fatalf("errors len = %d, want 2: %+v", len(got.Errors), got.Errors)
	}
	assertFieldError(t, got.Errors, "api_key", "required", "")
	assertFieldError(t, got.Errors, "provider", "oneof", "openai deepseek")
	if bytes.Contains(w.Body.Bytes(), []byte("Key:")) {
		t.Fatalf("response leaked raw validator error: %s", w.Body.String())
	}
}

func TestFromErrorIncludesJSONBindingErrors(t *testing.T) {
	for _, tt := range []struct {
		name      string
		err       error
		wantField string
		wantTag   string
	}{
		{
			name:    "syntax",
			err:     &json.SyntaxError{Offset: 3},
			wantTag: "json",
		},
		{
			name:      "type",
			err:       &json.UnmarshalTypeError{Field: "api_key", Value: "number", Type: nil},
			wantField: "api_key",
			wantTag:   "type",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			FromError(c, xerr.Validation(tt.err))

			var got Envelope
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("Unmarshal response: %v", err)
			}
			assertFieldError(t, got.Errors, tt.wantField, tt.wantTag, "")
		})
	}
}

func TestFromErrorDoesNotIncludeErrorsForNonValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	FromError(c, xerr.Unauthorized("请先登录"))

	var got Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal response: %v", err)
	}
	if len(got.Errors) != 0 {
		t.Fatalf("errors = %+v, want empty", got.Errors)
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestFromErrorLogsConflictAsWarn(t *testing.T) {
	var buf bytes.Buffer
	restore := useJSONLogger(&buf)
	defer restore()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/register", nil)

	FromError(c, xerr.UserExists())

	row := decodeLogRow(t, &buf)
	if row["level"] != "WARN" || row["msg"] != "http_error" {
		t.Fatalf("log row = %#v, want warn http_error", row)
	}
	if row["component"] != "transport.http.response" ||
		row["method"] != http.MethodPost ||
		row["path"] != "/api/auth/register" ||
		row["status"] != float64(http.StatusConflict) ||
		row["error_kind"] != string(xerr.KindConflict) ||
		row["error_code"] != float64(40901) {
		t.Fatalf("log row missing fields: %#v", row)
	}
}

func TestFromErrorLogsInternalCauseWithoutLeakingResponse(t *testing.T) {
	var buf bytes.Buffer
	restore := useJSONLogger(&buf)
	defer restore()

	cause := errors.New("pq: duplicate key")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/users", nil)

	FromError(c, xerr.Internal("保存用户失败", cause))

	if bytes.Contains(w.Body.Bytes(), []byte("pq: duplicate key")) {
		t.Fatalf("response leaked cause: %s", w.Body.String())
	}
	row := decodeLogRow(t, &buf)
	if row["level"] != "ERROR" ||
		row["error"] != "保存用户失败: pq: duplicate key" ||
		row["cause"] != "pq: duplicate key" {
		t.Fatalf("internal log row = %#v", row)
	}
	if row["error_file"] == "" || row["error_line"] == nil || row["error_func"] == "" {
		t.Fatalf("internal log missing location fields: %#v", row)
	}
	if !strings.Contains(row["error_func"].(string), "TestFromErrorLogsInternalCauseWithoutLeakingResponse") {
		t.Fatalf("error_func = %v, want caller test", row["error_func"])
	}
	if bytes.Contains(w.Body.Bytes(), []byte("error_file")) ||
		bytes.Contains(w.Body.Bytes(), []byte("response_test.go")) {
		t.Fatalf("response leaked error location: %s", w.Body.String())
	}
}

func TestFromErrorLogsValidationFieldDetails(t *testing.T) {
	var buf bytes.Buffer
	restore := useJSONLogger(&buf)
	defer restore()

	type createRequest struct {
		APIKey string `json:"api_key" binding:"required"`
	}
	registerValidationTagNames()
	err := binding.Validator.ValidateStruct(createRequest{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/model-configs", nil)

	FromError(c, xerr.Validation(err))

	row := decodeLogRow(t, &buf)
	if row["level"] != "WARN" {
		t.Fatalf("level = %v, want WARN", row["level"])
	}
	items, ok := row["validation_errors"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("validation_errors = %#v, want one item", row["validation_errors"])
	}
	item, ok := items[0].(map[string]any)
	if !ok || item["field"] != "api_key" || item["tag"] != "required" {
		t.Fatalf("validation error item = %#v", item)
	}
	if row["error_file"] == "" || row["error_line"] == nil || row["error_func"] == "" {
		t.Fatalf("validation log missing location fields: %#v", row)
	}
}

func assertFieldError(t *testing.T, errors []FieldError, field, tag, param string) {
	t.Helper()
	for _, item := range errors {
		if item.Field == field && item.Tag == tag {
			if param != "" && item.Param != param {
				t.Fatalf("field error %+v param, want %q", item, param)
			}
			if item.Message == "" {
				t.Fatalf("field error %+v missing message", item)
			}
			return
		}
	}
	t.Fatalf("field error %s/%s not found in %+v", field, tag, errors)
}

func registerValidationTagNames() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(jsonTagName)
	}
}

func useJSONLogger(buf *bytes.Buffer) func() {
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	return func() {
		slog.SetDefault(old)
	}
}

func decodeLogRow(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	out := strings.TrimSpace(buf.String())
	if out == "" {
		t.Fatal("log output is empty")
	}
	var row map[string]any
	if err := json.Unmarshal([]byte(out), &row); err != nil {
		t.Fatalf("log should be JSON: %v; out=%q", err, out)
	}
	return row
}
