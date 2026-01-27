// @fileoverview ゲーム管理のビジネスロジックを提供する。
package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"CloudLaunch_Go/internal/db"
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/result"
)

// GameService はゲーム関連の操作を提供する。
type GameService struct {
	repository *db.Repository
	logger     *slog.Logger
}

// NewGameService は GameService を生成する。
func NewGameService(repository *db.Repository, logger *slog.Logger) *GameService {
	return &GameService{repository: repository, logger: logger}
}

// ListGames は検索・フィルタ・ソート付きでゲーム一覧を取得する。
func (service *GameService) ListGames(
	ctx context.Context,
	searchText string,
	filter models.PlayStatus,
	sortBy string,
	sortDirection string,
) result.ApiResult[[]models.Game] {
	games, error := service.repository.ListGames(ctx, strings.TrimSpace(searchText), filter, sortBy, sortDirection)
	if error != nil {
		service.logger.Error("ゲーム一覧取得に失敗", "error", error)
		return result.ErrorResult[[]models.Game]("ゲーム一覧取得に失敗しました", error.Error())
	}
	return result.OkResult(games)
}

// GetGameByID はID指定でゲームを取得する。
func (service *GameService) GetGameByID(ctx context.Context, gameID string) result.ApiResult[*models.Game] {
	game, error := service.repository.GetGameByID(ctx, strings.TrimSpace(gameID))
	if error != nil {
		service.logger.Error("ゲーム取得に失敗", "error", error)
		return result.ErrorResult[*models.Game]("ゲーム取得に失敗しました", error.Error())
	}
	return result.OkResult(game)
}

// CreateGame はゲームを新規作成する。
func (service *GameService) CreateGame(ctx context.Context, input GameInput) result.ApiResult[*models.Game] {
	if error := validateGameInput(input); error != nil {
		return result.ErrorResult[*models.Game]("ゲーム入力が不正です", error.Error())
	}

	game := models.Game{
		Title:          strings.TrimSpace(input.Title),
		Publisher:      strings.TrimSpace(input.Publisher),
		ImagePath:      input.ImagePath,
		ExePath:        strings.TrimSpace(input.ExePath),
		SaveFolderPath: input.SaveFolderPath,
		PlayStatus:     models.PlayStatusUnplayed,
		TotalPlayTime:  0,
	}

	created, error := service.repository.CreateGame(ctx, game)
	if error != nil {
		service.logger.Error("ゲーム作成に失敗", "error", error)
		return result.ErrorResult[*models.Game]("ゲーム作成に失敗しました", error.Error())
	}

	if created != nil {
		_, _ = service.repository.CreateChapter(ctx, models.Chapter{
			Name:   "第1章",
			Order:  1,
			GameID: created.ID,
		})
	}

	service.logger.Info("ゲームを作成", "title", game.Title)
	return result.OkResult(created)
}

// UpdateGame はゲーム情報を更新する。
func (service *GameService) UpdateGame(ctx context.Context, gameID string, input GameUpdateInput) result.ApiResult[*models.Game] {
	if strings.TrimSpace(gameID) == "" {
		return result.ErrorResult[*models.Game]("ゲームIDが不正です", "gameIDが空です")
	}

	current, error := service.repository.GetGameByID(ctx, strings.TrimSpace(gameID))
	if error != nil {
		service.logger.Error("ゲーム取得に失敗", "error", error)
		return result.ErrorResult[*models.Game]("ゲーム取得に失敗しました", error.Error())
	}
	if current == nil {
		return result.ErrorResult[*models.Game]("ゲームが見つかりません", "指定されたIDが存在しません")
	}

	current.Title = strings.TrimSpace(input.Title)
	current.Publisher = strings.TrimSpace(input.Publisher)
	current.ImagePath = input.ImagePath
	current.ExePath = strings.TrimSpace(input.ExePath)
	current.SaveFolderPath = input.SaveFolderPath
	if input.PlayStatus != "" {
		current.PlayStatus = input.PlayStatus
	}
	current.ClearedAt = input.ClearedAt
	current.CurrentChapter = input.CurrentChapter

	updated, error := service.repository.UpdateGame(ctx, *current)
	if error != nil {
		service.logger.Error("ゲーム更新に失敗", "error", error)
		return result.ErrorResult[*models.Game]("ゲーム更新に失敗しました", error.Error())
	}
	return result.OkResult(updated)
}

// UpdatePlayTime はプレイ時間と最終プレイ日時を更新する。
func (service *GameService) UpdatePlayTime(ctx context.Context, gameID string, totalPlayTime int64, lastPlayed time.Time) result.ApiResult[*models.Game] {
	current, error := service.repository.GetGameByID(ctx, strings.TrimSpace(gameID))
	if error != nil {
		service.logger.Error("ゲーム取得に失敗", "error", error)
		return result.ErrorResult[*models.Game]("ゲーム取得に失敗しました", error.Error())
	}
	if current == nil {
		return result.ErrorResult[*models.Game]("ゲームが見つかりません", "指定されたIDが存在しません")
	}

	current.TotalPlayTime = totalPlayTime
	current.LastPlayed = &lastPlayed

	updated, error := service.repository.UpdateGame(ctx, *current)
	if error != nil {
		service.logger.Error("プレイ時間更新に失敗", "error", error)
		return result.ErrorResult[*models.Game]("プレイ時間更新に失敗しました", error.Error())
	}
	return result.OkResult(updated)
}

// DeleteGame はゲームを削除する。
func (service *GameService) DeleteGame(ctx context.Context, gameID string) result.ApiResult[bool] {
	if strings.TrimSpace(gameID) == "" {
		return result.ErrorResult[bool]("ゲームIDが不正です", "gameIDが空です")
	}

	if error := service.repository.DeleteGame(ctx, strings.TrimSpace(gameID)); error != nil {
		service.logger.Error("ゲーム削除に失敗", "error", error)
		return result.ErrorResult[bool]("ゲーム削除に失敗しました", error.Error())
	}
	return result.OkResult(true)
}

// GameInput はゲーム作成入力を表す。
type GameInput struct {
	Title          string
	Publisher      string
	ImagePath      *string
	ExePath        string
	SaveFolderPath *string
}

// GameUpdateInput はゲーム更新入力を表す。
type GameUpdateInput struct {
	Title          string
	Publisher      string
	ImagePath      *string
	ExePath        string
	SaveFolderPath *string
	PlayStatus     models.PlayStatus
	ClearedAt      *time.Time
	CurrentChapter *string
}

// validateGameInput はゲーム作成入力の簡易検証を行う。
func validateGameInput(input GameInput) error {
	if strings.TrimSpace(input.Title) == "" {
		return errors.New("titleが空です")
	}
	if strings.TrimSpace(input.Publisher) == "" {
		return errors.New("publisherが空です")
	}
	if strings.TrimSpace(input.ExePath) == "" {
		return errors.New("exePathが空です")
	}
	return nil
}
