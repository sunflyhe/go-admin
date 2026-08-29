// Package logger 提供基于 log/slog 的结构化 JSON 日志。
package logger

import (
	"log/slog"
	"os"
)

// New 初始化全局 slog Logger 并返回。level: debug|info|warn|error。
func New(level string) *slog.Logger {
	var lv slog.Level
	switch level {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv})
	l := slog.New(h)
	slog.SetDefault(l)
	return l
}
