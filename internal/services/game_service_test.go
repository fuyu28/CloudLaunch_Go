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
	listGamesFn                  func(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error)
	getGameByIDFn                func(ctx context.Context, gameID string) (*domain.Game, error)
	createGameFn                 func(ctx context.Context, game domain.Game) (*domain.Game, error)
	createGameWithInitialRouteFn func(ctx context.Context, game domain.Game, initialRoute domain.Route) (*domain.Game, error)
	updateGameFn                 func(ctx context.Context, game domain.Game) (*domain.Game, error)
	deleteGameFn                 func(ctx context.Context, gameID string) error
	listPendingMemoCleanupFn     func(ctx context.Context) ([]string, error)
	clearPendingMemoCleanupFn    func(ctx context.Context, gameID string) error
	initialRoute                 domain.Route
	createWithInitialRouteCalls  int
}

func (repository fakeGameRepository) ListGames(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error) {
	return repository.listGamesFn(ctx, searchText, filter, sortBy, sortDirection)
}

func (repository fakeGameRepository) GetGameByID(ctx context.Context, gameID string) (*domain.Game, error) {
	return repository.getGameByIDFn(ctx, gameID)
}

func (repository *fakeGameRepository) CreateGameWithInitialRoute(ctx context.Context, game domain.Game, initialRoute domain.Route) (*domain.Game, error) {
	repository.createWithInitialRouteCalls++
	repository.initialRoute = initialRoute
	if repository.createGameWithInitialRouteFn != nil {
		return repository.createGameWithInitialRouteFn(ctx, game, initialRoute)
	}
	return repository.createGameFn(ctx, game)
}

func (repository fakeGameRepository) UpdateGame(ctx context.Context, game domain.Game) (*domain.Game, error) {
	return repository.updateGameFn(ctx, game)
}

func (repository fakeGameRepository) DeleteGameAndQueueMemoCleanup(ctx context.Context, gameID string) error {
	return repository.deleteGameFn(ctx, gameID)
}

func (repository fakeGameRepository) ListPendingMemoCleanup(ctx context.Context) ([]string, error) {
	if repository.listPendingMemoCleanupFn != nil {
		return repository.listPendingMemoCleanupFn(ctx)
	}
	return nil, nil
}

func (repository fakeGameRepository) ClearPendingMemoCleanup(ctx context.Context, gameID string) error {
	if repository.clearPendingMemoCleanupFn != nil {
		return repository.clearPendingMemoCleanupFn(ctx, gameID)
	}
	return nil
}

type fakeMemoDirectoryCleaner struct {
	deleteFn func(gameID string) error
}

func (cleaner fakeMemoDirectoryCleaner) DeleteGameMemoFiles(gameID string) error {
	return cleaner.deleteFn(gameID)
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
	if repository.createWithInitialRouteCalls != 1 {
		t.Fatalf("expected atomic create boundary to be called once")
	}
	if repository.initialRoute.Name != "メインルート" || repository.initialRoute.Order != 1 || repository.initialRoute.GameID != "" {
		t.Fatalf("unexpected initial route: %#v", repository.initialRoute)
	}
}

func TestGameServiceCreateGameReturnsErrorWhenAtomicCreateFails(t *testing.T) {
	t.Parallel()

	repository := &fakeGameRepository{
		createGameWithInitialRouteFn: func(ctx context.Context, game domain.Game, initialRoute domain.Route) (*domain.Game, error) {
			return nil, errors.New("route insert failed")
		},
	}
	service := NewGameService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	created, err := service.CreateGame(context.Background(), GameInput{
		Title:     "Game",
		Publisher: "Publisher",
		ExePath:   "/games/game.exe",
	})

	if err == nil || created != nil {
		t.Fatalf("expected atomic create failure, got created=%#v err=%v", created, err)
	}
	if repository.createWithInitialRouteCalls != 1 {
		t.Fatalf("expected atomic create boundary to be called once")
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
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), fakeMemoDirectoryCleaner{
		deleteFn: func(gameID string) error { return nil },
	})

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

func TestGameServiceDeleteGameCleansMemoAndClearsPendingMarker(t *testing.T) {
	t.Parallel()

	var deletedGameID string
	var cleanedGameID string
	var clearedGameID string
	repository := &fakeGameRepository{
		deleteGameFn: func(ctx context.Context, gameID string) error {
			deletedGameID = gameID
			return nil
		},
		clearPendingMemoCleanupFn: func(ctx context.Context, gameID string) error {
			clearedGameID = gameID
			return nil
		},
	}
	service := NewGameService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)), fakeMemoDirectoryCleaner{
		deleteFn: func(gameID string) error {
			cleanedGameID = gameID
			return nil
		},
	})

	if err := service.DeleteGame(context.Background(), " game-1 "); err != nil {
		t.Fatalf("DeleteGame: %v", err)
	}
	if deletedGameID != "game-1" || cleanedGameID != "game-1" || clearedGameID != "game-1" {
		t.Fatalf("unexpected delete flow: delete=%q cleanup=%q clear=%q", deletedGameID, cleanedGameID, clearedGameID)
	}
}

func TestGameServiceDeleteGamePreservesMemoWhenDatabaseDeleteFails(t *testing.T) {
	t.Parallel()

	cleanupCalls := 0
	service := NewGameService(&fakeGameRepository{
		deleteGameFn: func(ctx context.Context, gameID string) error {
			return errors.New("delete failed")
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), fakeMemoDirectoryCleaner{
		deleteFn: func(gameID string) error {
			cleanupCalls++
			return nil
		},
	})

	if err := service.DeleteGame(context.Background(), "game-1"); err == nil {
		t.Fatal("expected database deletion failure")
	}
	if cleanupCalls != 0 {
		t.Fatalf("memo cleanup calls = %d, want 0", cleanupCalls)
	}
}

func TestGameServiceRetriesPendingMemoCleanupAfterFileFailure(t *testing.T) {
	t.Parallel()

	pending := true
	clearCalls := 0
	repository := &fakeGameRepository{
		deleteGameFn: func(ctx context.Context, gameID string) error {
			return nil
		},
		listPendingMemoCleanupFn: func(ctx context.Context) ([]string, error) {
			if pending {
				return []string{"game-1"}, nil
			}
			return nil, nil
		},
		clearPendingMemoCleanupFn: func(ctx context.Context, gameID string) error {
			clearCalls++
			pending = false
			return nil
		},
	}
	cleanupCalls := 0
	service := NewGameService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)), fakeMemoDirectoryCleaner{
		deleteFn: func(gameID string) error {
			cleanupCalls++
			if cleanupCalls == 1 {
				return errors.New("file is locked")
			}
			return nil
		},
	})

	if err := service.DeleteGame(context.Background(), "game-1"); err == nil {
		t.Fatal("expected initial memo cleanup failure")
	}
	if !pending || clearCalls != 0 {
		t.Fatalf("pending marker was cleared after failure: pending=%v clearCalls=%d", pending, clearCalls)
	}
	if err := service.RetryPendingMemoCleanup(context.Background()); err != nil {
		t.Fatalf("RetryPendingMemoCleanup: %v", err)
	}
	if pending || cleanupCalls != 2 || clearCalls != 1 {
		t.Fatalf("unexpected retry state: pending=%v cleanupCalls=%d clearCalls=%d", pending, cleanupCalls, clearCalls)
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
