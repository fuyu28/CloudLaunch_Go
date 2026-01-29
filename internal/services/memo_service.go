// @fileoverview メモ管理のビジネスロジックを提供する。
package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"CloudLaunch_Go/internal/db"
	"CloudLaunch_Go/internal/memo"
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/result"
)

// MemoService はメモ関連の操作を提供する。
type MemoService struct {
	repository  *db.Repository
	fileManager *memo.FileManager
	logger      *slog.Logger
}

// NewMemoService は MemoService を生成する。
func NewMemoService(repository *db.Repository, fileManager *memo.FileManager, logger *slog.Logger) *MemoService {
	return &MemoService{repository: repository, fileManager: fileManager, logger: logger}
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

	if service.fileManager != nil {
		if _, fileError := service.fileManager.CreateMemoFile(created.GameID, created.ID, created.Title, created.Content); fileError != nil {
			_ = service.repository.DeleteMemo(ctx, created.ID)
			service.logger.Error("メモファイル作成に失敗", "error", fileError)
			return result.ErrorResult[*models.Memo]("メモファイル作成に失敗しました", fileError.Error())
		}
	}
	return result.OkResult(created)
}

// UpdateMemo はメモを更新する。
func (service *MemoService) UpdateMemo(ctx context.Context, memoID string, input MemoUpdateInput) result.ApiResult[*models.Memo] {
	trimmedID, detail, ok := requireNonEmpty(memoID, "memoID")
	if !ok {
		return result.ErrorResult[*models.Memo]("メモIDが不正です", detail)
	}

	memo, error := service.repository.GetMemoByID(ctx, trimmedID)
	if error != nil {
		service.logger.Error("メモ取得に失敗", "error", error)
		return result.ErrorResult[*models.Memo]("メモ取得に失敗しました", error.Error())
	}
	if memo == nil {
		return result.ErrorResult[*models.Memo]("メモが見つかりません", "指定されたIDが存在しません")
	}

	oldTitle := memo.Title
	oldContent := memo.Content
	memo.Title = strings.TrimSpace(input.Title)
	memo.Content = input.Content

	updated, error := service.repository.UpdateMemo(ctx, *memo)
	if error != nil {
		service.logger.Error("メモ更新に失敗", "error", error)
		return result.ErrorResult[*models.Memo]("メモ更新に失敗しました", error.Error())
	}

	if service.fileManager != nil {
		if _, fileError := service.fileManager.UpdateMemoFile(updated.GameID, updated.ID, oldTitle, updated.Title, updated.Content); fileError != nil {
			memo.Title = oldTitle
			memo.Content = oldContent
			_, _ = service.repository.UpdateMemo(ctx, *memo)
			service.logger.Error("メモファイル更新に失敗", "error", fileError)
			return result.ErrorResult[*models.Memo]("メモファイル更新に失敗しました", fileError.Error())
		}
	}
	return result.OkResult(updated)
}

// GetMemoByID はメモIDでメモを取得する。
func (service *MemoService) GetMemoByID(ctx context.Context, memoID string) result.ApiResult[*models.Memo] {
	trimmedID, detail, ok := requireNonEmpty(memoID, "memoID")
	if !ok {
		return result.ErrorResult[*models.Memo]("メモIDが不正です", detail)
	}

	memo, error := service.repository.GetMemoByID(ctx, trimmedID)
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
	trimmedID, detail, ok := requireNonEmpty(memoID, "memoID")
	if !ok {
		return result.ErrorResult[bool]("メモIDが不正です", detail)
	}

	memo, error := service.repository.GetMemoByID(ctx, trimmedID)
	if error != nil {
		service.logger.Error("メモ取得に失敗", "error", error)
		return result.ErrorResult[bool]("メモ取得に失敗しました", error.Error())
	}
	if memo == nil {
		return result.ErrorResult[bool]("メモが見つかりません", "指定されたIDが存在しません")
	}

	if error := service.repository.DeleteMemo(ctx, trimmedID); error != nil {
		service.logger.Error("メモ削除に失敗", "error", error)
		return result.ErrorResult[bool]("メモ削除に失敗しました", error.Error())
	}
	if service.fileManager != nil {
		if fileError := service.fileManager.DeleteMemoFile(memo.GameID, memo.ID, memo.Title); fileError != nil {
			service.logger.Error("メモファイル削除に失敗", "error", fileError)
			return result.ErrorResult[bool]("メモファイル削除に失敗しました", fileError.Error())
		}
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
	if _, detail, ok := requireNonEmpty(input.Title, "title"); !ok {
		return errors.New(detail)
	}
	if _, detail, ok := requireNonEmpty(input.Content, "content"); !ok {
		return errors.New(detail)
	}
	if _, detail, ok := requireNonEmpty(input.GameID, "gameID"); !ok {
		return errors.New(detail)
	}
	return nil
}
