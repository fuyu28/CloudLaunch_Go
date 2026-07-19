//go:build !windows

// 非 Windows 向けのプロセス列挙スタブ。空一覧と source "unsupported" を返す（警告は出さない）。
package services

import "log/slog"

// defaultProcessProvider は未対応 OS では列挙しない。空スナップショットは正常系であり、
// 監視ループごとに Warn/Error を出さない（Windows の fallback 失敗ログと区別する）。
func defaultProcessProvider(logger *slog.Logger) func() ([]ProcessInfo, string) {
	_ = logger
	return func() ([]ProcessInfo, string) {
		return []ProcessInfo{}, "unsupported"
	}
}
