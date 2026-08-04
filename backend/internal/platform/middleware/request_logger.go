package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger 输出每个请求的结构化日志（slog JSON），含 request_id、method、path、
// status、耗时与客户端 IP。置于 RequestID 之后、Recovery 之前，便于按 request_id 关联
// 同一请求的完整链路。SSE 流式端点会阻塞到流结束，duration_ms 反映完整流式耗时。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}

		slog.Log(c.Request.Context(), level, "request",
			"request_id", c.GetString("request_id"),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
		)
	}
}
