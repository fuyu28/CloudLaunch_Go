// コンテンツアドレッシング同期関連の API を提供する。
package app

import (
	"strings"
	"time"

	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/services"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// CloudMetadataResult はクラウドメタ情報の API レスポンス。
type CloudMetadataResult struct {
	Version   int                    `json:"version"`
	UpdatedAt time.Time              `json:"updatedAt"`
	Games     []services.CloudGameInfo `json:"games"`
}

// SyncStatus は指定ゲームの同期状態を返す。
func (app *App) SyncStatus(gameID string) result.ApiResult[domain.SyncStatusDetail] {
	trimmed := strings.TrimSpace(gameID)
	if trimmed == "" {
		return result.ErrorResult[domain.SyncStatusDetail]("ゲームIDが不正です", "gameID is empty")
	}
	detail, err := app.ContentSyncService.Status(app.context(), trimmed)
	if err != nil {
		return serviceErrorResult[domain.SyncStatusDetail](err, "同期状態の取得に失敗しました")
	}
	return result.OkResult(detail)
}

// PushSync は指定ゲームのデータをリモートへアップロードする。
func (app *App) PushSync(gameID string) result.ApiResult[any] {
	trimmed := strings.TrimSpace(gameID)
	if trimmed == "" {
		return result.ErrorResult[any]("ゲームIDが不正です", "gameID is empty")
	}
	ctx := app.context()
	onProgress := func(current, total int) {
		wailsruntime.EventsEmit(ctx, "sync:progress", map[string]any{
			"operation": "push",
			"current":   current,
			"total":     total,
		})
	}
	if err := app.ContentSyncService.Push(ctx, trimmed, onProgress); err != nil {
		return serviceErrorResult[any](err, "アップロードに失敗しました")
	}
	return result.OkResult[any](nil)
}

// PullSync は指定ゲームのデータをリモートからダウンロードする。
func (app *App) PullSync(gameID string) result.ApiResult[any] {
	trimmed := strings.TrimSpace(gameID)
	if trimmed == "" {
		return result.ErrorResult[any]("ゲームIDが不正です", "gameID is empty")
	}
	ctx := app.context()
	onProgress := func(current, total int) {
		wailsruntime.EventsEmit(ctx, "sync:progress", map[string]any{
			"operation": "pull",
			"current":   current,
			"total":     total,
		})
	}
	if err := app.ContentSyncService.Pull(ctx, trimmed, onProgress); err != nil {
		return serviceErrorResult[any](err, "ダウンロードに失敗しました")
	}
	return result.OkResult[any](nil)
}

// ResolveConflict はコンフリクトを解決する。
func (app *App) ResolveConflict(gameID string, useLocal bool) result.ApiResult[any] {
	trimmed := strings.TrimSpace(gameID)
	if trimmed == "" {
		return result.ErrorResult[any]("ゲームIDが不正です", "gameID is empty")
	}
	if err := app.ContentSyncService.ResolveConflict(app.context(), trimmed, useLocal); err != nil {
		return serviceErrorResult[any](err, "コンフリクト解決に失敗しました")
	}
	return result.OkResult[any](nil)
}

func (app *App) syncGameAsync(gameID string) {
	if app.ContentSyncService == nil || strings.TrimSpace(gameID) == "" {
		return
	}
	go func(id string) {
		if err := app.ContentSyncService.Push(app.context(), id, nil); err != nil {
			app.Logger.Warn("クラウド同期に失敗", "gameId", id, "detail", err)
		}
	}(gameID)
}

// LoadCloudMetadata はクラウド上の全ゲームメタ情報を返す。
func (app *App) LoadCloudMetadata() result.ApiResult[CloudMetadataResult] {
	games, err := app.ContentSyncService.LoadCloudMetadata(app.context())
	if err != nil {
		return serviceErrorResult[CloudMetadataResult](err, "クラウドメタ情報の取得に失敗しました")
	}
	if games == nil {
		games = []services.CloudGameInfo{}
	}
	return result.OkResult(CloudMetadataResult{
		Version:   1,
		UpdatedAt: time.Now().UTC(),
		Games:     games,
	})
}

// DeleteGameFromCloud は指定ゲームのクラウドデータを削除する。
func (app *App) DeleteGameFromCloud(gameID string) result.ApiResult[any] {
	trimmed := strings.TrimSpace(gameID)
	if trimmed == "" {
		return result.ErrorResult[any]("ゲームIDが不正です", "gameID is empty")
	}
	if err := app.ContentSyncService.DeleteFromCloud(app.context(), trimmed); err != nil {
		return serviceErrorResult[any](err, "クラウドデータ削除に失敗しました")
	}
	return result.OkResult[any](nil)
}
