// @fileoverview グローバルホットキーの共通定義。
package services

import "log/slog"

// HotkeyHandler はホットキー押下時の処理を受け取る。
type HotkeyHandler func()

// HotkeyConfig はホットキー設定を保持する。
type HotkeyConfig struct {
	Combo  string
	Notify bool
}

// HotkeyService はグローバルホットキーを管理する。
type HotkeyService interface {
	Start() error
	Stop()
}

// NewHotkeyService はプラットフォームに応じたホットキーサービスを生成する。
func NewHotkeyService(logger *slog.Logger, config HotkeyConfig, handler HotkeyHandler) HotkeyService {
	return newHotkeyService(logger, config, handler)
}
