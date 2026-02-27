// @fileoverview slog を使ったログ初期化を提供する。
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	logDirName  = "logs"
	logFileName = "app.log"
)

// NewLogger はログレベルに応じた slog.Logger を生成する。
// 標準出力に加えて appDataDir/logs/app.log へも同時出力する。
func NewLogger(appDataDir string, level string) *slog.Logger {
	logLevel := ParseLevel(level)
	output := io.Writer(os.Stdout)

	if logFile, err := openLogFile(appDataDir); err == nil {
		output = io.MultiWriter(os.Stdout, logFile)
	} else if strings.TrimSpace(appDataDir) != "" {
		_, _ = fmt.Fprintf(os.Stderr, "failed to initialize log file: %v\n", err)
	}

	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	})
	return slog.New(handler).With("scope", "backend")
}

// ParseLevel は文字列から slog.Level を決定する。
func ParseLevel(level string) slog.Level {
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

func openLogFile(appDataDir string) (*os.File, error) {
	baseDir := strings.TrimSpace(appDataDir)
	if baseDir == "" {
		return nil, fmt.Errorf("appDataDir is empty")
	}

	logDir := filepath.Join(baseDir, logDirName)
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return nil, err
	}

	logPath := filepath.Join(logDir, logFileName)
	return os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
}
