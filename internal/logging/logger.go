// @fileoverview slog を使ったログ初期化を提供する。
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger はログレベルに応じた slog.Logger を生成する。
func NewLogger(level string) *slog.Logger {
	logLevel := parseLevel(level)
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
}

// parseLevel は文字列から slog.Level を決定する。
func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
