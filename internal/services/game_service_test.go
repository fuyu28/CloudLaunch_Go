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

type fakeGameRepository struct {
	listGamesFn      func(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error)
	getGameByIDFn    func(ctx context.Context, gameID string) (*domain.Game, error)
	createGameFn     func(ctx context.Context, game domain.Game) (*domain.Game, error)
	updateGameFn     func(ctx context.Context, game domain.Game) (*domain.Game, error)
	deleteGameFn     func(ctx context.Context, gameID string) error
	createRouteCalls int
}

func (repository fakeGameRepository) ListGames(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error) {
	return repository.listGamesFn(ctx, searchText, filter, sortBy, sortDirection)
}

func (repository fakeGameRepository) GetGameByID(ctx context.Context, gameID string) (*domain.Game, error) {
	return repository.getGameByIDFn(ctx, gameID)
}

func (repository fakeGameRepository) CreateGame(ctx context.Context, game domain.Game) (*domain.Game, error) {
	return repository.createGameFn(ctx, game)
}

func (repository fakeGameRepository) UpdateGame(ctx context.Context, game domain.Game) (*domain.Game, error) {
	return repository.updateGameFn(ctx, game)
}

func (repository fakeGameRepository) DeleteGame(ctx context.Context, gameID string) error {
	return repository.deleteGameFn(ctx, gameID)
}

func (repository *fakeGameRepository) CreateRoute(ctx context.Context, route domain.Route) (*domain.Route, error) {
	repository.createRouteCalls++
	return &route, nil
}

func TestGameServiceCreateGameUsesRepositoryBoundary(t *testing.T) {
	t.Parallel()

	repository := &fakeGameRepository{
		listGamesFn: func(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error) {
			return nil, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return nil, nil
		},
		createGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) {
			return &domain.Game{
				ID:        "game-1",
				Title:     game.Title,
				Publisher: game.Publisher,
				ExePath:   game.ExePath,
			}, nil
		},
		updateGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) {
			return &game, nil
		},
		deleteGameFn: func(ctx context.Context, gameID string) error {
			return nil
		},
	}
	service := NewGameService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := service.CreateGame(context.Background(), GameInput{
		Title:     "Game",
		Publisher: "Publisher",
		ExePath:   "/games/game.exe",
	})

	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result == nil || result.ID != "game-1" {
		t.Fatalf("expected created game id to be returned")
	}
	if repository.createRouteCalls != 1 {
		t.Fatalf("expected initial route to be created once")
	}
}

func TestGameServiceListGetDeleteUseRepositoryBoundary(t *testing.T) {
	t.Parallel()

	game := domain.Game{ID: "game-1", Title: "Game"}
	service := NewGameService(&fakeGameRepository{
		listGamesFn: func(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error) {
			return []domain.Game{game}, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return &game, nil
		},
		createGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) { return &game, nil },
		updateGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) { return &game, nil },
		deleteGameFn: func(ctx context.Context, gameID string) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	listed, err := service.ListGames(context.Background(), " game ", domain.PlayStatus(""), "title", "asc")
	if err != nil || len(listed) != 1 || listed[0].ID != "game-1" {
		t.Fatalf("unexpected list result: %#v", listed)
	}

	got, err := service.GetGameByID(context.Background(), "game-1")
	if err != nil || got == nil || got.ID != "game-1" {
		t.Fatalf("unexpected get result: %#v", got)
	}

	if err := service.DeleteGame(context.Background(), "game-1"); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
}

