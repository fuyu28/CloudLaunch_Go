package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/models"
)

type fakeCloudSyncRepository struct {
	getGameByIDFn              func(ctx context.Context, gameID string) (*models.Game, error)
	listGamesFn                func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error)
	listPlaySessionsByGameFn   func(ctx context.Context, gameID string) ([]models.PlaySession, error)
	upsertGameSyncFn           func(ctx context.Context, game models.Game) error
	deletePlaySessionsByGameFn func(ctx context.Context, gameID string) error
	upsertPlaySessionSyncFn    func(ctx context.Context, session models.PlaySession) error
	sumPlaySessionDurationsFn  func(ctx context.Context, gameID string) (int64, error)
	updateGameTotalPlayTimeFn  func(ctx context.Context, gameID string, totalPlayTime int64) error
}

func (repository fakeCloudSyncRepository) GetGameByID(ctx context.Context, gameID string) (*models.Game, error) {
	return repository.getGameByIDFn(ctx, gameID)
}

func (repository fakeCloudSyncRepository) ListGames(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
	return repository.listGamesFn(ctx, searchText, filter, sortBy, sortDirection)
}

func (repository fakeCloudSyncRepository) ListPlaySessionsByGame(ctx context.Context, gameID string) ([]models.PlaySession, error) {
	return repository.listPlaySessionsByGameFn(ctx, gameID)
}

func (repository fakeCloudSyncRepository) UpsertGameSync(ctx context.Context, game models.Game) error {
	return repository.upsertGameSyncFn(ctx, game)
}

func (repository fakeCloudSyncRepository) DeletePlaySessionsByGame(ctx context.Context, gameID string) error {
	return repository.deletePlaySessionsByGameFn(ctx, gameID)
}

func (repository fakeCloudSyncRepository) UpsertPlaySessionSync(ctx context.Context, session models.PlaySession) error {
	return repository.upsertPlaySessionSyncFn(ctx, session)
}

func (repository fakeCloudSyncRepository) SumPlaySessionDurationsByGame(ctx context.Context, gameID string) (int64, error) {
	return repository.sumPlaySessionDurationsFn(ctx, gameID)
}

func (repository fakeCloudSyncRepository) UpdateGameTotalPlayTime(ctx context.Context, gameID string, totalPlayTime int64) error {
	return repository.updateGameTotalPlayTimeFn(ctx, gameID, totalPlayTime)
}

func TestCloudSyncServiceLoadLocalGamesUsesRepositoryBoundary(t *testing.T) {
	t.Parallel()

	service := NewCloudSyncService(config.Config{}, nil, fakeCloudSyncRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) {
			return &models.Game{ID: gameID, Title: "Game"}, nil
		},
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
		listPlaySessionsByGameFn: func(ctx context.Context, gameID string) ([]models.PlaySession, error) {
			return []models.PlaySession{{ID: "session-1", GameID: gameID, PlayedAt: time.Now(), Duration: 10}}, nil
		},
		upsertGameSyncFn:           func(ctx context.Context, game models.Game) error { return nil },
		deletePlaySessionsByGameFn: func(ctx context.Context, gameID string) error { return nil },
		upsertPlaySessionSyncFn:    func(ctx context.Context, session models.PlaySession) error { return nil },
		sumPlaySessionDurationsFn:  func(ctx context.Context, gameID string) (int64, error) { return 0, nil },
		updateGameTotalPlayTimeFn:  func(ctx context.Context, gameID string, totalPlayTime int64) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := service.loadLocalGames(context.Background(), "game-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected one local game bundle")
	}
	if result["game-1"].Game.ID != "game-1" {
		t.Fatalf("expected bundle keyed by requested game id")
	}
}

