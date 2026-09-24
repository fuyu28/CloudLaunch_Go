package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"CloudLaunch_Go/internal/domain"
)

type fakeSessionRepository struct {
	session                *domain.PlaySession
	touchedGameID          string
	createAndRefreshCalls  int
	deleteAndRefreshCalls  int
	lastCreatedDuration    int64
	refreshSkippedOnMutate bool
	deleteReturnsGameID    string
}

func (repository *fakeSessionRepository) CreatePlaySessionAndRefreshGame(ctx context.Context, session domain.PlaySession) (*domain.PlaySession, error) {
	repository.createAndRefreshCalls++
	session.ID = "session-1"
	repository.session = &session
	repository.lastCreatedDuration = session.Duration
	return &session, nil
}

func (repository *fakeSessionRepository) ListPlaySessionsByGame(ctx context.Context, gameID string) ([]domain.PlaySession, error) {
	if repository.session == nil {
		return nil, nil
	}
	return []domain.PlaySession{*repository.session}, nil
}

func (repository *fakeSessionRepository) GetPlaySessionByID(ctx context.Context, sessionID string) (*domain.PlaySession, error) {
	return repository.session, nil
}

func (repository *fakeSessionRepository) DeletePlaySessionAndRefreshGame(ctx context.Context, sessionID string) (string, error) {
	repository.deleteAndRefreshCalls++
	if repository.deleteReturnsGameID != "" {
		return repository.deleteReturnsGameID, nil
	}
	if repository.session == nil {
		return "", nil
	}
	return repository.session.GameID, nil
}

func (repository *fakeSessionRepository) UpdatePlaySessionRoute(ctx context.Context, sessionID string, chapterID *string) error {
	if repository.session != nil {
		repository.session.RouteID = chapterID
	}
	repository.refreshSkippedOnMutate = true
	return nil
}

func (repository *fakeSessionRepository) UpdatePlaySessionName(ctx context.Context, sessionID string, sessionName string) error {
	if repository.session != nil {
		repository.session.SessionName = &sessionName
	}
	repository.refreshSkippedOnMutate = true
	return nil
}

func (repository *fakeSessionRepository) TouchGameUpdatedAt(ctx context.Context, gameID string) error {
	repository.touchedGameID = gameID
	return nil
}

func TestSessionServiceDeleteSessionUsesAtomicRefresh(t *testing.T) {
	t.Parallel()

	repository := &fakeSessionRepository{
		session: &domain.PlaySession{
			ID:       "session-1",
			GameID:   "game-1",
			PlayedAt: time.Now(),
			Duration: 120,
		},
	}
	service := NewSessionService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := service.DeleteSession(context.Background(), "session-1"); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if repository.deleteAndRefreshCalls != 1 {
		t.Fatalf("expected atomic delete+refresh, got %d calls", repository.deleteAndRefreshCalls)
	}
}

func TestSessionServiceListSessionsByGameUsesRepositoryBoundary(t *testing.T) {
	t.Parallel()

	repository := &fakeSessionRepository{
		session: func() *domain.PlaySession {
			name := "Session 1"
			return &domain.PlaySession{
				ID:          "session-1",
				GameID:      "game-1",
				PlayedAt:    time.Now(),
				Duration:    120,
				SessionName: &name,
			}
		}(),
	}
	service := NewSessionService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := service.ListSessionsByGame(context.Background(), "game-1")
	if err != nil || len(result) != 1 || result[0].ID != "session-1" {
		t.Fatalf("unexpected list result: %#v", result)
	}
}

func TestSessionServiceCreateSessionUsesAtomicRefresh(t *testing.T) {
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
	if repository.createAndRefreshCalls != 1 {
		t.Fatalf("expected atomic create+refresh, got %d calls", repository.createAndRefreshCalls)
	}
	if repository.lastCreatedDuration != 300 {
		t.Fatalf("expected duration 300, got %d", repository.lastCreatedDuration)
	}
	if repository.touchedGameID != "" {
		t.Fatalf("create should not separately touch updatedAt")
	}
}

