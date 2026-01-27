// @fileoverview メモ管理のビジネスロジックを提供する。
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

// MemoService はメモ関連の操作を提供する。
type MemoService struct {
	repository *db.Repository
	logger     *slog.Logger
}

// NewMemoService は MemoService を生成する。
func NewMemoService(repository *db.Repository, logger *slog.Logger) *MemoService {
	return &MemoService{repository: repository, logger: logger}
}

// CreateMemo はメモを作成する。
func (service *MemoService) CreateMemo(ctx context.Context, input MemoInput) result.ApiResult[*models.Memo] {
	if error := validateMemoInput(input); error != nil {
		return result.ErrorResult[*models.Memo]("メモ入力が不正です", error.Error())
	}

	memo := models.Memo{
		Title:   strings.TrimSpace(input.Title),
		Content: input.Content,
		GameID:  strings.TrimSpace(input.GameID),
	}

	created, error := service.repository.CreateMemo(ctx, memo)
	if error != nil {
		service.logger.Error("メモ作成に失敗", "error", error)
		return result.ErrorResult[*models.Memo]("メモ作成に失敗しました", error.Error())
	}
	return result.OkResult(created)
}

// UpdateMemo はメモを更新する。
func (service *MemoService) UpdateMemo(ctx context.Context, memoID string, input MemoUpdateInput) result.ApiResult[*models.Memo] {
	if strings.TrimSpace(memoID) == "" {
		return result.ErrorResult[*models.Memo]("メモIDが不正です", "memoIDが空です")
	}

	memo, error := service.repository.GetMemoByID(ctx, strings.TrimSpace(memoID))
	if error != nil {
		service.logger.Error("メモ取得に失敗", "error", error)
		return result.ErrorResult[*models.Memo]("メモ取得に失敗しました", error.Error())
	}
	if memo == nil {
		return result.ErrorResult[*models.Memo]("メモが見つかりません", "指定されたIDが存在しません")
	}

	memo.Title = strings.TrimSpace(input.Title)
	memo.Content = input.Content

	updated, error := service.repository.UpdateMemo(ctx, *memo)
	if error != nil {
		service.logger.Error("メモ更新に失敗", "error", error)
		return result.ErrorResult[*models.Memo]("メモ更新に失敗しました", error.Error())
	}
	return result.OkResult(updated)
}

// GetMemoByID はメモIDでメモを取得する。
func (service *MemoService) GetMemoByID(ctx context.Context, memoID string) result.ApiResult[*models.Memo] {
	if strings.TrimSpace(memoID) == "" {
		return result.ErrorResult[*models.Memo]("メモIDが不正です", "memoIDが空です")
	}

	memo, error := service.repository.GetMemoByID(ctx, strings.TrimSpace(memoID))
	if error != nil {
		service.logger.Error("メモ取得に失敗", "error", error)
		return result.ErrorResult[*models.Memo]("メモ取得に失敗しました", error.Error())
	}
	return result.OkResult(memo)
}

// ListMemosByGame はゲームIDでメモ一覧を取得する。
func (service *MemoService) ListMemosByGame(ctx context.Context, gameID string) result.ApiResult[[]models.Memo] {
	memos, error := service.repository.ListMemosByGame(ctx, strings.TrimSpace(gameID))
	if error != nil {
		service.logger.Error("メモ取得に失敗", "error", error)
		return result.ErrorResult[[]models.Memo]("メモ取得に失敗しました", error.Error())
	}
	return result.OkResult(memos)
}

// ListAllMemos は全メモを取得する。
func (service *MemoService) ListAllMemos(ctx context.Context) result.ApiResult[[]models.Memo] {
	memos, error := service.repository.ListAllMemos(ctx)
	if error != nil {
		service.logger.Error("メモ取得に失敗", "error", error)
		return result.ErrorResult[[]models.Memo]("メモ取得に失敗しました", error.Error())
	}
	return result.OkResult(memos)
}

// DeleteMemo はメモを削除する。
func (service *MemoService) DeleteMemo(ctx context.Context, memoID string) result.ApiResult[bool] {
	if strings.TrimSpace(memoID) == "" {
		return result.ErrorResult[bool]("メモIDが不正です", "memoIDが空です")
	}

	if error := service.repository.DeleteMemo(ctx, strings.TrimSpace(memoID)); error != nil {
		service.logger.Error("メモ削除に失敗", "error", error)
		return result.ErrorResult[bool]("メモ削除に失敗しました", error.Error())
	}
	return result.OkResult(true)
}

// MemoInput はメモ作成入力を表す。
type MemoInput struct {
	Title   string
	Content string
	GameID  string
}

// MemoUpdateInput はメモ更新入力を表す。
type MemoUpdateInput struct {
	Title   string
	Content string
}

// validateMemoInput はメモ入力の基本チェックを行う。
func validateMemoInput(input MemoInput) error {
	if strings.TrimSpace(input.Title) == "" {
		return errors.New("titleが空です")
	}
	if strings.TrimSpace(input.Content) == "" {
		return errors.New("contentが空です")
	}
	if strings.TrimSpace(input.GameID) == "" {
		return errors.New("gameIDが空です")
	}
	return nil
}
