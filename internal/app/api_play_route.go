package app

import (
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/services"
)

// CreatePlayRoute はプレイルートを作成する。
func (app *App) CreatePlayRoute(input services.PlayRouteInput) result.ApiResult[*models.PlayRoute] {
	route, err := app.PlayRouteService.CreatePlayRoute(app.context(), input)
	return serviceResult(route, err, "プレイルート作成に失敗しました")
}

// ListPlayRoutesByGame はゲーム配下のプレイルート一覧を取得する。
func (app *App) ListPlayRoutesByGame(gameID string) result.ApiResult[[]models.PlayRoute] {
	routes, err := app.PlayRouteService.ListPlayRoutesByGame(app.context(), gameID)
	return serviceResult(routes, err, "プレイルート取得に失敗しました")
}

// DeletePlayRoute はプレイルートを削除する。
func (app *App) DeletePlayRoute(routeID string) result.ApiResult[bool] {
	if err := app.PlayRouteService.DeletePlayRoute(app.context(), routeID); err != nil {
		return serviceErrorResult[bool](err, "プレイルート削除に失敗しました")
	}
	return result.OkResult(true)
}
