package http_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/boxify/api-go/internal/observability/xlog"
)

func TestDuplicateRegisterUsesUnifiedAppError(t *testing.T) {
	router := newTestRouter(t)
	body := `{"username":"dup","email":"dup@example.com","password":"secret123"}`

	first := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(first, req)
	if first.Code != http.StatusOK {
		t.Fatalf("first register status = %d body=%s", first.Code, first.Body.String())
	}

	second := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(second, req)
	if second.Code != http.StatusConflict {
		t.Fatalf("second register status = %d body=%s", second.Code, second.Body.String())
	}
	if strings.TrimSpace(second.Body.String()) != `{"code":40901,"message":"用户已存在"}` {
		t.Fatalf("body = %s", second.Body.String())
	}
}

func TestPanicRecoveryReturnsUnifiedErrorAndLogsStack(t *testing.T) {
	var buf bytes.Buffer
	xlog.Configure(xlog.Config{Env: "production", Level: slog.LevelInfo, Writer: &buf})
	router := newTestRouter(t, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/debug/panic", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":50000`) {
		t.Fatalf("body = %s", w.Body.String())
	}

	rows := decodeJSONLogRows(t, buf.Bytes())
	var panicRow map[string]any
	httpErrorCount := 0
	for _, row := range rows {
		switch row["msg"] {
		case "panic_recovered":
			panicRow = row
		case "http_error":
			httpErrorCount++
		}
	}
	if panicRow == nil || panicRow["stack"] == "" || panicRow["panic"] == "" {
		t.Fatalf("panic log missing stack/panic: %#v", rows)
	}
	if httpErrorCount != 0 {
		t.Fatalf("panic recovery logged duplicate http_error count=%d rows=%#v", httpErrorCount, rows)
	}
}

func decodeJSONLogRows(t *testing.T, data []byte) []map[string]any {
	t.Helper()
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var rows []map[string]any
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal(line, &row); err != nil {
			t.Fatalf("panic log should be JSON: %v; line=%q out=%q", err, string(line), string(data))
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan logs: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("no log rows; out=%q", string(data))
	}
	return rows
}
