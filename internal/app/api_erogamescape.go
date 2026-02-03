// @fileoverview 批評空間からのゲーム情報取得APIを提供する。
package app

import (
	"errors"

	"CloudLaunch_Go/internal/models"
)

// FetchFromErogameScape は批評空間URLからゲーム情報を取得する。
func (app *App) FetchFromErogameScape(gamePageURL string) (models.GameImport, error) {
	if app.ErogameScapeService == nil {
		return models.GameImport{}, errors.New("ErogameScapeService is not initialized")
	}
	return app.ErogameScapeService.FetchFromErogameScape(app.context(), gamePageURL)
}

// SearchErogameScape は批評空間の検索結果を取得する。
func (app *App) SearchErogameScape(query string, pageURL string) (models.ErogameScapeSearchResult, error) {
	if app.ErogameScapeService == nil {
		return models.ErogameScapeSearchResult{}, errors.New("ErogameScapeService is not initialized")
	}
	return app.ErogameScapeService.SearchErogameScape(app.context(), query, pageURL)
}
