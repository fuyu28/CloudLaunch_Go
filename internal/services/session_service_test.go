package services

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"CloudLaunch_Go/internal/models"
)

type fakeSessionRepository struct {
	session       *models.PlaySession
	totalDuration int64
}

func (repository *fakeSessionRepository) CreatePlaySession(ctx context.Context, session models.PlaySession) (*models.PlaySession, error) {
	session.ID = "session-1"
	repository.session = &session
	return &session, nil
}

func (repository *fakeSessionRepository) ListPlaySessionsByGame(ctx context.Context, gameID string) ([]models.PlaySession, error) {
	if repository.session == nil {
		return nil, nil
	}
	return []models.PlaySession{*repository.session}, nil
}

func (repository *fakeSessionRepository) GetPlaySessionByID(ctx context.Context, sessionID string) (*models.PlaySession, error) {
	return repository.session, nil
}

func (repository *fakeSessionRepository) DeletePlaySession(ctx context.Context, sessionID string) error {
	return nil
}

func (repository *fakeSessionRepository) UpdatePlaySessionChapter(ctx context.Context, sessionID string, chapterID *string) error {
	if repository.session != nil {
		repository.session.ChapterID = chapterID
	}
	return nil
}

func (repository *fakeSessionRepository) UpdatePlaySessionName(ctx context.Context, sessionID string, sessionName string) error {
	if repository.session != nil {
		repository.session.SessionName = &sessionName
	}
	return nil
}

func (repository *fakeSessionRepository) TouchGameUpdatedAt(ctx context.Context, gameID string) error {
	return nil
}

func (repository *fakeSessionRepository) SumPlaySessionDurationsByGame(ctx context.Context, gameID string) (int64, error) {
	if repository.totalDuration != 0 {
		return repository.totalDuration, nil
	}
	if repository.session == nil {
		return 0, nil
	}
	return repository.session.Duration, nil
}

func (repository *fakeSessionRepository) UpdateGameTotalPlayTime(ctx context.Context, gameID string, totalPlayTime int64) error {
	repository.totalDuration = totalPlayTime
	return nil
}

func (repository *fakeSessionRepository) UpdateGameTotalPlayTimeWithLastPlayed(ctx context.Context, gameID string, totalPlayTime int64, playedAt time.Time) error {
	repository.totalDuration = totalPlayTime
	return nil
}

func TestSessionServiceDeleteSessionReturnsGameIDForAdapterUse(t *testing.T) {
	t.Parallel()

	repository := &fakeSessionRepository{
		session: &models.PlaySession{
			ID:       "session-1",
			GameID:   "game-1",
			PlayedAt: time.Now(),
			Duration: 120,
		},
	}
	service := NewSessionService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.DeleteSession(context.Background(), "session-1")

	if !result.Success {
		t.Fatalf("expected success, got %#v", result.Error)
	}
	if result.Data.GameID != "game-1" {
		t.Fatalf("expected affected game id to be returned")
	}
}
