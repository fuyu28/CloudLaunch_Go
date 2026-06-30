// @fileoverview 批評空間からのゲーム情報取得APIを提供する。
package app

import (
	"errors"
	"strings"

	"CloudLaunch_Go/internal/domain"
)

// FetchFromErogameScape は批評空間URLからゲーム情報を取得する。
func (app *App) FetchFromErogameScape(gamePageURL string) (domain.GameImport, error) {
	if app.ErogameScapeService == nil {
		app.Logger.Error("批評空間サービスが未初期化です", "operation", "FetchFromErogameScape")
		return domain.GameImport{}, errors.New("ErogameScapeService is not initialized")
	}
	result, err := app.ErogameScapeService.FetchFromErogameScape(app.context(), gamePageURL)
	if err != nil {
		app.Logger.Error("批評空間からの取得に失敗しました", "operation", "FetchFromErogameScape", "url", strings.TrimSpace(gamePageURL), "error", err)
		return domain.GameImport{}, err
	}
	return result, nil
}

// SearchErogameScape は批評空間の検索結果を取得する。
func (app *App) SearchErogameScape(query string, pageURL string) (domain.ErogameScapeSearchResult, error) {
	if app.ErogameScapeService == nil {
		app.Logger.Error("批評空間サービスが未初期化です", "operation", "SearchErogameScape")
		return domain.ErogameScapeSearchResult{}, errors.New("ErogameScapeService is not initialized")
	}
	result, err := app.ErogameScapeService.SearchErogameScape(app.context(), query, pageURL)
	if err != nil {
		app.Logger.Error("批評空間検索に失敗しました", "operation", "SearchErogameScape", "query", strings.TrimSpace(query), "pageUrl", strings.TrimSpace(pageURL), "error", err)
		return domain.ErogameScapeSearchResult{}, err
	}
	return result, nil
}
