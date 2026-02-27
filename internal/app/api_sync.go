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
		app.Logger.Error("同期機能が利用できません", "operation", "SyncAllGames", "reason", "CloudSyncService is nil")
		return result.ErrorResult[services.CloudSyncSummary]("同期機能が利用できません", "CloudSyncServiceが未初期化です")
	}
	app.Logger.Info("クラウド同期を開始", "operation", "SyncAllGames")
	syncResult := app.CloudSyncService.SyncAllGames(app.context(), "default")
	if syncResult.Success {
		app.Logger.Info("クラウド同期が完了", "operation", "SyncAllGames", "summary", syncResult.Data)
	} else {
		app.Logger.Warn("クラウド同期が失敗", "operation", "SyncAllGames", "detail", syncResult.Error)
	}
	return syncResult
}

// SyncGame は指定ゲームのクラウド同期を行う。
func (app *App) SyncGame(gameID string) result.ApiResult[services.CloudSyncSummary] {
	if app.CloudSyncService == nil {
		app.Logger.Error("同期機能が利用できません", "operation", "SyncGame", "reason", "CloudSyncService is nil")
		return result.ErrorResult[services.CloudSyncSummary]("同期機能が利用できません", "CloudSyncServiceが未初期化です")
	}
	trimmedID := strings.TrimSpace(gameID)
	app.Logger.Info("ゲーム単位クラウド同期を開始", "operation", "SyncGame", "gameId", trimmedID)
	syncResult := app.CloudSyncService.SyncGame(app.context(), "default", gameID)
	if syncResult.Success {
		app.Logger.Info("ゲーム単位クラウド同期が完了", "operation", "SyncGame", "gameId", trimmedID, "summary", syncResult.Data)
	} else {
		app.Logger.Warn("ゲーム単位クラウド同期が失敗", "operation", "SyncGame", "gameId", trimmedID, "detail", syncResult.Error)
	}
	return syncResult
}

// DeleteCloudGame は指定ゲームのクラウドデータを削除する。
func (app *App) DeleteCloudGame(gameID string) result.ApiResult[bool] {
	if app.CloudSyncService == nil {
		app.Logger.Error("削除機能が利用できません", "operation", "DeleteCloudGame", "reason", "CloudSyncService is nil")
		return result.ErrorResult[bool]("削除機能が利用できません", "CloudSyncServiceが未初期化です")
	}
	trimmedID := strings.TrimSpace(gameID)
	app.Logger.Info("クラウドゲーム削除を開始", "operation", "DeleteCloudGame", "gameId", trimmedID)
	deleteResult := app.CloudSyncService.DeleteGameFromCloud(app.context(), "default", gameID)
	if !deleteResult.Success {
		app.Logger.Warn("クラウドゲーム削除が失敗", "operation", "DeleteCloudGame", "gameId", trimmedID, "detail", deleteResult.Error)
	}
	return deleteResult
}

// UpdateOfflineMode はオフラインモードを更新する。
func (app *App) UpdateOfflineMode(enabled bool) result.ApiResult[bool] {
	if app.CloudSyncService != nil {
		app.CloudSyncService.SetOfflineMode(enabled)
	}
	app.Logger.Info("オフラインモードを更新", "enabled", enabled)
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
