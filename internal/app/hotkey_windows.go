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
	app.HotkeyService = app.startHotkeyService(app.Config.ScreenshotHotkey, app.handleHotkeyCapture)
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

func (app *App) handleHotkeyCapture() bool {
	if app.ScreenshotService == nil {
		return false
	}
	gameID, path, err := app.ScreenshotService.CaptureHotkey(app.context())
	if err != nil {
		if err == services.ErrNoNewScreenshot {
			return false
		}
		app.Logger.Error("ホットキーキャプチャに失敗", "error", err)
		return false
	}
	app.syncScreenshotAfterHotkey(gameID, path)
	return true
}

func (app *App) syncScreenshotAfterHotkey(gameID string, path string) {
	if strings.TrimSpace(gameID) == "" {
		return
	}
	if app.Config.ScreenshotSyncEnabled {
		if syncErr := app.uploadScreenshot(app.context(), gameID, path); syncErr != nil {
			app.Logger.Error("スクリーンショット同期に失敗", "error", syncErr)
		}
	}
}
