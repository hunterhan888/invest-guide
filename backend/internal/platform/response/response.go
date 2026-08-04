package response

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 领域 sentinel 错误 — 任何 service/repository 返回的错误都通过 errors.Is 匹配这些
var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrNotFound       = errors.New("not found")
	ErrConflict       = errors.New("conflict")
	ErrRateLimited    = errors.New("rate limited")
	ErrInternal       = errors.New("internal error")
	ErrBadGateway     = errors.New("bad gateway")
	ErrGatewayTimeout = errors.New("gateway timeout")
)

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    string `json:"code"`
}

func Ok(c *gin.Context, status int, data interface{}) {
	c.JSON(status, SuccessResponse{Success: true, Data: data})
}

func OkWithMessage(c *gin.Context, status int, data interface{}, message string) {
	c.JSON(status, SuccessResponse{Success: true, Data: data, Message: message})
}

func Fail(c *gin.Context, err error) {
	status, code := mapError(err)
	msg := err.Error()
	if status >= 500 {
		slog.Error("request failed", "err", err)
		msg = "internal error"
	}
	c.JSON(status, ErrorResponse{Success: false, Error: msg, Code: code})
}

func mapError(err error) (status int, code string) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest, "INVALID_INPUT"
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized, "UNAUTHORIZED"
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden, "FORBIDDEN"
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, ErrConflict):
		return http.StatusConflict, "CONFLICT"
	case errors.Is(err, ErrRateLimited):
		return http.StatusTooManyRequests, "RATE_LIMITED"
	case errors.Is(err, ErrBadGateway):
		return http.StatusBadGateway, "BAD_GATEWAY"
	case errors.Is(err, ErrGatewayTimeout):
		return http.StatusGatewayTimeout, "GATEWAY_TIMEOUT"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR"
	}
}
