// @fileoverview アップロード履歴管理のビジネスロジックを提供する。
package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"CloudLaunch_Go/internal/models"
)

// UploadService はアップロード履歴の操作を提供する。
type UploadService struct {
	repository UploadRepository
	logger     *slog.Logger
}

// NewUploadService は UploadService を生成する。
func NewUploadService(repository UploadRepository, logger *slog.Logger) *UploadService {
	return &UploadService{repository: repository, logger: logger}
}

// CreateUpload はアップロード履歴を作成する。
func (service *UploadService) CreateUpload(ctx context.Context, input UploadInput) (*models.Upload, error) {
	if error := validateUploadInput(input); error != nil {
		service.logger.Warn("アップロード入力が不正です", "error", error)
		return nil, newServiceError("アップロード入力が不正です", error.Error())
	}

	upload := models.Upload{
		ClientID: input.ClientID,
		Comment:  strings.TrimSpace(input.Comment),
		GameID:   strings.TrimSpace(input.GameID),
	}

	created, error := service.repository.CreateUpload(ctx, upload)
	if error != nil {
		service.logger.Error("アップロード作成に失敗", "error", error)
		return nil, newServiceError("アップロード作成に失敗しました", error.Error())
	}
	return created, nil
}

// ListUploadsByGame はゲームIDでアップロード履歴を取得する。
func (service *UploadService) ListUploadsByGame(ctx context.Context, gameID string) ([]models.Upload, error) {
	uploads, error := service.repository.ListUploadsByGame(ctx, strings.TrimSpace(gameID))
	if error != nil {
		service.logger.Error("アップロード取得に失敗", "error", error)
		return nil, newServiceError("アップロード取得に失敗しました", error.Error())
	}
	return uploads, nil
}

// UploadInput はアップロード作成入力を表す。
type UploadInput struct {
	ClientID *string
	Comment  string
	GameID   string
}

// validateUploadInput はアップロード入力の基本チェックを行う。
func validateUploadInput(input UploadInput) error {
	if _, detail, ok := requireNonEmpty(input.GameID, "gameID"); !ok {
		return errors.New(detail)
	}
	if _, detail, ok := requireNonEmpty(input.Comment, "comment"); !ok {
		return errors.New(detail)
	}
	return nil
}
