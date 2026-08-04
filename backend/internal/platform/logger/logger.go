package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type ctxKey struct{}

var defaultLogger *slog.Logger

// OpenFile 打开（必要时创建）日志文件，追加模式；自动创建父目录。
func OpenFile(path string) (io.WriteCloser, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
}

// Init 初始化全局 JSON logger。level: debug/info/warn/error
func Init(w io.Writer, level string) {
	lvl := slog.LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}
	defaultLogger = slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl}))
	slog.SetDefault(defaultLogger)
}

// WithRequestID 把 request_id 写入 context
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext 取出可携带 request_id 的 logger；无 ctx 时回退默认 logger
func FromContext(ctx context.Context) *slog.Logger {
	if id, ok := ctx.Value(ctxKey{}).(string); ok && id != "" {
		return defaultLogger.With("request_id", id)
	}
	return defaultLogger
}
