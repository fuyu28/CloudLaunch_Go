// ルート管理のビジネスロジックを提供する。
package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"CloudLaunch_Go/internal/domain"
)

// RouteService はルート関連の操作を提供する。
type RouteService struct {
	repository RouteRepository
	logger     *slog.Logger
}

// NewRouteService は RouteService を生成する。
func NewRouteService(repository RouteRepository, logger *slog.Logger) *RouteService {
	return &RouteService{repository: repository, logger: logger}
}

// requireField は必須の文字列フィールドを検証し、空なら警告ログを出して ServiceError を返す。
func (service *RouteService) requireField(value, field, errMessage string) (string, error) {
	trimmed, detail, ok := requireNonEmpty(value, field)
	if !ok {
		service.logger.Warn(errMessage, "detail", detail, field, value)
		return "", newServiceError(errMessage, detail)
	}
	return trimmed, nil
}

// ListRoutesByGame はゲームIDでルート一覧を取得する。
func (service *RouteService) ListRoutesByGame(ctx context.Context, gameID string) ([]domain.Route, error) {
	routes, error := service.repository.ListRoutesByGame(ctx, strings.TrimSpace(gameID))
	if error != nil {
		service.logger.Error("ルート取得に失敗", "error", error)
		return nil, newServiceError("ルート取得に失敗しました", error.Error())
	}
	return routes, nil
}

// CreateRoute はルートを作成する。
func (service *RouteService) CreateRoute(ctx context.Context, input RouteInput) (*domain.Route, error) {
	if error := validateRouteInput(input); error != nil {
		service.logger.Warn("ルート入力が不正です", "error", error)
		return nil, newServiceError("ルート入力が不正です", error.Error())
	}

	route := domain.Route{
		Name:   strings.TrimSpace(input.Name),
		Order:  input.Order,
		GameID: strings.TrimSpace(input.GameID),
	}

	created, error := service.repository.CreateRoute(ctx, route)
	if error != nil {
		service.logger.Error("ルート作成に失敗", "error", error)
		return nil, newServiceError("ルート作成に失敗しました", error.Error())
	}
	return created, nil
}

// UpdateRoute はルートを更新する。
func (service *RouteService) UpdateRoute(ctx context.Context, routeID string, input RouteUpdateInput) (*domain.Route, error) {
	trimmedID, err := service.requireField(routeID, "routeID", "ルートIDが不正です")
	if err != nil {
		return nil, err
	}

	route, error := service.repository.GetRouteByID(ctx, trimmedID)
	if error != nil {
		service.logger.Error("ルート取得に失敗", "error", error)
		return nil, newServiceError("ルート取得に失敗しました", error.Error())
	}
	if route == nil {
		service.logger.Warn("ルートが見つかりません", "routeId", trimmedID)
		return nil, newServiceError("ルートが見つかりません", "指定されたIDが存在しません")
	}

	route.Name = strings.TrimSpace(input.Name)
	route.Order = input.Order

	updated, error := service.repository.UpdateRoute(ctx, *route)
	if error != nil {
		service.logger.Error("ルート更新に失敗", "error", error)
		return nil, newServiceError("ルート更新に失敗しました", error.Error())
	}
	return updated, nil
}

// DeleteRoute はルートを削除する。
func (service *RouteService) DeleteRoute(ctx context.Context, routeID string) error {
	trimmedID, err := service.requireField(routeID, "routeID", "ルートIDが不正です")
	if err != nil {
		return err
	}

	if error := service.repository.DeleteRoute(ctx, trimmedID); error != nil {
		service.logger.Error("ルート削除に失敗", "error", error)
		return newServiceError("ルート削除に失敗しました", error.Error())
	}
	return nil
}

// UpdateRouteOrders はルートの並び順を一括更新する。
func (service *RouteService) UpdateRouteOrders(ctx context.Context, gameID string, orders []RouteOrderUpdate) error {
	if _, err := service.requireField(gameID, "gameID", "ゲームIDが不正です"); err != nil {
		return err
	}
	items := make([]domain.RouteOrderItem, 0, len(orders))
	for _, order := range orders {
		if _, err := service.requireField(order.ID, "routeID", "ルートIDが不正です"); err != nil {
			return err
		}
		if order.Order < 0 {
			service.logger.Warn("ルート順序が不正です", "routeId", order.ID, "order", order.Order)
			return newServiceError("ルート順序が不正です", "orderが不正です")
		}
		items = append(items, domain.RouteOrderItem{ID: order.ID, Order: order.Order})
	}
	if err := service.repository.UpdateRouteOrders(ctx, items); err != nil {
		service.logger.Error("ルート順序更新に失敗", "error", err)
		return newServiceError("ルート順序更新に失敗しました", err.Error())
	}
	return nil
}

// GetRouteStats はルートの統計を取得する。
func (service *RouteService) GetRouteStats(ctx context.Context, gameID string) ([]domain.RouteStat, error) {
	trimmedGameID, err := service.requireField(gameID, "gameID", "ゲームIDが不正です")
	if err != nil {
		return nil, err
	}
	stats, error := service.repository.GetRouteStats(ctx, trimmedGameID)
	if error != nil {
		service.logger.Error("ルート統計取得に失敗", "error", error)
		return nil, newServiceError("ルート統計取得に失敗しました", error.Error())
	}
	return stats, nil
}

// SetCurrentRoute はゲームの現在ルートを設定する。
func (service *RouteService) SetCurrentRoute(ctx context.Context, gameID string, routeID string) error {
	trimmedGameID, err := service.requireField(gameID, "gameID", "ゲームIDが不正です")
	if err != nil {
		return err
	}
	trimmedRouteID, err := service.requireField(routeID, "routeID", "ルートIDが不正です")
	if err != nil {
		return err
	}
	game, error := service.repository.GetGameByID(ctx, trimmedGameID)
	if error != nil {
		service.logger.Error("ゲーム取得に失敗", "error", error)
		return newServiceError("ゲーム取得に失敗しました", error.Error())
	}
	if game == nil {
		service.logger.Warn("ゲームが見つかりません", "gameId", trimmedGameID)
		return newServiceError("ゲームが見つかりません", "指定されたIDが存在しません")
	}
	game.CurrentRouteID = &trimmedRouteID
	if _, error := service.repository.UpdateGame(ctx, *game); error != nil {
		service.logger.Error("現在ルート更新に失敗", "error", error)
		return newServiceError("現在ルート更新に失敗しました", error.Error())
	}
	return nil
}

// RouteInput はルート作成入力を表す。
type RouteInput struct {
	Name   string
	Order  int64
	GameID string
}

// RouteUpdateInput はルート更新入力を表す。
type RouteUpdateInput struct {
	Name  string
	Order int64
}

// RouteOrderUpdate はルート順序更新の入力を表す。
type RouteOrderUpdate struct {
	ID    string
	Order int64
}

// validateRouteInput はルート入力の基本チェックを行う。
func validateRouteInput(input RouteInput) error {
	if _, detail, ok := requireNonEmpty(input.Name, "name"); !ok {
		return errors.New(detail)
	}
	if _, detail, ok := requireNonEmpty(input.GameID, "gameID"); !ok {
		return errors.New(detail)
	}
	if input.Order < 0 {
		return errors.New("orderが不正です")
	}
	return nil
}
