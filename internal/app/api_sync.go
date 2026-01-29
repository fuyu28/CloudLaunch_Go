// @fileoverview クラウド同期関連のAPIを提供する。
package app

import (
	"strings"

	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/services"
)

// SyncAllGames は全ゲームのクラウド同期を行う。
func (app *App) SyncAllGames() result.ApiResult[services.CloudSyncSummary] {
	if app.CloudSyncService == nil {
		return result.ErrorResult[services.CloudSyncSummary]("同期機能が利用できません", "CloudSyncServiceが未初期化です")
	}
	return app.CloudSyncService.SyncAllGames(app.context(), "default")
}

// SyncGame は指定ゲームのクラウド同期を行う。
func (app *App) SyncGame(gameID string) result.ApiResult[services.CloudSyncSummary] {
	if app.CloudSyncService == nil {
		return result.ErrorResult[services.CloudSyncSummary]("同期機能が利用できません", "CloudSyncServiceが未初期化です")
	}
	return app.CloudSyncService.SyncGame(app.context(), "default", gameID)
}

// UpdateOfflineMode はオフラインモードを更新する。
func (app *App) UpdateOfflineMode(enabled bool) result.ApiResult[bool] {
	if app.CloudSyncService != nil {
		app.CloudSyncService.SetOfflineMode(enabled)
	}
	return result.OkResult(true)
}

func (app *App) syncGameAsync(gameID string) {
	if app.CloudSyncService == nil || strings.TrimSpace(gameID) == "" {
		return
	}
	go func(targetID string) {
		result := app.CloudSyncService.SyncGame(app.context(), "default", targetID)
		if !result.Success {
			app.Logger.Warn("クラウド同期に失敗", "gameId", targetID, "detail", result.Error)
		}
	}(gameID)
}
