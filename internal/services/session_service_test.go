package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"CloudLaunch_Go/internal/models"
)

type fakeSessionRepository struct {
	session               *models.PlaySession
	totalDuration         int64
	touchedGameID         string
	updatedWithLastPlayed *time.Time
	updateTotalCalls      int
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

func (repository *fakeSessionRepository) TouchGameUpdatedAt(ctx context.Context, gameID string) error {
	repository.touchedGameID = gameID
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
	repository.updateTotalCalls++
	return nil
}

func (repository *fakeSessionRepository) UpdateGameTotalPlayTimeWithLastPlayed(ctx context.Context, gameID string, totalPlayTime int64, playedAt time.Time) error {
	repository.totalDuration = totalPlayTime
	repository.updatedWithLastPlayed = &playedAt
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

	result, err := service.DeleteSession(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if result.GameID != "game-1" {
		t.Fatalf("expected affected game id to be returned")
	}
	if repository.touchedGameID != "game-1" {
		t.Fatalf("expected touch updated at to be called")
	}
	if repository.updateTotalCalls != 1 {
		t.Fatalf("expected total play time recalculation without playedAt")
	}
}

func TestSessionServiceListSessionsByGameUsesRepositoryBoundary(t *testing.T) {
	t.Parallel()

	repository := &fakeSessionRepository{
		session: func() *models.PlaySession {
			return &models.PlaySession{
				ID:       "session-1",
				GameID:   "game-1",
				PlayedAt: time.Now(),
				Duration: 120,
			}
		}(),
	}
	service := NewSessionService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := service.ListSessionsByGame(context.Background(), "game-1")
	if err != nil || len(result) != 1 || result[0].ID != "session-1" {
		t.Fatalf("unexpected list result: %#v", result)
	}
}

func TestSessionServiceCreateSessionRecalculatesTotalWithLastPlayed(t *testing.T) {
	t.Parallel()

	repository := &fakeSessionRepository{}
	service := NewSessionService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	playedAt := time.Date(2026, 4, 24, 18, 0, 0, 0, time.UTC)

	_, err := service.CreateSession(context.Background(), SessionInput{
		GameID:   "game-1",
		PlayedAt: playedAt,
		Duration: 300,
	})

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if repository.updatedWithLastPlayed == nil || !repository.updatedWithLastPlayed.Equal(playedAt) {
		t.Fatalf("expected last played update to be called")
	}
	if repository.touchedGameID != "game-1" {
		t.Fatalf("expected game touch after create")
	}
}

func TestSessionServiceDeleteSessionHandlesLookupError(t *testing.T) {
	t.Parallel()

	repository := &fakeSessionRepositoryWithError{getErr: errors.New("db down")}
	service := NewSessionService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.DeleteSession(context.Background(), "session-1")
	if err == nil {
		t.Fatalf("expected failure")
	}
}

type fakeSessionRepositoryWithError struct {
	getErr error
}

func (repository *fakeSessionRepositoryWithError) CreatePlaySession(ctx context.Context, session models.PlaySession) (*models.PlaySession, error) {
	return &session, nil
}
func (repository *fakeSessionRepositoryWithError) ListPlaySessionsByGame(ctx context.Context, gameID string) ([]models.PlaySession, error) {
	return nil, nil
}
func (repository *fakeSessionRepositoryWithError) GetPlaySessionByID(ctx context.Context, sessionID string) (*models.PlaySession, error) {
	return nil, repository.getErr
}
func (repository *fakeSessionRepositoryWithError) DeletePlaySession(ctx context.Context, sessionID string) error {
	return nil
}
func (repository *fakeSessionRepositoryWithError) TouchGameUpdatedAt(ctx context.Context, gameID string) error {
	return nil
}
func (repository *fakeSessionRepositoryWithError) SumPlaySessionDurationsByGame(ctx context.Context, gameID string) (int64, error) {
	return 0, nil
}
func (repository *fakeSessionRepositoryWithError) UpdateGameTotalPlayTime(ctx context.Context, gameID string, totalPlayTime int64) error {
	return nil
}
func (repository *fakeSessionRepositoryWithError) UpdateGameTotalPlayTimeWithLastPlayed(ctx context.Context, gameID string, totalPlayTime int64, playedAt time.Time) error {
	return nil
}
