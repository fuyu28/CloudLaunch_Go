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
	if app.ScreenshotService == nil {
		return
	}
	gameID, path, err := app.ScreenshotService.CaptureForegroundWindow(app.context(), target)
	if err != nil {
		app.Logger.Error("ホットキーキャプチャに失敗", "error", err)
		return
	}
	if app.Config.ScreenshotSyncEnabled {
		if syncErr := app.uploadScreenshot(app.context(), gameID, path); syncErr != nil {
			app.Logger.Error("スクリーンショット同期に失敗", "error", syncErr)
		}
	}
}
