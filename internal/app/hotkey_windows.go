//go:build windows

package app

import "CloudLaunch_Go/internal/services"

func (app *App) startHotkey() {
	if app.ScreenshotService == nil {
		return
	}
	config := services.HotkeyConfig{Combo: app.Config.ScreenshotHotkey}
	service := services.NewHotkeyService(app.Logger, config, app.handleHotkeyCapture)
	if service == nil {
		return
	}
	if err := service.Start(); err != nil {
		app.Logger.Warn("ホットキーを開始できませんでした", "error", err)
		return
	}
	app.HotkeyService = service
}

func (app *App) stopHotkey() {
	if app.HotkeyService == nil {
		return
	}
	app.HotkeyService.Stop()
	app.HotkeyService = nil
}

func (app *App) handleHotkeyCapture(target services.CaptureTarget) {
	app.Logger.Info("ホットキーを検知しました", "hwnd", target.HWND, "fallback", target.FromFallback)
}
