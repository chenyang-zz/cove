package xerr

import (
	"errors"
	"fmt"
	"net/http"
)

type Kind string

const (
	KindBadRequest   Kind = "bad_request"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindInternal     Kind = "internal"
	KindUnavailable  Kind = "unavailable"
)

type AppError struct {
	Kind        Kind
	Code        int
	Message     string
	SafeMessage string
	HTTPStatus  int
	Cause       error
	Fields      map[string]any
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	msg := e.Message
	if msg == "" {
		msg = e.SafeMessage
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", msg, e.Cause)
	}
	return msg
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *AppError) WithField(key string, value any) *AppError {
	if e.Fields == nil {
		e.Fields = map[string]any{}
	}
	e.Fields[key] = value
	return e
}

func New(kind Kind, safeMessage string) error {
	return newAppError(kind, defaultCodeForKind(kind), safeMessage, nil)
}

func Wrap(cause error, safeMessage string) error {
	return newAppError(KindInternal, defaultCodeForKind(KindInternal), safeMessage, cause)
}

func Wrapf(cause error, format string, args ...any) error {
	return Wrap(cause, fmt.Sprintf(format, args...))
}

func newAppError(kind Kind, code int, safeMessage string, cause error) error {
	status := statusForKind(kind)
	if code == 0 {
		code = defaultCodeForKind(kind)
	}
	if safeMessage == "" {
		safeMessage = defaultMessageForKind(kind)
	}
	return &AppError{
		Kind:        kind,
		Code:        code,
		Message:     safeMessage,
		SafeMessage: safeMessage,
		HTTPStatus:  status,
		Cause:       cause,
	}
}

func BadRequest(safeMessage string) error {
	return newAppError(KindBadRequest, defaultCodeForKind(KindBadRequest), safeMessage, nil)
}

func BadRequestf(format string, args ...any) error {
	return BadRequest(fmt.Sprintf(format, args...))
}

func Unauthorized(safeMessage string) error {
	return newAppError(KindUnauthorized, defaultCodeForKind(KindUnauthorized), safeMessage, nil)
}

func Unauthorizedf(format string, args ...any) error {
	return Unauthorized(fmt.Sprintf(format, args...))
}

func Forbidden(safeMessage string) error {
	return newAppError(KindForbidden, defaultCodeForKind(KindForbidden), safeMessage, nil)
}

func Forbiddenf(format string, args ...any) error {
	return Forbidden(fmt.Sprintf(format, args...))
}

func NotFound(safeMessage string) error {
	return newAppError(KindNotFound, defaultCodeForKind(KindNotFound), safeMessage, nil)
}

func NotFoundf(format string, args ...any) error {
	return NotFound(fmt.Sprintf(format, args...))
}

func Conflict(safeMessage string) error {
	return newAppError(KindConflict, defaultCodeForKind(KindConflict), safeMessage, nil)
}

func Conflictf(format string, args ...any) error {
	return Conflict(fmt.Sprintf(format, args...))
}

func Internal(safeMessage string, cause error) error {
	return newAppError(KindInternal, defaultCodeForKind(KindInternal), safeMessage, cause)
}

func Internalf(cause error, format string, args ...any) error {
	return Internal(fmt.Sprintf(format, args...), cause)
}

func Unavailable(safeMessage string) error {
	return newAppError(KindUnavailable, defaultCodeForKind(KindUnavailable), safeMessage, nil)
}

func Unavailablef(format string, args ...any) error {
	return Unavailable(fmt.Sprintf(format, args...))
}

func Validation(cause error) error {
	return newAppError(KindBadRequest, 40001, "请求参数错误", cause)
}

func UserExists() error {
	return newAppError(KindConflict, 40901, "用户已存在", nil)
}

func InvalidCredential() error {
	return newAppError(KindUnauthorized, 40103, "邮箱或密码错误", nil)
}

func InvalidToken() error {
	return newAppError(KindUnauthorized, 40102, "登录已失效", nil)
}

func From(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return &AppError{
		Kind:        KindInternal,
		Code:        defaultCodeForKind(KindInternal),
		Message:     defaultMessageForKind(KindInternal),
		SafeMessage: defaultMessageForKind(KindInternal),
		HTTPStatus:  http.StatusInternalServerError,
		Cause:       err,
	}
}

func ToHTTP(err error) (status int, code int, message string) {
	appErr := From(err)
	if appErr == nil {
		return http.StatusOK, 0, "ok"
	}
	status = appErr.HTTPStatus
	if status == 0 {
		status = statusForKind(appErr.Kind)
	}
	code = appErr.Code
	if code == 0 {
		code = defaultCodeForKind(appErr.Kind)
	}
	message = appErr.SafeMessage
	if message == "" {
		message = defaultMessageForKind(appErr.Kind)
	}
	return status, code, message
}

func statusForKind(kind Kind) int {
	switch kind {
	case KindBadRequest:
		return http.StatusBadRequest
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict:
		return http.StatusConflict
	case KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func defaultCodeForKind(kind Kind) int {
	switch kind {
	case KindBadRequest:
		return 40000
	case KindUnauthorized:
		return 40100
	case KindForbidden:
		return 40300
	case KindNotFound:
		return 40400
	case KindConflict:
		return 40900
	case KindUnavailable:
		return 50300
	default:
		return 50000
	}
}

func defaultMessageForKind(kind Kind) string {
	switch kind {
	case KindBadRequest:
		return "请求参数错误"
	case KindUnauthorized:
		return "请先登录"
	case KindForbidden:
		return "没有权限"
	case KindNotFound:
		return "资源不存在"
	case KindConflict:
		return "资源冲突"
	case KindUnavailable:
		return "服务暂时不可用"
	default:
		return "系统开小差了"
	}
}