func TestCloudSyncServiceLoadLocalGamesReturnsRepositoryError(t *testing.T) {
	t.Parallel()

	service := NewCloudSyncService(config.Config{}, nil, fakeCloudSyncRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) {
			return nil, nil
		},
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, errors.New("db down")
		},
		listPlaySessionsByGameFn:   func(ctx context.Context, gameID string) ([]models.PlaySession, error) { return nil, nil },
		upsertGameSyncFn:           func(ctx context.Context, game models.Game) error { return nil },
		deletePlaySessionsByGameFn: func(ctx context.Context, gameID string) error { return nil },
		upsertPlaySessionSyncFn:    func(ctx context.Context, session models.PlaySession) error { return nil },
		sumPlaySessionDurationsFn:  func(ctx context.Context, gameID string) (int64, error) { return 0, nil },
		updateGameTotalPlayTimeFn:  func(ctx context.Context, gameID string, totalPlayTime int64) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.loadLocalGames(context.Background(), "")
	if err == nil {
		t.Fatalf("expected repository error")
	}
}

func TestCloudSyncServiceSyncGameRejectsInvalidGameID(t *testing.T) {
	t.Parallel()

	service := NewCloudSyncService(config.Config{}, nil, fakeCloudSyncRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) { return nil, nil },
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
		listPlaySessionsByGameFn:   func(ctx context.Context, gameID string) ([]models.PlaySession, error) { return nil, nil },
		upsertGameSyncFn:           func(ctx context.Context, game models.Game) error { return nil },
		deletePlaySessionsByGameFn: func(ctx context.Context, gameID string) error { return nil },
		upsertPlaySessionSyncFn:    func(ctx context.Context, session models.PlaySession) error { return nil },
		sumPlaySessionDurationsFn:  func(ctx context.Context, gameID string) (int64, error) { return 0, nil },
		updateGameTotalPlayTimeFn:  func(ctx context.Context, gameID string, totalPlayTime int64) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.SyncGame(context.Background(), "default", "   ")
	if result.Success {
		t.Fatalf("expected invalid game id to fail")
	}
}

func TestCloudSyncServiceSyncAllGamesFailsInOfflineMode(t *testing.T) {
	t.Parallel()

	service := NewCloudSyncService(config.Config{}, nil, fakeCloudSyncRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) { return nil, nil },
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
		listPlaySessionsByGameFn:   func(ctx context.Context, gameID string) ([]models.PlaySession, error) { return nil, nil },
		upsertGameSyncFn:           func(ctx context.Context, game models.Game) error { return nil },
		deletePlaySessionsByGameFn: func(ctx context.Context, gameID string) error { return nil },
		upsertPlaySessionSyncFn:    func(ctx context.Context, session models.PlaySession) error { return nil },
		sumPlaySessionDurationsFn:  func(ctx context.Context, gameID string) (int64, error) { return 0, nil },
		updateGameTotalPlayTimeFn:  func(ctx context.Context, gameID string, totalPlayTime int64) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.SetOfflineMode(true)

	result := service.SyncAllGames(context.Background(), "default")
	if result.Success {
		t.Fatalf("expected offline sync to fail")
	}
}

func TestCloudSyncServiceDeleteGameFromCloudFailsInOfflineMode(t *testing.T) {
	t.Parallel()

	service := NewCloudSyncService(config.Config{}, nil, fakeCloudSyncRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) { return nil, nil },
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
		listPlaySessionsByGameFn:   func(ctx context.Context, gameID string) ([]models.PlaySession, error) { return nil, nil },
		upsertGameSyncFn:           func(ctx context.Context, game models.Game) error { return nil },
		deletePlaySessionsByGameFn: func(ctx context.Context, gameID string) error { return nil },
		upsertPlaySessionSyncFn:    func(ctx context.Context, session models.PlaySession) error { return nil },
		sumPlaySessionDurationsFn:  func(ctx context.Context, gameID string) (int64, error) { return 0, nil },
		updateGameTotalPlayTimeFn:  func(ctx context.Context, gameID string, totalPlayTime int64) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.SetOfflineMode(true)

	result := service.DeleteGameFromCloud(context.Background(), "default", "game-1")
	if result.Success {
		t.Fatalf("expected offline delete to fail")
	}
}
