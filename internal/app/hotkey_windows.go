//go:build windows

package app

import (
	"errors"
	"strings"

	"CloudLaunch_Go/internal/services"
)

func (app *App) startHotkey() error {
	if app.ScreenshotService == nil {
		return nil
	}
	service, err := app.startHotkeyService(app.Config.ScreenshotHotkey, app.handleHotkeyCapture)
	if err != nil {
		app.HotkeyService = nil
		return err
	}
	app.HotkeyService = service
	return nil
}

func (app *App) stopHotkey() {
	if app.HotkeyService == nil {
		return
	}
	app.HotkeyService.Stop()
	app.HotkeyService = nil
}

func (app *App) startHotkeyService(combo string, handler services.HotkeyHandler) (services.HotkeyService, error) {
	trimmed := strings.TrimSpace(combo)
	if trimmed == "" {
		return nil, errors.New("hotkey is not configured")
	}
	config := services.HotkeyConfig{
		Combo:  trimmed,
		Notify: app.Config.ScreenshotHotkeyNotify,
	}
	service := services.NewHotkeyService(app.Logger, config, handler)
	if service == nil {
		return nil, errors.New("failed to create hotkey service")
	}
	if err := service.Start(); err != nil {
		app.Logger.Warn("ホットキーを開始できませんでした", "combo", trimmed, "error", err)
		return nil, err
	}
	return service, nil
}

func (app *App) handleHotkeyCapture() bool {
	if app.ScreenshotService == nil {
		return false
	}
	hotkeyTargetGameID := ""
	if app.ProcessMonitor != nil {
		hotkeyTargetGameID = app.ProcessMonitor.GetHotkeyTargetGameID()
	}
	gameID, path, err := app.ScreenshotService.CaptureHotkey(app.context(), hotkeyTargetGameID)
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
