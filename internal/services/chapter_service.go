// @fileoverview 章管理のビジネスロジックを提供する。
package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"CloudLaunch_Go/internal/db"
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/result"
)

// ChapterService は章関連の操作を提供する。
type ChapterService struct {
	repository *db.Repository
	logger     *slog.Logger
}

// NewChapterService は ChapterService を生成する。
func NewChapterService(repository *db.Repository, logger *slog.Logger) *ChapterService {
	return &ChapterService{repository: repository, logger: logger}
}

// ListChaptersByGame はゲームIDで章一覧を取得する。
func (service *ChapterService) ListChaptersByGame(ctx context.Context, gameID string) result.ApiResult[[]models.Chapter] {
	chapters, error := service.repository.ListChaptersByGame(ctx, strings.TrimSpace(gameID))
	if error != nil {
		service.logger.Error("章取得に失敗", "error", error)
		return result.ErrorResult[[]models.Chapter]("章取得に失敗しました", error.Error())
	}
	return result.OkResult(chapters)
}

// CreateChapter は章を作成する。
func (service *ChapterService) CreateChapter(ctx context.Context, input ChapterInput) result.ApiResult[*models.Chapter] {
	if error := validateChapterInput(input); error != nil {
		return result.ErrorResult[*models.Chapter]("章入力が不正です", error.Error())
	}

	chapter := models.Chapter{
		Name:   strings.TrimSpace(input.Name),
		Order:  input.Order,
		GameID: strings.TrimSpace(input.GameID),
	}

	created, error := service.repository.CreateChapter(ctx, chapter)
	if error != nil {
		service.logger.Error("章作成に失敗", "error", error)
		return result.ErrorResult[*models.Chapter]("章作成に失敗しました", error.Error())
	}
	return result.OkResult(created)
}

// UpdateChapter は章を更新する。
func (service *ChapterService) UpdateChapter(ctx context.Context, chapterID string, input ChapterUpdateInput) result.ApiResult[*models.Chapter] {
	trimmedID, detail, ok := requireNonEmpty(chapterID, "chapterID")
	if !ok {
		return result.ErrorResult[*models.Chapter]("章IDが不正です", detail)
	}

	chapter, error := service.repository.GetChapterByID(ctx, trimmedID)
	if error != nil {
		service.logger.Error("章取得に失敗", "error", error)
		return result.ErrorResult[*models.Chapter]("章取得に失敗しました", error.Error())
	}
	if chapter == nil {
		return result.ErrorResult[*models.Chapter]("章が見つかりません", "指定されたIDが存在しません")
	}

	chapter.Name = strings.TrimSpace(input.Name)
	chapter.Order = input.Order

	updated, error := service.repository.UpdateChapter(ctx, *chapter)
	if error != nil {
		service.logger.Error("章更新に失敗", "error", error)
		return result.ErrorResult[*models.Chapter]("章更新に失敗しました", error.Error())
	}
	return result.OkResult(updated)
}

// DeleteChapter は章を削除する。
func (service *ChapterService) DeleteChapter(ctx context.Context, chapterID string) result.ApiResult[bool] {
	trimmedID, detail, ok := requireNonEmpty(chapterID, "chapterID")
	if !ok {
		return result.ErrorResult[bool]("章IDが不正です", detail)
	}

	if error := service.repository.DeleteChapter(ctx, trimmedID); error != nil {
		service.logger.Error("章削除に失敗", "error", error)
		return result.ErrorResult[bool]("章削除に失敗しました", error.Error())
	}
	return result.OkResult(true)
}

// UpdateChapterOrders は章の並び順を更新する。
func (service *ChapterService) UpdateChapterOrders(ctx context.Context, gameID string, orders []ChapterOrderUpdate) result.ApiResult[bool] {
	trimmedGameID, detail, ok := requireNonEmpty(gameID, "gameID")
	if !ok {
		return result.ErrorResult[bool]("ゲームIDが不正です", detail)
	}
	for _, order := range orders {
		if _, detail, ok := requireNonEmpty(order.ID, "chapterID"); !ok {
			return result.ErrorResult[bool]("章IDが不正です", detail)
		}
		if order.Order < 0 {
			return result.ErrorResult[bool]("章順序が不正です", "orderが不正です")
		}
		if error := service.repository.UpdateChapterOrder(ctx, order.ID, order.Order); error != nil {
			service.logger.Error("章順序更新に失敗", "error", error)
			return result.ErrorResult[bool]("章順序更新に失敗しました", error.Error())
		}
	}
	return result.OkResult(true)
}

// GetChapterStats は章の統計を取得する。
func (service *ChapterService) GetChapterStats(ctx context.Context, gameID string) result.ApiResult[[]models.ChapterStat] {
	trimmedGameID, detail, ok := requireNonEmpty(gameID, "gameID")
	if !ok {
		return result.ErrorResult[[]models.ChapterStat]("ゲームIDが不正です", detail)
	}
	stats, error := service.repository.GetChapterStats(ctx, trimmedGameID)
	if error != nil {
		service.logger.Error("章統計取得に失敗", "error", error)
		return result.ErrorResult[[]models.ChapterStat]("章統計取得に失敗しました", error.Error())
	}
	return result.OkResult(stats)
}

// SetCurrentChapter はゲームの現在章を設定する。
func (service *ChapterService) SetCurrentChapter(ctx context.Context, gameID string, chapterID string) result.ApiResult[bool] {
	trimmedGameID, detail, ok := requireNonEmpty(gameID, "gameID")
	if !ok {
		return result.ErrorResult[bool]("ゲームIDが不正です", detail)
	}
	trimmedChapterID, detail, ok := requireNonEmpty(chapterID, "chapterID")
	if !ok {
		return result.ErrorResult[bool]("章IDが不正です", detail)
	}
	game, error := service.repository.GetGameByID(ctx, trimmedGameID)
	if error != nil {
		service.logger.Error("ゲーム取得に失敗", "error", error)
		return result.ErrorResult[bool]("ゲーム取得に失敗しました", error.Error())
	}
	if game == nil {
		return result.ErrorResult[bool]("ゲームが見つかりません", "指定されたIDが存在しません")
	}
	game.CurrentChapter = &trimmedChapterID
	if _, error := service.repository.UpdateGame(ctx, *game); error != nil {
		service.logger.Error("現在章更新に失敗", "error", error)
		return result.ErrorResult[bool]("現在章更新に失敗しました", error.Error())
	}
	return result.OkResult(true)
}

// ChapterInput は章作成入力を表す。
type ChapterInput struct {
	Name   string
	Order  int64
	GameID string
}

// ChapterUpdateInput は章更新入力を表す。
type ChapterUpdateInput struct {
	Name  string
	Order int64
}

// ChapterOrderUpdate は章順序更新の入力を表す。
type ChapterOrderUpdate struct {
	ID    string
	Order int64
}

// validateChapterInput は章入力の基本チェックを行う。
func validateChapterInput(input ChapterInput) error {
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