func TestGameServiceUpdateGameHandlesRepositoryError(t *testing.T) {
	t.Parallel()

	service := NewGameService(&fakeGameRepository{
		listGamesFn: func(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error) {
			return nil, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return nil, errors.New("db down")
		},
		createGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) {
			return nil, nil
		},
		updateGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) {
			return nil, nil
		},
		deleteGameFn: func(ctx context.Context, gameID string) error {
			return nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.UpdateGame(context.Background(), "game-1", GameUpdateInput{
		Title:     "Updated",
		Publisher: "Publisher",
		ExePath:   "/games/game.exe",
	})

	if err == nil {
		t.Fatalf("expected failure")
	}
	if serviceErr := new(ServiceError); !errors.As(err, &serviceErr) || serviceErr.Message == "" {
		t.Fatalf("expected service error details, got %v", err)
	}
}

func TestGameServiceCreateGameRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	service := NewGameService(&fakeGameRepository{
		listGamesFn: func(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error) {
			return nil, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) { return nil, nil },
		createGameFn:  func(ctx context.Context, game domain.Game) (*domain.Game, error) { return nil, nil },
		updateGameFn:  func(ctx context.Context, game domain.Game) (*domain.Game, error) { return &game, nil },
		deleteGameFn:  func(ctx context.Context, gameID string) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.CreateGame(context.Background(), GameInput{
		Title:     "Game",
		Publisher: "",
		ExePath:   "/games/game.exe",
	})

	if err == nil {
		t.Fatalf("expected invalid input to fail")
	}
}

func TestGameServiceUpdateGameReturnsNotFoundWhenMissing(t *testing.T) {
	t.Parallel()

	service := NewGameService(&fakeGameRepository{
		listGamesFn: func(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error) {
			return nil, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) { return nil, nil },
		createGameFn:  func(ctx context.Context, game domain.Game) (*domain.Game, error) { return &game, nil },
		updateGameFn:  func(ctx context.Context, game domain.Game) (*domain.Game, error) { return &game, nil },
		deleteGameFn:  func(ctx context.Context, gameID string) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.UpdateGame(context.Background(), "game-1", GameUpdateInput{
		Title:     "Updated",
		Publisher: "Publisher",
		ExePath:   "/games/game.exe",
	})

	if err == nil {
		t.Fatalf("expected missing game to fail")
	}
}

func TestGameServiceUpdateGameTrimsInputAndPreservesPlayTotals(t *testing.T) {
	t.Parallel()

	lastPlayed := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	clearedAt := lastPlayed.Add(2 * time.Hour)
	currentRouteID := "chapter-3"
	var updatedGame domain.Game
	service := NewGameService(&fakeGameRepository{
		listGamesFn: func(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error) {
			return nil, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return &domain.Game{
				ID:            gameID,
				Title:         "Old",
				Publisher:     "Old Publisher",
				ExePath:       "/old/game.exe",
				PlayStatus:    domain.PlayStatusPlaying,
				TotalPlayTime: 360,
				LastPlayed:    &lastPlayed,
			}, nil
		},
		createGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) { return &game, nil },
		updateGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) {
			updatedGame = game
			return &game, nil
		},
		deleteGameFn: func(ctx context.Context, gameID string) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.UpdateGame(context.Background(), " game-1 ", GameUpdateInput{
		Title:          " New Title ",
		Publisher:      " New Publisher ",
		ExePath:        " /games/new.exe ",
		ClearedAt:      &clearedAt,
		CurrentRouteID: &currentRouteID,
	})

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if updatedGame.Title != "New Title" || updatedGame.Publisher != "New Publisher" || updatedGame.ExePath != "/games/new.exe" {
		t.Fatalf("expected trimmed game fields, got %#v", updatedGame)
	}
	if updatedGame.TotalPlayTime != 360 || updatedGame.LastPlayed == nil || !updatedGame.LastPlayed.Equal(lastPlayed) {
		t.Fatalf("expected play totals to be preserved, got %#v", updatedGame)
	}
	if updatedGame.ClearedAt == nil ||
		!updatedGame.ClearedAt.Equal(clearedAt) ||
		updatedGame.CurrentRouteID == nil ||
		*updatedGame.CurrentRouteID != "chapter-3" {
		t.Fatalf("expected progress fields to be updated, got %#v", updatedGame)
	}
}

// TestGameServiceUpdateGamePreservesClearedAtAndCurrentRouteIDWhenInputNil は、
// フロントの updateGame() がタイトル編集時に ClearedAt/CurrentRouteID を
// undefined（=nil）で送ってきても、既存値を破壊しないことを確認する。
func TestGameServiceUpdateGamePreservesClearedAtAndCurrentRouteIDWhenInputNil(t *testing.T) {
	t.Parallel()

	existingClearedAt := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	existingRouteID := "true-end"
	var updatedGame domain.Game
	service := NewGameService(&fakeGameRepository{
		listGamesFn: func(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error) {
			return nil, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return &domain.Game{
				ID:             gameID,
				Title:          "Old Title",
				Publisher:      "Old Publisher",
				ExePath:        "/old/game.exe",
				PlayStatus:     domain.PlayStatusPlayed,
				ClearedAt:      &existingClearedAt,
				CurrentRouteID: &existingRouteID,
			}, nil
		},
		createGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) { return &game, nil },
		updateGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) {
			updatedGame = game
			return &game, nil
		},
		deleteGameFn: func(ctx context.Context, gameID string) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.UpdateGame(context.Background(), "game-1", GameUpdateInput{
		Title:     "New Title",
		Publisher: "New Publisher",
		ExePath:   "/new/game.exe",
		// PlayStatus, ClearedAt, CurrentRouteID は意図的に未指定（フロント updateGame と同じ形）
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if updatedGame.ClearedAt == nil || !updatedGame.ClearedAt.Equal(existingClearedAt) {
		t.Fatalf("clearedAt は維持されるべき: got %#v", updatedGame.ClearedAt)
	}
	if updatedGame.CurrentRouteID == nil || *updatedGame.CurrentRouteID != existingRouteID {
		t.Fatalf("currentRouteID は維持されるべき: got %#v", updatedGame.CurrentRouteID)
	}
	if updatedGame.PlayStatus != domain.PlayStatusPlayed {
		t.Fatalf("playStatus も維持されるべき: got %q", updatedGame.PlayStatus)
	}
}

// TestGameServiceUpdateGameClearsClearedAtWhenPlayStatusLeavesPlayed は、
// playStatus を played 以外に変更するとき、ClearedAt が自動的にクリアされる
// （整合性のため：played でないゲームにクリア日時が残っていてはならない）。
func TestGameServiceUpdateGameClearsClearedAtWhenPlayStatusLeavesPlayed(t *testing.T) {
	t.Parallel()

	existingClearedAt := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	var updatedGame domain.Game
	service := NewGameService(&fakeGameRepository{
		listGamesFn: func(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error) {
			return nil, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return &domain.Game{
				ID:         gameID,
				Title:      "Game",
				Publisher:  "Pub",
				ExePath:    "/game.exe",
				PlayStatus: domain.PlayStatusPlayed,
				ClearedAt:  &existingClearedAt,
			}, nil
		},
		createGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) { return &game, nil },
		updateGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) {
			updatedGame = game
			return &game, nil
		},
		deleteGameFn: func(ctx context.Context, gameID string) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.UpdateGame(context.Background(), "game-1", GameUpdateInput{
		Title:      "Game",
		Publisher:  "Pub",
		ExePath:    "/game.exe",
		PlayStatus: domain.PlayStatusPlaying,
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if updatedGame.PlayStatus != domain.PlayStatusPlaying {
		t.Fatalf("playStatus = %q, want playing", updatedGame.PlayStatus)
	}
	if updatedGame.ClearedAt != nil {
		t.Fatalf("playStatus が played でないので clearedAt はクリアされるべき: got %#v", updatedGame.ClearedAt)
	}
}

func TestGameServiceListGamesTrimsSearchText(t *testing.T) {
	t.Parallel()

	var capturedSearch string
	service := NewGameService(&fakeGameRepository{
		listGamesFn: func(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error) {
			capturedSearch = searchText
			return []domain.Game{{ID: "game-1", Title: "Game"}}, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) { return nil, nil },
		createGameFn:  func(ctx context.Context, game domain.Game) (*domain.Game, error) { return &game, nil },
		updateGameFn:  func(ctx context.Context, game domain.Game) (*domain.Game, error) { return &game, nil },
		deleteGameFn:  func(ctx context.Context, gameID string) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.ListGames(context.Background(), "  Game  ", domain.PlayStatus(""), "title", "asc")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if capturedSearch != "Game" {
		t.Fatalf("expected search text to be trimmed, got %q", capturedSearch)
	}
}

func TestGameServiceUpdatePlayTimeStoresLastPlayed(t *testing.T) {
	t.Parallel()

	var updatedGame domain.Game
	lastPlayed := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	service := NewGameService(&fakeGameRepository{
		listGamesFn: func(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error) {
			return nil, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return &domain.Game{ID: gameID, Title: "Game"}, nil
		},
		createGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) { return &game, nil },
		updateGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) {
			updatedGame = game
			return &game, nil
		},
		deleteGameFn: func(ctx context.Context, gameID string) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.UpdatePlayTime(context.Background(), "game-1", 240, lastPlayed)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if updatedGame.LastPlayed == nil || !updatedGame.LastPlayed.Equal(lastPlayed) {
		t.Fatalf("expected lastPlayed to be updated")
	}
	if updatedGame.TotalPlayTime != 240 {
		t.Fatalf("expected total play time to be updated")
	}
}
