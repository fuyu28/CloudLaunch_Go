// @fileoverview メモファイル関連APIを提供する。
package app

import (
	"CloudLaunch_Go/internal/memo"
	"CloudLaunch_Go/internal/result"
)

// GetMemoRootDir はメモのルートディレクトリを返す。
func (app *App) GetMemoRootDir() result.ApiResult[string] {
	manager := app.memoManager()
	return result.OkResult(manager.RootDir())
}

// GetMemoFilePath はメモIDからファイルパスを推定する。
func (app *App) GetMemoFilePath(memoID string) result.ApiResult[string] {
	memoData, error := app.Database.GetMemoByID(app.context(), memoID)
	if error != nil {
		return result.ErrorResult[string]("メモ取得に失敗しました", error.Error())
	}
	if memoData == nil {
		return result.ErrorResult[string]("メモが見つかりません", "指定されたIDが存在しません")
	}
	manager := app.memoManager()
	return result.OkResult(manager.MemoFilePath(memoData.GameID, memoData.ID, memoData.Title))
}

// GetGameMemoDir はゲームのメモディレクトリを返す。
func (app *App) GetGameMemoDir(gameID string) result.ApiResult[string] {
	manager := app.memoManager()
	return result.OkResult(manager.GameDir(gameID))
}

func (app *App) memoManager() *memo.FileManager {
	if app.MemoFiles != nil {
		return app.MemoFiles
	}
	return memo.NewFileManager(app.Config.AppDataDir)
}
