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
	if created != nil {
		_ = service.repository.TouchGameUpdatedAt(ctx, created.GameID)
		service.updateGamePlayTime(ctx, created.GameID, input.Duration, input.PlayedAt)
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
	trimmedID, detail, ok := requireNonEmpty(sessionID, "sessionID")
	if !ok {
		return result.ErrorResult[bool]("セッションIDが不正です", detail)
	}

	session, error := service.repository.GetPlaySessionByID(ctx, trimmedID)
	if error != nil {
		service.logger.Error("セッション取得に失敗", "error", error)
		return result.ErrorResult[bool]("セッション取得に失敗しました", error.Error())
	}
	if error := service.repository.DeletePlaySession(ctx, trimmedID); error != nil {
		service.logger.Error("セッション削除に失敗", "error", error)
		return result.ErrorResult[bool]("セッション削除に失敗しました", error.Error())
	}
	if session != nil {
		_ = service.repository.TouchGameUpdatedAt(ctx, session.GameID)
		service.recalculateTotalPlayTime(ctx, session.GameID)
	}
	return result.OkResult(true)
}

// UpdateSessionChapter はセッションの章を更新する。
func (service *SessionService) UpdateSessionChapter(ctx context.Context, sessionID string, chapterID *string) result.ApiResult[bool] {
	trimmedID, detail, ok := requireNonEmpty(sessionID, "sessionID")
	if !ok {
		return result.ErrorResult[bool]("セッションIDが不正です", detail)
	}
	session, error := service.repository.GetPlaySessionByID(ctx, trimmedID)
	if error != nil {
		service.logger.Error("セッション取得に失敗", "error", error)
		return result.ErrorResult[bool]("セッション取得に失敗しました", error.Error())
	}
	if error := service.repository.UpdatePlaySessionChapter(ctx, trimmedID, chapterID); error != nil {
		service.logger.Error("セッション章更新に失敗", "error", error)
		return result.ErrorResult[bool]("セッション章更新に失敗しました", error.Error())
	}
	if session != nil {
		_ = service.repository.TouchGameUpdatedAt(ctx, session.GameID)
	}
	return result.OkResult(true)
}

// UpdateSessionName はセッション名を更新する。
func (service *SessionService) UpdateSessionName(ctx context.Context, sessionID string, sessionName string) result.ApiResult[bool] {
	trimmedID, detail, ok := requireNonEmpty(sessionID, "sessionID")
	if !ok {
		return result.ErrorResult[bool]("セッションIDが不正です", detail)
	}
	trimmedName, detail, ok := requireNonEmpty(sessionName, "sessionName")
	if !ok {
		return result.ErrorResult[bool]("セッション名が不正です", detail)
	}
	session, error := service.repository.GetPlaySessionByID(ctx, trimmedID)
	if error != nil {
		service.logger.Error("セッション取得に失敗", "error", error)
		return result.ErrorResult[bool]("セッション取得に失敗しました", error.Error())
	}
	if error := service.repository.UpdatePlaySessionName(ctx, trimmedID, trimmedName); error != nil {
		service.logger.Error("セッション名更新に失敗", "error", error)
		return result.ErrorResult[bool]("セッション名更新に失敗しました", error.Error())
	}
	if session != nil {
		_ = service.repository.TouchGameUpdatedAt(ctx, session.GameID)
	}
	return result.OkResult(true)
}

func (service *SessionService) updateGamePlayTime(
	ctx context.Context,
	gameID string,
	duration int64,
	playedAt time.Time,
) {
	current, err := service.repository.GetGameByID(ctx, gameID)
	if err != nil || current == nil {
		service.logger.Error("ゲーム取得に失敗", "error", err, "gameId", gameID)
		return
	}

	if duration > 0 {
		total, sumErr := service.repository.SumPlaySessionDurationsByGame(ctx, gameID)
		if sumErr != nil {
			service.logger.Error("セッション合計時間の取得に失敗", "error", sumErr, "gameId", gameID)
			current.TotalPlayTime += duration
		} else {
			current.TotalPlayTime = total
		}
	}
	if current.LastPlayed == nil || playedAt.After(*current.LastPlayed) {
		current.LastPlayed = &playedAt
	}

	if _, err := service.repository.UpdateGame(ctx, *current); err != nil {
		service.logger.Error("プレイ時間更新に失敗", "error", err, "gameId", gameID)
	}
}

func (service *SessionService) recalculateTotalPlayTime(ctx context.Context, gameID string) {
	current, err := service.repository.GetGameByID(ctx, gameID)
	if err != nil || current == nil {
		service.logger.Error("ゲーム取得に失敗", "error", err, "gameId", gameID)
		return
	}
	total, sumErr := service.repository.SumPlaySessionDurationsByGame(ctx, gameID)
	if sumErr != nil {
		service.logger.Error("セッション合計時間の取得に失敗", "error", sumErr, "gameId", gameID)
		return
	}
	current.TotalPlayTime = total
	if _, err := service.repository.UpdateGame(ctx, *current); err != nil {
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
