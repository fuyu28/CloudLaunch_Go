// プレイセッション管理を提供する。
package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"CloudLaunch_Go/internal/domain"
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

// CreateSession は新しいセッションを作成する。
// Game.totalPlayTime / lastPlayed はリポジトリの原子的再計算で更新される。
func (service *SessionService) CreateSession(ctx context.Context, input SessionInput) (*domain.PlaySession, error) {
	if error := validateSessionInput(input); error != nil {
		service.logger.Warn("セッション入力が不正です", "error", error)
		return nil, newServiceError("セッション入力が不正です", error.Error())
	}

	session := domain.PlaySession{
		GameID:      strings.TrimSpace(input.GameID),
		PlayedAt:    input.PlayedAt,
		Duration:    input.Duration,
		SessionName: input.SessionName,
		RouteID:     input.RouteID,
	}

	created, error := service.repository.CreatePlaySessionAndRefreshGame(ctx, session)
	if error != nil {
		service.logger.Error("セッション作成に失敗", "error", error)
		return nil, newServiceError("セッション作成に失敗しました", error.Error())
	}
	return created, nil
}

// ListSessionsByGame はゲームIDでセッション一覧を取得する。
func (service *SessionService) ListSessionsByGame(ctx context.Context, gameID string) ([]domain.PlaySession, error) {
	sessions, error := service.repository.ListPlaySessionsByGame(ctx, strings.TrimSpace(gameID))
	if error != nil {
		service.logger.Error("セッション取得に失敗", "error", error)
		return nil, newServiceError("セッション取得に失敗しました", error.Error())
	}
	return sessions, nil
}

// DeleteSession はセッションを削除する。
func (service *SessionService) DeleteSession(ctx context.Context, sessionID string) error {
	trimmedID, detail, ok := requireNonEmpty(sessionID, "sessionID")
	if !ok {
		service.logger.Warn("セッションIDが不正です", "detail", detail, "sessionId", sessionID)
		return newServiceError("セッションIDが不正です", detail)
	}

	if _, error := service.repository.DeletePlaySessionAndRefreshGame(ctx, trimmedID); error != nil {
		service.logger.Error("セッション削除に失敗", "error", error)
		return newServiceError("セッション削除に失敗しました", error.Error())
	}
	return nil
}

// UpdateSessionRoute はセッションのルートを更新する。
// duration は変わらないためプレイ時間は再計算せず、Game.updatedAt のみ触る。
func (service *SessionService) UpdateSessionRoute(ctx context.Context, sessionID string, chapterID *string) error {
	trimmedID, detail, ok := requireNonEmpty(sessionID, "sessionID")
	if !ok {
		service.logger.Warn("セッションIDが不正です", "detail", detail, "sessionId", sessionID)
		return newServiceError("セッションIDが不正です", detail)
	}

	session, error := service.repository.GetPlaySessionByID(ctx, trimmedID)
	if error != nil {
		service.logger.Error("セッション取得に失敗", "error", error)
		return newServiceError("セッション取得に失敗しました", error.Error())
	}
	if error := service.repository.UpdatePlaySessionRoute(ctx, trimmedID, chapterID); error != nil {
		service.logger.Error("セッションルート更新に失敗", "error", error)
		return newServiceError("セッションルート更新に失敗しました", error.Error())
	}
	if session != nil {
		_ = service.repository.TouchGameUpdatedAt(ctx, session.GameID)
	}
	return nil
}

// UpdateSessionName はセッション名を更新する。
// 空文字（または空白のみ）を渡した場合は NULL クリアとして扱う。
// フロントエンドから「セッション名を消したい」ユースケースを許可するため。
// duration は変わらないためプレイ時間は再計算せず、Game.updatedAt のみ触る。
func (service *SessionService) UpdateSessionName(ctx context.Context, sessionID string, sessionName string) error {
	trimmedID, detail, ok := requireNonEmpty(sessionID, "sessionID")
	if !ok {
		service.logger.Warn("セッションIDが不正です", "detail", detail, "sessionId", sessionID)
		return newServiceError("セッションIDが不正です", detail)
	}
	trimmedName := strings.TrimSpace(sessionName)

	session, error := service.repository.GetPlaySessionByID(ctx, trimmedID)
	if error != nil {
		service.logger.Error("セッション取得に失敗", "error", error)
		return newServiceError("セッション取得に失敗しました", error.Error())
	}
	if error := service.repository.UpdatePlaySessionName(ctx, trimmedID, trimmedName); error != nil {
		service.logger.Error("セッション名更新に失敗", "error", error)
		return newServiceError("セッション名更新に失敗しました", error.Error())
	}
	if session != nil {
		_ = service.repository.TouchGameUpdatedAt(ctx, session.GameID)
	}
	return nil
}

// SessionInput はセッション作成入力を表す。
type SessionInput struct {
	GameID      string
	PlayedAt    time.Time
	Duration    int64
	SessionName *string
	RouteID     *string
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
