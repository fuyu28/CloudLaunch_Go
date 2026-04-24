package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"CloudLaunch_Go/internal/models"
)

type fakeGameRepository struct {
	listGamesFn   func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error)
	getGameByIDFn func(ctx context.Context, gameID string) (*models.Game, error)
	createGameFn  func(ctx context.Context, game models.Game) (*models.Game, error)
	updateGameFn  func(ctx context.Context, game models.Game) (*models.Game, error)
	deleteGameFn  func(ctx context.Context, gameID string) error
}

func (repository fakeGameRepository) ListGames(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
	return repository.listGamesFn(ctx, searchText, filter, sortBy, sortDirection)
}

func (repository fakeGameRepository) GetGameByID(ctx context.Context, gameID string) (*models.Game, error) {
	return repository.getGameByIDFn(ctx, gameID)
}

func (repository fakeGameRepository) CreateGame(ctx context.Context, game models.Game) (*models.Game, error) {
	return repository.createGameFn(ctx, game)
}

func (repository fakeGameRepository) UpdateGame(ctx context.Context, game models.Game) (*models.Game, error) {
	return repository.updateGameFn(ctx, game)
}

func (repository fakeGameRepository) DeleteGame(ctx context.Context, gameID string) error {
	return repository.deleteGameFn(ctx, gameID)
}

func (repository fakeGameRepository) CreateChapter(ctx context.Context, chapter models.Chapter) (*models.Chapter, error) {
	return &chapter, nil
}

func TestGameServiceCreateGameUsesRepositoryBoundary(t *testing.T) {
	t.Parallel()

	service := NewGameService(fakeGameRepository{
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) {
			return nil, nil
		},
		createGameFn: func(ctx context.Context, game models.Game) (*models.Game, error) {
			return &models.Game{
				ID:        "game-1",
				Title:     game.Title,
				Publisher: game.Publisher,
				ExePath:   game.ExePath,
			}, nil
		},
		updateGameFn: func(ctx context.Context, game models.Game) (*models.Game, error) {
			return &game, nil
		},
		deleteGameFn: func(ctx context.Context, gameID string) error {
			return nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.CreateGame(context.Background(), GameInput{
		Title:     "Game",
		Publisher: "Publisher",
		ExePath:   "/games/game.exe",
	})

	if !result.Success {
		t.Fatalf("expected success, got error: %#v", result.Error)
	}
	if result.Data == nil || result.Data.ID != "game-1" {
		t.Fatalf("expected created game id to be returned")
	}
}

func TestGameServiceUpdateGameHandlesRepositoryError(t *testing.T) {
	t.Parallel()

	service := NewGameService(fakeGameRepository{
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) {
			return nil, errors.New("db down")
		},
		createGameFn: func(ctx context.Context, game models.Game) (*models.Game, error) {
			return nil, nil
		},
		updateGameFn: func(ctx context.Context, game models.Game) (*models.Game, error) {
			return nil, nil
		},
		deleteGameFn: func(ctx context.Context, gameID string) error {
			return nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.UpdateGame(context.Background(), "game-1", GameUpdateInput{
		Title:     "Updated",
		Publisher: "Publisher",
		ExePath:   "/games/game.exe",
	})

	if result.Success {
		t.Fatalf("expected failure")
	}
	if result.Error == nil || result.Error.Message == "" {
		t.Fatalf("expected api error details")
	}
}
