package services

import (
	"context"
	"log/slog"
	"strings"

	"CloudLaunch_Go/internal/models"
)

// PlayRouteService はプレイルート関連操作を提供する。
type PlayRouteService struct {
	repository PlayRouteRepository
	logger     *slog.Logger
}

// NewPlayRouteService は PlayRouteService を生成する。
func NewPlayRouteService(repository PlayRouteRepository, logger *slog.Logger) *PlayRouteService {
	return &PlayRouteService{repository: repository, logger: logger}
}

// PlayRouteInput はプレイルート作成入力を表す。
type PlayRouteInput struct {
	GameID    string
	Name      string
	SortOrder int
}

// CreatePlayRoute はプレイルートを新規作成する。
func (service *PlayRouteService) CreatePlayRoute(ctx context.Context, input PlayRouteInput) (*models.PlayRoute, error) {
	gameID, detail, ok := requireNonEmpty(input.GameID, "gameID")
	if !ok {
		service.logger.Warn("ゲームIDが不正です", "detail", detail, "gameId", input.GameID)
		return nil, newServiceError("ゲームIDが不正です", detail)
	}
	name, detail, ok := requireNonEmpty(input.Name, "name")
	if !ok {
		service.logger.Warn("ルート名が不正です", "detail", detail, "name", input.Name)
		return nil, newServiceError("ルート名が不正です", detail)
	}
	if input.SortOrder < 0 {
		service.logger.Warn("表示順が不正です", "sortOrder", input.SortOrder)
		return nil, newServiceError("表示順が不正です", "sortOrderが負です")
	}

	created, err := service.repository.CreatePlayRoute(ctx, models.PlayRoute{
		GameID:    gameID,
		Name:      name,
		SortOrder: input.SortOrder,
	})
	if err != nil {
		service.logger.Error("プレイルート作成に失敗", "error", err)
		return nil, newServiceError("プレイルート作成に失敗しました", err.Error())
	}
	return created, nil
}

// ListPlayRoutesByGame はゲーム配下のプレイルート一覧を取得する。
func (service *PlayRouteService) ListPlayRoutesByGame(ctx context.Context, gameID string) ([]models.PlayRoute, error) {
	trimmed := strings.TrimSpace(gameID)
	routes, err := service.repository.ListPlayRoutesByGame(ctx, trimmed)
	if err != nil {
		service.logger.Error("プレイルート取得に失敗", "error", err)
		return nil, newServiceError("プレイルート取得に失敗しました", err.Error())
	}
	return routes, nil
}

// DeletePlayRoute はプレイルートを削除する。
func (service *PlayRouteService) DeletePlayRoute(ctx context.Context, routeID string) error {
	trimmedID, detail, ok := requireNonEmpty(routeID, "routeID")
	if !ok {
		service.logger.Warn("プレイルートIDが不正です", "detail", detail, "routeId", routeID)
		return newServiceError("プレイルートIDが不正です", detail)
	}
	if err := service.repository.DeletePlayRoute(ctx, trimmedID); err != nil {
		service.logger.Error("プレイルート削除に失敗", "error", err)
		return newServiceError("プレイルート削除に失敗しました", err.Error())
	}
	return nil
}
