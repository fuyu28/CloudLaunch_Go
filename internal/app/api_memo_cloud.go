// @fileoverview メモのクラウド同期APIを提供する。
package app

import (
	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/services"
)

// GetCloudMemos はクラウドメモ一覧を取得する。
func (app *App) GetCloudMemos() result.ApiResult[[]services.CloudMemoInfo] {
	return app.MemoCloudService.GetCloudMemos(app.context())
}

// DownloadMemoFromCloud はクラウドからメモ内容を取得する。
func (app *App) DownloadMemoFromCloud(gameID string, memoFileName string) result.ApiResult[string] {
	return app.MemoCloudService.DownloadMemoFromCloud(app.context(), gameID, memoFileName)
}

// UploadMemoToCloud はメモをクラウドへ保存する。
func (app *App) UploadMemoToCloud(memoID string) result.ApiResult[bool] {
	return app.MemoCloudService.UploadMemoToCloud(app.context(), memoID)
}

// SyncMemosFromCloud はメモをクラウドと同期する。
func (app *App) SyncMemosFromCloud(gameID string) result.ApiResult[services.MemoSyncResult] {
	return app.MemoCloudService.SyncMemosFromCloud(app.context(), gameID)
}
