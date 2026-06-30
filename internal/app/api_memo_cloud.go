// @fileoverview メモのクラウド同期APIを提供する。
package app

import (
	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/services"
)

// GetCloudMemos はクラウドメモ一覧を取得する。
func (app *App) GetCloudMemos() result.ApiResult[[]services.CloudMemoInfo] {
	memos, err := app.MemoCloudService.GetCloudMemos(app.context())
	return serviceResult(memos, err, "クラウドメモ取得に失敗しました")
}

// DownloadMemoFromCloud はクラウドからメモ内容を取得する。
func (app *App) DownloadMemoFromCloud(gameID string, memoFileName string) result.ApiResult[string] {
	content, err := app.MemoCloudService.DownloadMemoFromCloud(app.context(), gameID, memoFileName)
	return serviceResult(content, err, "メモのダウンロードに失敗しました")
}

// UploadMemoToCloud はメモをクラウドへ保存する。
func (app *App) UploadMemoToCloud(memoID string) result.ApiResult[bool] {
	if err := app.MemoCloudService.UploadMemoToCloud(app.context(), memoID); err != nil {
		return serviceErrorResult[bool](err, "メモのアップロードに失敗しました")
	}
	return result.OkResult(true)
}

// SyncMemosFromCloud はメモをクラウドと同期する。
func (app *App) SyncMemosFromCloud(gameID string) result.ApiResult[services.MemoSyncResult] {
	summary, err := app.MemoCloudService.SyncMemosFromCloud(app.context(), gameID)
	return serviceResult(summary, err, "メモ同期に失敗しました")
}
