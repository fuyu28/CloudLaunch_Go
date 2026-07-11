// goroutine 等での panic 回収とエラーログ記録を提供する。
package logging

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
)

// Recover は panic を回収し、スタックトレース付きで Error ログに残す。
// goroutine や定期処理で `defer logging.Recover(logger, "scope")` として使い、
// 想定外の panic でアプリ全体が落ちる/ログに残らないのを防ぐ。
// panic は再送出しないため、回収後は呼び出し元の処理（goroutine/ループ反復）が
// 正常終了したものとして継続する。
func Recover(logger *slog.Logger, scope string) {
	r := recover()
	if r == nil {
		return
	}
	stack := string(debug.Stack())
	if logger != nil {
		logger.Error("panic を回収しました", "scope", scope, "panic", fmt.Sprintf("%v", r), "stack", stack)
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "panic in %s: %v\n%s\n", scope, r, stack)
}
