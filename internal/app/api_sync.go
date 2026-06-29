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
	Version   int                      `json:"version"`
	UpdatedAt time.Time                `json:"updatedAt"`
	Games     []services.CloudGameInfo `json:"games"`
}

// SyncStatus は指定ゲームの同期状態を返す。
func (app *App) SyncStatus(gameID string) result.ApiResult[domain.SyncStatusDetail] {
	trimmed, errResult, ok := requireGameID[domain.SyncStatusDetail](gameID)
	if !ok {
		return errResult
	}
	detail, err := app.ContentSyncService.Status(app.context(), trimmed)
	if err != nil {
		return serviceErrorResult[domain.SyncStatusDetail](err, "同期状態の取得に失敗しました")
	}
	return result.OkResult(detail)
}

// PushSync は指定ゲームのデータをリモートへアップロードする。
func (app *App) PushSync(gameID string) result.ApiResult[any] {
	trimmed, errResult, ok := requireGameID[any](gameID)
	if !ok {
		return errResult
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
// deleteUntracked=false で未追跡ファイルの削除が必要な場合、ダウンロードを行わず
// PullResult{Applied:false, UntrackedDeletes:...} を返す（呼び出し側で確認）。
func (app *App) PullSync(gameID string, deleteUntracked bool) result.ApiResult[domain.PullResult] {
	trimmed, errResult, ok := requireGameID[domain.PullResult](gameID)
	if !ok {
		return errResult
	}
	ctx := app.context()
	onProgress := func(current, total int) {
		wailsruntime.EventsEmit(ctx, "sync:progress", map[string]any{
			"operation": "pull",
			"current":   current,
			"total":     total,
		})
	}
	res, err := app.ContentSyncService.Pull(ctx, trimmed, onProgress, deleteUntracked)
	if err != nil {
		return serviceErrorResult[domain.PullResult](err, "ダウンロードに失敗しました")
	}
	return result.OkResult(res)
}

// ResolveConflict はコンフリクトを解決する。
// useLocal=false（リモート採用）は Pull と同様に未追跡ファイルの削除確認を経由する。
func (app *App) ResolveConflict(gameID string, useLocal, deleteUntracked bool) result.ApiResult[domain.PullResult] {
	trimmed, errResult, ok := requireGameID[domain.PullResult](gameID)
	if !ok {
		return errResult
	}
	res, err := app.ContentSyncService.ResolveConflict(app.context(), trimmed, useLocal, deleteUntracked)
	if err != nil {
		return serviceErrorResult[domain.PullResult](err, "コンフリクト解決に失敗しました")
	}
	return result.OkResult(res)
}

// syncGameAsync は指定ゲームのクラウド同期を非同期に要求する。
// 同一 gameID の同期は直列化され、実行中の再要求は完了後に1回だけ畳み込まれる。
func (app *App) syncGameAsync(gameID string) {
	if app.ContentSyncService == nil || app.syncCoalescer == nil {
		return
	}
	id := strings.TrimSpace(gameID)
	if id == "" {
		return
	}
	app.syncCoalescer.trigger(id)
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
	trimmed, errResult, ok := requireGameID[any](gameID)
	if !ok {
		return errResult
	}
	if err := app.ContentSyncService.DeleteFromCloud(app.context(), trimmed); err != nil {
		return serviceErrorResult[any](err, "クラウドデータ削除に失敗しました")
	}
	return result.OkResult[any](nil)
}