func TestSessionServiceUpdateSessionNameAllowsEmptyToClear(t *testing.T) {
	t.Parallel()

	// 空白のみの入力は「セッション名をクリア（NULL 化）」として許容される。
	// リポジトリ層では NULLIF によって空文字が NULL に変換されるため、
	// サービス層では空白トリム後の空文字をそのままリポジトリへ渡す挙動を検証する。
	existingName := "old-name"
	repository := &fakeSessionRepository{
		session: &domain.PlaySession{ID: "session-1", GameID: "game-1", SessionName: &existingName},
	}
	service := NewSessionService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := service.UpdateSessionName(context.Background(), "session-1", "   "); err != nil {
		t.Fatalf("expected empty/whitespace session name to succeed for clearing, got %v", err)
	}
	if repository.session.SessionName == nil || *repository.session.SessionName != "" {
		t.Fatalf("expected session name to be cleared to empty string (repository-level NULLIF handles the NULL conversion)")
	}
}

func TestSessionServiceUpdateSessionNameTouchesUpdatedAtWithoutRefresh(t *testing.T) {
	t.Parallel()

	repository := &fakeSessionRepository{
		session: &domain.PlaySession{ID: "session-1", GameID: "game-1", Duration: 120},
	}
	service := NewSessionService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := service.UpdateSessionName(context.Background(), "session-1", "  Chapter 1  "); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if repository.session.SessionName == nil || *repository.session.SessionName != "Chapter 1" {
		t.Fatalf("expected session name to be trimmed and stored")
	}
	if repository.touchedGameID != "game-1" {
		t.Fatalf("expected game updated timestamp to be touched")
	}
	if repository.createAndRefreshCalls != 0 || repository.deleteAndRefreshCalls != 0 {
		t.Fatalf("name change must not recalculate play time")
	}
}

func TestSessionServiceUpdateSessionRouteTouchesUpdatedAtWithoutRefresh(t *testing.T) {
	t.Parallel()

	repository := &fakeSessionRepository{
		session: &domain.PlaySession{ID: "session-1", GameID: "game-1", Duration: 180},
	}
	service := NewSessionService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	chapterID := "chapter-2"

	if err := service.UpdateSessionRoute(context.Background(), "session-1", &chapterID); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if repository.session.RouteID == nil || *repository.session.RouteID != "chapter-2" {
		t.Fatalf("expected route id to be stored")
	}
	if repository.touchedGameID != "game-1" {
		t.Fatalf("expected game updated timestamp to be touched")
	}
	if repository.createAndRefreshCalls != 0 || repository.deleteAndRefreshCalls != 0 {
		t.Fatalf("route change must not recalculate play time")
	}
}

func TestSessionServiceDeleteSessionHandlesLookupError(t *testing.T) {
	t.Parallel()

	repository := &fakeSessionRepositoryWithError{deleteErr: errors.New("db down")}
	service := NewSessionService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := service.DeleteSession(context.Background(), "session-1"); err == nil {
		t.Fatalf("expected failure")
	}
}

type fakeSessionRepositoryWithError struct {
	deleteErr error
}

func (repository *fakeSessionRepositoryWithError) CreatePlaySessionAndRefreshGame(ctx context.Context, session domain.PlaySession) (*domain.PlaySession, error) {
	return &session, nil
}
func (repository *fakeSessionRepositoryWithError) ListPlaySessionsByGame(ctx context.Context, gameID string) ([]domain.PlaySession, error) {
	return nil, nil
}
func (repository *fakeSessionRepositoryWithError) GetPlaySessionByID(ctx context.Context, sessionID string) (*domain.PlaySession, error) {
	return &domain.PlaySession{ID: sessionID, GameID: "game-1"}, nil
}
func (repository *fakeSessionRepositoryWithError) DeletePlaySessionAndRefreshGame(ctx context.Context, sessionID string) (string, error) {
	return "", repository.deleteErr
}
func (repository *fakeSessionRepositoryWithError) UpdatePlaySessionRoute(ctx context.Context, sessionID string, chapterID *string) error {
	return nil
}
func (repository *fakeSessionRepositoryWithError) UpdatePlaySessionName(ctx context.Context, sessionID string, sessionName string) error {
	return nil
}
func (repository *fakeSessionRepositoryWithError) TouchGameUpdatedAt(ctx context.Context, gameID string) error {
	return nil
}
