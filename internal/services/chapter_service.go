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
	if strings.TrimSpace(chapterID) == "" {
		return result.ErrorResult[*models.Chapter]("章IDが不正です", "chapterIDが空です")
	}

	chapter, error := service.repository.GetChapterByID(ctx, strings.TrimSpace(chapterID))
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
	if strings.TrimSpace(chapterID) == "" {
		return result.ErrorResult[bool]("章IDが不正です", "chapterIDが空です")
	}

	if error := service.repository.DeleteChapter(ctx, strings.TrimSpace(chapterID)); error != nil {
		service.logger.Error("章削除に失敗", "error", error)
		return result.ErrorResult[bool]("章削除に失敗しました", error.Error())
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

// validateChapterInput は章入力の基本チェックを行う。
func validateChapterInput(input ChapterInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("nameが空です")
	}
	if strings.TrimSpace(input.GameID) == "" {
		return errors.New("gameIDが空です")
	}
	if input.Order < 0 {
		return errors.New("orderが不正です")
	}
	return nil
}
