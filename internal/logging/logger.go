// slog を使ったログ初期化を提供する。
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	logDirName    = "logs"
	logFileName   = "app.log"
	errorFileName = "error.log"

	// maxLogSize は1ファイルあたりの上限サイズ。超えるとローテーションする。
	maxLogSize = 5 * 1024 * 1024 // 5MB
	// maxLogBackups は保持する世代数（app.log.1 .. app.log.N）。
	maxLogBackups = 3
)

// NewLogger はログレベルに応じた slog.Logger を生成する。
// 標準出力に加えて appDataDir/logs/app.log へ全レベルを、appDataDir/logs/error.log へ
// error 以上を同時出力する。各ファイルはサイズ上限でローテーションする。
func NewLogger(appDataDir string, level string) *slog.Logger {
	logLevel := ParseLevel(level)

	var mainWriter io.Writer = os.Stdout
	var errorHandler slog.Handler

	logDir, dirErr := ensureLogDir(appDataDir)
	if dirErr == nil {
		if appWriter, err := newRotatingWriter(filepath.Join(logDir, logFileName), maxLogSize, maxLogBackups); err == nil {
			mainWriter = io.MultiWriter(os.Stdout, appWriter)
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "failed to initialize log file: %v\n", err)
		}
		// error 以上だけを集約する専用ファイル。重大なエラーを探しやすくする。
		if errWriter, err := newRotatingWriter(filepath.Join(logDir, errorFileName), maxLogSize, maxLogBackups); err == nil {
			errorHandler = slog.NewJSONHandler(errWriter, &slog.HandlerOptions{Level: slog.LevelError, AddSource: true})
		} else {
			_, _ = fmt.Fprintf(os.Stderr, "failed to initialize error log file: %v\n", err)
		}
	} else if strings.TrimSpace(appDataDir) != "" {
		_, _ = fmt.Fprintf(os.Stderr, "failed to initialize log dir: %v\n", dirErr)
	}

	baseHandler := slog.NewJSONHandler(mainWriter, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	})
	var handler slog.Handler = baseHandler
	if errorHandler != nil {
		handler = &teeErrorHandler{base: baseHandler, errorH: errorHandler}
	}
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

func ensureLogDir(appDataDir string) (string, error) {
	baseDir := strings.TrimSpace(appDataDir)
	if baseDir == "" {
		return "", fmt.Errorf("appDataDir is empty")
	}
	logDir := filepath.Join(baseDir, logDirName)
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return "", err
	}
	return logDir, nil
}

// teeErrorHandler は全レコードを base へ、error 以上を errorH へも転送する slog.Handler。
type teeErrorHandler struct {
	base   slog.Handler
	errorH slog.Handler
}

func (h *teeErrorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

func (h *teeErrorHandler) Handle(ctx context.Context, record slog.Record) error {
	err := h.base.Handle(ctx, record)
	if record.Level >= slog.LevelError {
		if e := h.errorH.Handle(ctx, record.Clone()); e != nil && err == nil {
			err = e
		}
	}
	return err
}

func (h *teeErrorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &teeErrorHandler{base: h.base.WithAttrs(attrs), errorH: h.errorH.WithAttrs(attrs)}
}

func (h *teeErrorHandler) WithGroup(name string) slog.Handler {
	return &teeErrorHandler{base: h.base.WithGroup(name), errorH: h.errorH.WithGroup(name)}
}

// rotatingWriter はサイズ上限を超えたらローテーションする並行安全な io.Writer。
type rotatingWriter struct {
	mu         sync.Mutex
	path       string
	maxSize    int64
	maxBackups int
	file       *os.File
	size       int64
}

func newRotatingWriter(path string, maxSize int64, maxBackups int) (*rotatingWriter, error) {
	w := &rotatingWriter{path: path, maxSize: maxSize, maxBackups: maxBackups}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

// open は既存ファイルを追記モードで開き、現在のサイズを把握する。
func (w *rotatingWriter) open() error {
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	w.file = f
	w.size = info.Size()
	return nil
}

func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return 0, fmt.Errorf("log writer is closed")
	}
	if w.size+int64(len(p)) > w.maxSize {
		// ローテーションに失敗しても既存ファイルへの書き込みは継続を試みる。
		_ = w.rotate()
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

// rotate は現在のファイルを path.1 .. path.N へシフトし、新しい空ファイルを開く。
func (w *rotatingWriter) rotate() error {
	_ = w.file.Close()
	w.file = nil
	for i := w.maxBackups; i >= 1; i-- {
		src := w.backupPath(i - 1)
		dst := w.backupPath(i)
		if _, err := os.Stat(src); err == nil {
			_ = os.Rename(src, dst)
		}
	}
	return w.open()
}

func (w *rotatingWriter) backupPath(i int) string {
	if i == 0 {
		return w.path
	}
	return fmt.Sprintf("%s.%d", w.path, i)
}
