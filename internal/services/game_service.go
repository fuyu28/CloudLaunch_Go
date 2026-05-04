// @fileoverview ゲーム管理のビジネスロジックを提供する。
package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"CloudLaunch_Go/internal/models"
)

// GameService はゲーム関連の操作を提供する。
type GameService struct {
	repository GameRepository
	logger     *slog.Logger
}

// NewGameService は GameService を生成する。
func NewGameService(repository GameRepository, logger *slog.Logger) *GameService {
	return &GameService{repository: repository, logger: logger}
}

// ListGames は検索・フィルタ・ソート付きでゲーム一覧を取得する。
func (service *GameService) ListGames(
	ctx context.Context,
	searchText string,
	filter models.PlayStatus,
	sortBy string,
	sortDirection string,
) ([]models.Game, error) {
	games, error := service.repository.ListGames(ctx, strings.TrimSpace(searchText), filter, sortBy, sortDirection)
	if error != nil {
		service.logger.Error("ゲーム一覧取得に失敗", "error", error)
		return nil, newServiceError("ゲーム一覧取得に失敗しました", error.Error())
	}
	return games, nil
}

// GetGameByID はID指定でゲームを取得する。
func (service *GameService) GetGameByID(ctx context.Context, gameID string) (*models.Game, error) {
	game, error := service.repository.GetGameByID(ctx, strings.TrimSpace(gameID))
	if error != nil {
		service.logger.Error("ゲーム取得に失敗", "error", error)
		return nil, newServiceError("ゲーム取得に失敗しました", error.Error())
	}
	return game, nil
}

// CreateGame はゲームを新規作成する。
func (service *GameService) CreateGame(ctx context.Context, input GameInput) (*models.Game, error) {
	if error := validateGameInput(input); error != nil {
		service.logger.Warn("ゲーム入力が不正です", "error", error)
		return nil, newServiceError("ゲーム入力が不正です", error.Error())
	}

	game := models.Game{
		Title:          strings.TrimSpace(input.Title),
		Publisher:      strings.TrimSpace(input.Publisher),
		ImagePath:      input.ImagePath,
		ExePath:        strings.TrimSpace(input.ExePath),
		SaveFolderPath: input.SaveFolderPath,
		PlayStatus:     models.PlayStatusUnplayed,
		TotalPlayTime:  0,
		ClearedAt:      input.ClearedAt,
	}
	if input.ClearedAt != nil {
		game.PlayStatus = models.PlayStatusPlayed
	}

	created, error := service.repository.CreateGame(ctx, game)
	if error != nil {
		service.logger.Error("ゲーム作成に失敗", "error", error)
		return nil, newServiceError("ゲーム作成に失敗しました", error.Error())
	}

	service.logger.Info("ゲームを作成", "title", game.Title)
	return created, nil
}

// UpdateGame はゲーム情報を更新する。
func (service *GameService) UpdateGame(ctx context.Context, gameID string, input GameUpdateInput) (*models.Game, error) {
	trimmedID, detail, ok := requireNonEmpty(gameID, "gameID")
	if !ok {
		service.logger.Warn("ゲームIDが不正です", "detail", detail, "gameId", gameID)
		return nil, newServiceError("ゲームIDが不正です", detail)
	}

	current, error := service.repository.GetGameByID(ctx, trimmedID)
	if error != nil {
		service.logger.Error("ゲーム取得に失敗", "error", error)
		return nil, newServiceError("ゲーム取得に失敗しました", error.Error())
	}
	if current == nil {
		service.logger.Warn("ゲームが見つかりません", "gameId", trimmedID)
		return nil, newServiceError("ゲームが見つかりません", "指定されたIDが存在しません")
	}

	current.Title = strings.TrimSpace(input.Title)
	current.Publisher = strings.TrimSpace(input.Publisher)
	current.ImagePath = input.ImagePath
	current.ExePath = strings.TrimSpace(input.ExePath)
	current.SaveFolderPath = input.SaveFolderPath
	current.ClearedAt = input.ClearedAt

	updated, error := service.repository.UpdateGame(ctx, *current)
	if error != nil {
		service.logger.Error("ゲーム更新に失敗", "error", error)
		return nil, newServiceError("ゲーム更新に失敗しました", error.Error())
	}
	return updated, nil
}

// UpdatePlayTime はプレイ時間と最終プレイ日時を更新する。
func (service *GameService) UpdatePlayTime(ctx context.Context, gameID string, totalPlayTime int64, lastPlayed time.Time) (*models.Game, error) {
	trimmedID, detail, ok := requireNonEmpty(gameID, "gameID")
	if !ok {
		service.logger.Warn("ゲームIDが不正です", "detail", detail, "gameId", gameID)
		return nil, newServiceError("ゲームIDが不正です", detail)
	}

	current, error := service.repository.GetGameByID(ctx, trimmedID)
	if error != nil {
		service.logger.Error("ゲーム取得に失敗", "error", error)
		return nil, newServiceError("ゲーム取得に失敗しました", error.Error())
	}
	if current == nil {
		service.logger.Warn("ゲームが見つかりません", "gameId", trimmedID)
		return nil, newServiceError("ゲームが見つかりません", "指定されたIDが存在しません")
	}

	current.TotalPlayTime = totalPlayTime
	current.LastPlayed = &lastPlayed

	updated, error := service.repository.UpdateGame(ctx, *current)
	if error != nil {
		service.logger.Error("プレイ時間更新に失敗", "error", error)
		return nil, newServiceError("プレイ時間更新に失敗しました", error.Error())
	}
	return updated, nil
}

// DeleteGame はゲームを削除する。
func (service *GameService) DeleteGame(ctx context.Context, gameID string) error {
	trimmedID, detail, ok := requireNonEmpty(gameID, "gameID")
	if !ok {
		service.logger.Warn("ゲームIDが不正です", "detail", detail, "gameId", gameID)
		return newServiceError("ゲームIDが不正です", detail)
	}

	if error := service.repository.DeleteGame(ctx, trimmedID); error != nil {
		service.logger.Error("ゲーム削除に失敗", "error", error)
		return newServiceError("ゲーム削除に失敗しました", error.Error())
	}
	return nil
}

// GameInput はゲーム作成入力を表す。
type GameInput struct {
	Title          string
	Publisher      string
	ImagePath      *string
	ExePath        string
	SaveFolderPath *string
	ClearedAt      *time.Time
}

// GameUpdateInput はゲーム更新入力を表す。
type GameUpdateInput struct {
	Title          string
	Publisher      string
	ImagePath      *string
	ExePath        string
	SaveFolderPath *string
	ClearedAt      *time.Time
}

// validateGameInput はゲーム作成入力の簡易検証を行う。
func validateGameInput(input GameInput) error {
	if _, detail, ok := requireNonEmpty(input.Title, "title"); !ok {
		return errors.New(detail)
	}
	if _, detail, ok := requireNonEmpty(input.Publisher, "publisher"); !ok {
		return errors.New(detail)
	}
	if _, detail, ok := requireNonEmpty(input.ExePath, "exePath"); !ok {
		return errors.New(detail)
	}
	return nil
}
