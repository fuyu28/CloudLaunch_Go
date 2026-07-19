package logging

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLoggerWritesAppAndErrorFiles(t *testing.T) {
	dir := t.TempDir()
	logger, _, closer := NewLogger(dir, "info")
	t.Cleanup(func() {
		_ = closer.Close()
	})

	logger.Info("情報ログ", "k", "v")
	logger.Error("重大エラー", "k", "v")

	appLog := readFile(t, filepath.Join(dir, logDirName, logFileName))
	errLog := readFile(t, filepath.Join(dir, logDirName, errorFileName))

	// app.log には info と error の両方が出る。
	if !strings.Contains(appLog, "情報ログ") || !strings.Contains(appLog, "重大エラー") {
		t.Fatalf("app.log に想定のログがない: %q", appLog)
	}
	// error.log には error のみ。
	if strings.Contains(errLog, "情報ログ") {
		t.Fatalf("error.log に info ログが混入している: %q", errLog)
	}
	if !strings.Contains(errLog, "重大エラー") {
		t.Fatalf("error.log に error ログがない: %q", errLog)
	}
}

func TestRotatingWriterRotates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	w, err := newRotatingWriter(path, 100, 2)
	if err != nil {
		t.Fatalf("newRotatingWriter: %v", err)
	}
	t.Cleanup(func() {
		_ = w.Close()
	})

	chunk := strings.Repeat("a", 60) + "\n"
	for i := 0; i < 5; i++ {
		if _, err := w.Write([]byte(chunk)); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	// 上限 100 を超えるたびにローテーションするので、バックアップが生成される。
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("現行ログが存在しない: %v", err)
	}
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("バックアップ .1 が生成されていない: %v", err)
	}
	// maxBackups=2 を超える世代は残らない。
	if _, err := os.Stat(path + ".3"); err == nil {
		t.Fatalf("maxBackups を超える世代が残っている")
	}
}

func TestTeeErrorHandlerEnabledMatchesBase(t *testing.T) {
	base := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	errH := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})
	h := &teeErrorHandler{base: base, errorH: errH}

	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatalf("info が有効になっていない")
	}
	if h.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatalf("debug が無効になっていない")
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
