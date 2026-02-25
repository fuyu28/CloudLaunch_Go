//go:build windows

package app

import (
	"strings"

	"CloudLaunch_Go/internal/services"
)

func (app *App) startHotkey() {
	if app.ScreenshotService == nil {
		return
	}
	app.HotkeyService = app.startHotkeyService(app.Config.ScreenshotHotkey, app.handleHotkeyCaptureClient)
}

func (app *App) stopHotkey() {
	if app.HotkeyService == nil {
		return
	}
	app.HotkeyService.Stop()
	app.HotkeyService = nil
}

func (app *App) startHotkeyService(combo string, handler services.HotkeyHandler) services.HotkeyService {
	trimmed := strings.TrimSpace(combo)
	if trimmed == "" {
		return nil
	}
	config := services.HotkeyConfig{
		Combo:  trimmed,
		Notify: app.Config.ScreenshotHotkeyNotify,
	}
	service := services.NewHotkeyService(app.Logger, config, handler)
	if service == nil {
		return nil
	}
	if err := service.Start(); err != nil {
		app.Logger.Warn("ホットキーを開始できませんでした", "combo", trimmed, "error", err)
		return nil
	}
	return service
}

func (app *App) handleHotkeyCaptureClient(target services.CaptureTarget) {
	if app.ScreenshotService == nil {
		return
	}
	gameID, path, err := app.ScreenshotService.CaptureForegroundWindow(app.context(), target)
	if err != nil {
		if err == services.ErrNoNewScreenshot {
			return
		}
		app.Logger.Error("ホットキーキャプチャに失敗", "mode", "client", "error", err)
		return
	}
	app.syncScreenshotAfterHotkey(gameID, path)
}

func (app *App) syncScreenshotAfterHotkey(gameID string, path string) {
	if app.Config.ScreenshotSyncEnabled {
		if syncErr := app.uploadScreenshot(app.context(), gameID, path); syncErr != nil {
			app.Logger.Error("スクリーンショット同期に失敗", "error", syncErr)
		}
	}
}
