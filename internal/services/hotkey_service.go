// @fileoverview グローバルホットキーの共通定義。
package services

import "log/slog"

// CaptureTarget はホットキー押下時に撮影対象となるウィンドウ情報を保持する。
type CaptureTarget struct {
	HWND         uintptr
	FromFallback bool
}

// HotkeyHandler はホットキー押下時の処理を受け取る。
type HotkeyHandler func(CaptureTarget)

// HotkeyConfig はホットキー設定を保持する。
type HotkeyConfig struct {
	Combo string
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
