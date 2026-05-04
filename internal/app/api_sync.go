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
	summary, err := app.CloudSyncService.SyncAllGames(app.context(), "default")
	if err != nil {
		app.Logger.Warn("クラウド同期が失敗", "operation", "SyncAllGames", "detail", err)
		return serviceErrorResult[services.CloudSyncSummary](err, "クラウド同期に失敗しました")
	}
	app.Logger.Info("クラウド同期が完了", "operation", "SyncAllGames", "summary", summary)
	return result.OkResult(summary)
}

// SyncGame は指定ゲームのクラウド同期を行う。
func (app *App) SyncGame(gameID string) result.ApiResult[services.CloudSyncSummary] {
	if app.CloudSyncService == nil {
		app.Logger.Error("同期機能が利用できません", "operation", "SyncGame", "reason", "CloudSyncService is nil")
		return result.ErrorResult[services.CloudSyncSummary]("同期機能が利用できません", "CloudSyncServiceが未初期化です")
	}
	trimmedID := strings.TrimSpace(gameID)
	app.Logger.Info("ゲーム単位クラウド同期を開始", "operation", "SyncGame", "gameId", trimmedID)
	summary, err := app.CloudSyncService.SyncGame(app.context(), "default", gameID)
	if err != nil {
		app.Logger.Warn("ゲーム単位クラウド同期が失敗", "operation", "SyncGame", "gameId", trimmedID, "detail", err)
		return serviceErrorResult[services.CloudSyncSummary](err, "クラウド同期に失敗しました")
	}
	app.Logger.Info("ゲーム単位クラウド同期が完了", "operation", "SyncGame", "gameId", trimmedID, "summary", summary)
	return result.OkResult(summary)
}

// DeleteCloudGame は指定ゲームのクラウドデータを削除する。
func (app *App) DeleteCloudGame(gameID string) result.ApiResult[bool] {
	if app.CloudSyncService == nil {
		app.Logger.Error("削除機能が利用できません", "operation", "DeleteCloudGame", "reason", "CloudSyncService is nil")
		return result.ErrorResult[bool]("削除機能が利用できません", "CloudSyncServiceが未初期化です")
	}
	trimmedID := strings.TrimSpace(gameID)
	app.Logger.Info("クラウドゲーム削除を開始", "operation", "DeleteCloudGame", "gameId", trimmedID)
	if err := app.CloudSyncService.DeleteGameFromCloud(app.context(), "default", gameID); err != nil {
		app.Logger.Warn("クラウドゲーム削除が失敗", "operation", "DeleteCloudGame", "gameId", trimmedID, "detail", err)
		return serviceErrorResult[bool](err, "クラウドゲーム削除に失敗しました")
	}
	return result.OkResult(true)
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
		if _, err := app.CloudSyncService.SyncGame(app.context(), "default", targetID); err != nil {
			app.Logger.Warn("クラウド同期に失敗", "gameId", targetID, "detail", err)
		}
	}(gameID)
}
