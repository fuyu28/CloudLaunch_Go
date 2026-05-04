// @fileoverview プレイセッション管理を提供する。
package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"CloudLaunch_Go/internal/models"
)

// SessionService はプレイセッション関連操作を提供する。
type SessionService struct {
	repository SessionRepository
	logger     *slog.Logger
}

// NewSessionService は SessionService を生成する。
func NewSessionService(repository SessionRepository, logger *slog.Logger) *SessionService {
	return &SessionService{repository: repository, logger: logger}
}

// SessionMutationResult represents metadata that the Wails adapter can use after a session write.
type SessionMutationResult struct {
	GameID string `json:"gameId"`
}

// CreateSession は新しいセッションを作成する。
func (service *SessionService) CreateSession(ctx context.Context, input SessionInput) (*models.PlaySession, error) {
	if error := validateSessionInput(input); error != nil {
		service.logger.Warn("セッション入力が不正です", "error", error)
		return nil, newServiceError("セッション入力が不正です", error.Error())
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
		return nil, newServiceError("セッション作成に失敗しました", error.Error())
	}
	if created != nil {
		service.afterSessionChange(ctx, created.GameID, &input.PlayedAt)
	}
	return created, nil
}

// ListSessionsByGame はゲームIDでセッション一覧を取得する。
func (service *SessionService) ListSessionsByGame(ctx context.Context, gameID string) ([]models.PlaySession, error) {
	sessions, error := service.repository.ListPlaySessionsByGame(ctx, strings.TrimSpace(gameID))
	if error != nil {
		service.logger.Error("セッション取得に失敗", "error", error)
		return nil, newServiceError("セッション取得に失敗しました", error.Error())
	}
	return sessions, nil
}

// DeleteSession はセッションを削除する。
func (service *SessionService) DeleteSession(ctx context.Context, sessionID string) (SessionMutationResult, error) {
	trimmedID, detail, ok := requireNonEmpty(sessionID, "sessionID")
	if !ok {
		service.logger.Warn("セッションIDが不正です", "detail", detail, "sessionId", sessionID)
		return SessionMutationResult{}, newServiceError("セッションIDが不正です", detail)
	}

	session, error := service.repository.GetPlaySessionByID(ctx, trimmedID)
	if error != nil {
		service.logger.Error("セッション取得に失敗", "error", error)
		return SessionMutationResult{}, newServiceError("セッション取得に失敗しました", error.Error())
	}
	if error := service.repository.DeletePlaySession(ctx, trimmedID); error != nil {
		service.logger.Error("セッション削除に失敗", "error", error)
		return SessionMutationResult{}, newServiceError("セッション削除に失敗しました", error.Error())
	}

	mutation := SessionMutationResult{}
	if session != nil {
		service.afterSessionChange(ctx, session.GameID, nil)
		mutation.GameID = session.GameID
	}
	return mutation, nil
}

// UpdateSessionChapter はセッションの章を更新する。
func (service *SessionService) UpdateSessionChapter(ctx context.Context, sessionID string, chapterID *string) (SessionMutationResult, error) {
	trimmedID, detail, ok := requireNonEmpty(sessionID, "sessionID")
	if !ok {
		service.logger.Warn("セッションIDが不正です", "detail", detail, "sessionId", sessionID)
		return SessionMutationResult{}, newServiceError("セッションIDが不正です", detail)
	}

	session, error := service.repository.GetPlaySessionByID(ctx, trimmedID)
	if error != nil {
		service.logger.Error("セッション取得に失敗", "error", error)
		return SessionMutationResult{}, newServiceError("セッション取得に失敗しました", error.Error())
	}
	if error := service.repository.UpdatePlaySessionChapter(ctx, trimmedID, chapterID); error != nil {
		service.logger.Error("セッション章更新に失敗", "error", error)
		return SessionMutationResult{}, newServiceError("セッション章更新に失敗しました", error.Error())
	}

	mutation := SessionMutationResult{}
	if session != nil {
		service.afterSessionChange(ctx, session.GameID, nil)
		mutation.GameID = session.GameID
	}
	return mutation, nil
}

// UpdateSessionName はセッション名を更新する。
func (service *SessionService) UpdateSessionName(ctx context.Context, sessionID string, sessionName string) (SessionMutationResult, error) {
	trimmedID, detail, ok := requireNonEmpty(sessionID, "sessionID")
	if !ok {
		service.logger.Warn("セッションIDが不正です", "detail", detail, "sessionId", sessionID)
		return SessionMutationResult{}, newServiceError("セッションIDが不正です", detail)
	}
	trimmedName, detail, ok := requireNonEmpty(sessionName, "sessionName")
	if !ok {
		service.logger.Warn("セッション名が不正です", "detail", detail, "sessionId", sessionID)
		return SessionMutationResult{}, newServiceError("セッション名が不正です", detail)
	}

	session, error := service.repository.GetPlaySessionByID(ctx, trimmedID)
	if error != nil {
		service.logger.Error("セッション取得に失敗", "error", error)
		return SessionMutationResult{}, newServiceError("セッション取得に失敗しました", error.Error())
	}
	if error := service.repository.UpdatePlaySessionName(ctx, trimmedID, trimmedName); error != nil {
		service.logger.Error("セッション名更新に失敗", "error", error)
		return SessionMutationResult{}, newServiceError("セッション名更新に失敗しました", error.Error())
	}

	mutation := SessionMutationResult{}
	if session != nil {
		service.afterSessionChange(ctx, session.GameID, nil)
		mutation.GameID = session.GameID
	}
	return mutation, nil
}

func (service *SessionService) afterSessionChange(ctx context.Context, gameID string, playedAt *time.Time) {
	_ = service.repository.TouchGameUpdatedAt(ctx, gameID)
	service.recalculateTotalPlayTime(ctx, gameID, playedAt)
}

func (service *SessionService) recalculateTotalPlayTime(ctx context.Context, gameID string, playedAt *time.Time) {
	total, sumErr := service.repository.SumPlaySessionDurationsByGame(ctx, gameID)
	if sumErr != nil {
		service.logger.Error("セッション合計時間の取得に失敗", "error", sumErr, "gameId", gameID)
		return
	}
	if playedAt != nil {
		if err := service.repository.UpdateGameTotalPlayTimeWithLastPlayed(ctx, gameID, total, *playedAt); err != nil {
			service.logger.Error("プレイ時間更新に失敗", "error", err, "gameId", gameID)
		}
		return
	}
	if err := service.repository.UpdateGameTotalPlayTime(ctx, gameID, total); err != nil {
		service.logger.Error("プレイ時間更新に失敗", "error", err, "gameId", gameID)
	}
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
	if _, detail, ok := requireNonEmpty(input.GameID, "gameID"); !ok {
		return errors.New(detail)
	}
	if input.PlayedAt.IsZero() {
		return errors.New("playedAtが空です")
	}
	if input.Duration < 0 {
		return errors.New("durationが不正です")
	}
	return nil
}
