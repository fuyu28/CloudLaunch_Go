// @fileoverview プレイセッション管理を提供する。
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

// SessionService はプレイセッション関連操作を提供する。
type SessionService struct {
	repository *db.Repository
	logger     *slog.Logger
}

// NewSessionService は SessionService を生成する。
func NewSessionService(repository *db.Repository, logger *slog.Logger) *SessionService {
	return &SessionService{repository: repository, logger: logger}
}

// CreateSession は新しいセッションを作成する。
func (service *SessionService) CreateSession(ctx context.Context, input SessionInput) result.ApiResult[*models.PlaySession] {
	if error := validateSessionInput(input); error != nil {
		return result.ErrorResult[*models.PlaySession]("セッション入力が不正です", error.Error())
	}

	session := models.PlaySession{
		GameID:      strings.TrimSpace(input.GameID),
		PlayedAt:    input.PlayedAt,
		Duration:    input.Duration,
		SessionName: input.SessionName,
		ChapterID:   input.ChapterID,
		UploadID:    input.UploadID,
	}

	created, error := service.repository.CreatePlaySession(ctx, session)
	if error != nil {
		service.logger.Error("セッション作成に失敗", "error", error)
		return result.ErrorResult[*models.PlaySession]("セッション作成に失敗しました", error.Error())
	}
	return result.OkResult(created)
}

// ListSessionsByGame はゲームIDでセッション一覧を取得する。
func (service *SessionService) ListSessionsByGame(ctx context.Context, gameID string) result.ApiResult[[]models.PlaySession] {
	sessions, error := service.repository.ListPlaySessionsByGame(ctx, strings.TrimSpace(gameID))
	if error != nil {
		service.logger.Error("セッション取得に失敗", "error", error)
		return result.ErrorResult[[]models.PlaySession]("セッション取得に失敗しました", error.Error())
	}
	return result.OkResult(sessions)
}

// DeleteSession はセッションを削除する。
func (service *SessionService) DeleteSession(ctx context.Context, sessionID string) result.ApiResult[bool] {
	if strings.TrimSpace(sessionID) == "" {
		return result.ErrorResult[bool]("セッションIDが不正です", "sessionIDが空です")
	}

	if error := service.repository.DeletePlaySession(ctx, strings.TrimSpace(sessionID)); error != nil {
		service.logger.Error("セッション削除に失敗", "error", error)
		return result.ErrorResult[bool]("セッション削除に失敗しました", error.Error())
	}
	return result.OkResult(true)
}

// UpdateSessionChapter はセッションの章を更新する。
func (service *SessionService) UpdateSessionChapter(ctx context.Context, sessionID string, chapterID *string) result.ApiResult[bool] {
	if strings.TrimSpace(sessionID) == "" {
		return result.ErrorResult[bool]("セッションIDが不正です", "sessionIDが空です")
	}
	if error := service.repository.UpdatePlaySessionChapter(ctx, strings.TrimSpace(sessionID), chapterID); error != nil {
		service.logger.Error("セッション章更新に失敗", "error", error)
		return result.ErrorResult[bool]("セッション章更新に失敗しました", error.Error())
	}
	return result.OkResult(true)
}

// UpdateSessionName はセッション名を更新する。
func (service *SessionService) UpdateSessionName(ctx context.Context, sessionID string, sessionName string) result.ApiResult[bool] {
	if strings.TrimSpace(sessionID) == "" {
		return result.ErrorResult[bool]("セッションIDが不正です", "sessionIDが空です")
	}
	if strings.TrimSpace(sessionName) == "" {
		return result.ErrorResult[bool]("セッション名が不正です", "sessionNameが空です")
	}
	if error := service.repository.UpdatePlaySessionName(ctx, strings.TrimSpace(sessionID), strings.TrimSpace(sessionName)); error != nil {
		service.logger.Error("セッション名更新に失敗", "error", error)
		return result.ErrorResult[bool]("セッション名更新に失敗しました", error.Error())
	}
	return result.OkResult(true)
}

// SessionInput はセッション作成入力を表す。
type SessionInput struct {
	GameID      string
	PlayedAt    time.Time
	Duration    int64
	SessionName *string
	ChapterID   *string
	UploadID    *string
}

// validateSessionInput はセッション入力を検証する。
func validateSessionInput(input SessionInput) error {
	if strings.TrimSpace(input.GameID) == "" {
		return errors.New("gameIDが空です")
	}
	if input.PlayedAt.IsZero() {
		return errors.New("playedAtが空です")
	}
	if input.Duration < 0 {
		return errors.New("durationが不正です")
	}
	return nil
}
