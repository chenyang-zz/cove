package xerr_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/boxify/api-go/internal/xerr"
)

func TestWrapPreservesCauseAndHidesUnsafeMessageFromHTTP(t *testing.T) {
	cause := errors.New("database password leaked in detail")

	err := xerr.Wrap(cause, "系统开小差了")

	if !errors.Is(err, cause) {
		t.Fatal("wrapped error should preserve cause for errors.Is")
	}
	var appErr *xerr.AppError
	if !errors.As(err, &appErr) {
		t.Fatal("wrapped error should support errors.As AppError")
	}
	status, code, msg := xerr.ToHTTP(err)
	if status != 500 || code != 50000 || msg != "系统开小差了" {
		t.Fatalf("http = (%d,%d,%q), want (500,50000,系统开小差了)", status, code, msg)
	}
	if msg == cause.Error() {
		t.Fatal("safe HTTP message leaked unsafe cause")
	}
}

func TestKnownErrorMappings(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   int
	}{
		{"bad request", xerr.BadRequest("参数错误"), 400, 40000},
		{"unauthorized", xerr.Unauthorized("请先登录"), 401, 40100},
		{"conflict", xerr.Conflict("资源已存在"), 409, 40900},
		{"user exists", xerr.UserExists(), 409, 40901},
		{"invalid credential", xerr.InvalidCredential(), 401, 40103},
		{"unknown", errors.New("boom"), 500, 50000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, code, _ := xerr.ToHTTP(tc.err)
			if status != tc.status || code != tc.code {
				t.Fatalf("mapping = (%d,%d), want (%d,%d)", status, code, tc.status, tc.code)
			}
		})
	}
}

func TestFormattedErrors(t *testing.T) {
	status, code, msg := xerr.ToHTTP(xerr.NotFoundf("知识库 %s 不存在", "kb1"))
	if status != 404 || code != 40400 || msg != "知识库 kb1 不存在" {
		t.Fatalf("not found f = (%d,%d,%q)", status, code, msg)
	}

	cause := errors.New("provider returned api key detail")
	err := xerr.Internalf(cause, "调用 %s 失败", "deepseek")
	if !errors.Is(err, cause) {
		t.Fatal("Internalf should preserve cause")
	}
	status, code, msg = xerr.ToHTTP(err)
	if status != 500 || code != 50000 || msg != "调用 deepseek 失败" {
		t.Fatalf("internal f = (%d,%d,%q)", status, code, msg)
	}
	if msg == cause.Error() {
		t.Fatal("Internalf leaked cause")
	}

	wrapped := xerr.Wrapf(cause, "解析模型配置 %s 失败", "cfg1")
	if !errors.Is(wrapped, cause) {
		t.Fatal("Wrapf should preserve cause")
	}
	_, _, msg = xerr.ToHTTP(wrapped)
	if msg != "解析模型配置 cfg1 失败" {
		t.Fatalf("wrapf message = %q", msg)
	}
}

func TestAppErrorCapturesCallLocation(t *testing.T) {
	err := xerr.BadRequest("参数错误")

	var appErr *xerr.AppError
	if !errors.As(err, &appErr) {
		t.Fatal("error should support errors.As AppError")
	}
	if appErr.Location == nil {
		t.Fatal("Location = nil")
	}
	if !strings.HasSuffix(appErr.Location.File, "internal/xerr/xerr_test.go") {
		t.Fatalf("Location.File = %q, want xerr_test.go caller", appErr.Location.File)
	}
	if strings.HasSuffix(appErr.Location.File, "internal/xerr/xerr.go") {
		t.Fatalf("Location.File points to xerr internals: %q", appErr.Location.File)
	}
	if appErr.Location.Line == 0 {
		t.Fatalf("Location.Line = 0")
	}
	if !strings.Contains(appErr.Location.Func, "TestAppErrorCapturesCallLocation") {
		t.Fatalf("Location.Func = %q, want test caller", appErr.Location.Func)
	}
}

func TestWrapfPreservesCauseAndCapturesLocation(t *testing.T) {
	cause := errors.New("boom")

	err := xerr.Wrapf(cause, "包装失败: %s", "case1")

	if !errors.Is(err, cause) {
		t.Fatal("Wrapf should preserve cause")
	}
	var appErr *xerr.AppError
	if !errors.As(err, &appErr) {
		t.Fatal("Wrapf should support errors.As AppError")
	}
	if appErr.Location == nil || !strings.Contains(appErr.Location.Func, "TestWrapfPreservesCauseAndCapturesLocation") {
		t.Fatalf("Location = %#v, want Wrapf caller", appErr.Location)
	}
}
