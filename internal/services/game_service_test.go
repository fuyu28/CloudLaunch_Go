package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
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
	refreshGamePlayTimeFn        func(ctx context.Context, gameID string) error
	initialRoute                 domain.Route
	createWithInitialRouteCalls  int
	refreshGamePlayTimeCalls     int
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

func (repository *fakeGameRepository) RefreshGamePlayTimeFromSessions(ctx context.Context, gameID string) error {
	repository.refreshGamePlayTimeCalls++
	if repository.refreshGamePlayTimeFn != nil {
		return repository.refreshGamePlayTimeFn(ctx, gameID)
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

	tests := []struct {
		name  string
		input GameInput
	}{
		{
			name:  "空白のみのタイトル",
			input: GameInput{Title: "　 \t", Publisher: "Publisher", ExePath: "/games/game.exe"},
		},
		{
			name:  "空白のみのブランド名",
			input: GameInput{Title: "Game", Publisher: "　 \t", ExePath: "/games/game.exe"},
		},
		{
			name:  "空白のみの実行ファイルパス",
			input: GameInput{Title: "Game", Publisher: "Publisher", ExePath: "　 \t"},
		},
		{
			name:  "日本語101文字のタイトル",
			input: GameInput{Title: strings.Repeat("界", 101), Publisher: "Publisher", ExePath: "/games/game.exe"},
		},
		{
			name:  "絵文字51文字のブランド名",
			input: GameInput{Title: "Game", Publisher: strings.Repeat("🎮", 51), ExePath: "/games/game.exe"},
		},
		{
			name:  "対象外の実行ファイル拡張子",
			input: GameInput{Title: "Game", Publisher: "Publisher", ExePath: "/games/game.bin"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository := &fakeGameRepository{}
			service := NewGameService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

			created, err := service.CreateGame(context.Background(), test.input)
			if err == nil || created != nil {
				t.Fatalf("expected invalid input to fail, got created=%#v err=%v", created, err)
			}
			if repository.createWithInitialRouteCalls != 0 {
				t.Fatalf("repository calls = %d, want 0", repository.createWithInitialRouteCalls)
			}
		})
	}
}

func TestGameServiceCreateGameAcceptsUnicodeBoundariesAndUppercaseExtension(t *testing.T) {
	t.Parallel()

	var stored domain.Game
	repository := &fakeGameRepository{
		createGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) {
			stored = game
			return &game, nil
		},
	}
	service := NewGameService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	title := strings.Repeat("界", 100)
	publisher := strings.Repeat("🎮", 50)
	created, err := service.CreateGame(context.Background(), GameInput{
		Title:     " " + title + " ",
		Publisher: " " + publisher + " ",
		ExePath:   " /games/GAME.EXE ",
	})
	if err != nil || created == nil {
		t.Fatalf("expected boundary input to succeed, got created=%#v err=%v", created, err)
	}
	if repository.createWithInitialRouteCalls != 1 {
		t.Fatalf("repository calls = %d, want 1", repository.createWithInitialRouteCalls)
	}
	if stored.Title != title || stored.Publisher != publisher || stored.ExePath != "/games/GAME.EXE" {
		t.Fatalf("unexpected stored input: %#v", stored)
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

func TestGameServiceUpdateGameRejectsInvalidInputBeforeRepositoryAccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input GameUpdateInput
	}{
		{
			name:  "空白のみのタイトル",
			input: GameUpdateInput{Title: "　 \t", Publisher: "Publisher", ExePath: "/games/game.exe"},
		},
		{
			name:  "空白のみのブランド名",
			input: GameUpdateInput{Title: "Game", Publisher: "　 \t", ExePath: "/games/game.exe"},
		},
		{
			name:  "空白のみの実行ファイルパス",
			input: GameUpdateInput{Title: "Game", Publisher: "Publisher", ExePath: "　 \t"},
		},
		{
			name:  "日本語101文字のタイトル",
			input: GameUpdateInput{Title: strings.Repeat("界", 101), Publisher: "Publisher", ExePath: "/games/game.exe"},
		},
		{
			name:  "絵文字51文字のブランド名",
			input: GameUpdateInput{Title: "Game", Publisher: strings.Repeat("🎮", 51), ExePath: "/games/game.exe"},
		},
		{
			name:  "対象外の実行ファイル拡張子",
			input: GameUpdateInput{Title: "Game", Publisher: "Publisher", ExePath: "/games/game.bin"},
		},
		{
			name:  "対象外のプレイ状態",
			input: GameUpdateInput{Title: "Game", Publisher: "Publisher", ExePath: "/games/game.exe", PlayStatus: "invalid"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			getCalls := 0
			updateCalls := 0
			service := NewGameService(&fakeGameRepository{
				getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
					getCalls++
					return &domain.Game{ID: gameID}, nil
				},
				updateGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) {
					updateCalls++
					return &game, nil
				},
			}, slog.New(slog.NewTextHandler(io.Discard, nil)))

			updated, err := service.UpdateGame(context.Background(), "game-1", test.input)
			if err == nil || updated != nil {
				t.Fatalf("expected invalid input to fail, got updated=%#v err=%v", updated, err)
			}
			if getCalls != 0 || updateCalls != 0 {
				t.Fatalf("repository calls: get=%d update=%d, want 0", getCalls, updateCalls)
			}
		})
	}
}

func TestGameServiceUpdateGameAcceptsUnicodeBoundariesAndUppercaseExtension(t *testing.T) {
	t.Parallel()

	getCalls := 0
	updateCalls := 0
	var stored domain.Game
	service := NewGameService(&fakeGameRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			getCalls++
			return &domain.Game{ID: gameID}, nil
		},
		updateGameFn: func(ctx context.Context, game domain.Game) (*domain.Game, error) {
			updateCalls++
			stored = game
			return &game, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	title := strings.Repeat("界", 100)
	publisher := strings.Repeat("🎮", 50)
	updated, err := service.UpdateGame(context.Background(), "game-1", GameUpdateInput{
		Title:     " " + title + " ",
		Publisher: " " + publisher + " ",
		ExePath:   " /Applications/GAME.APP ",
	})
	if err != nil || updated == nil {
		t.Fatalf("expected boundary input to succeed, got updated=%#v err=%v", updated, err)
	}
	if getCalls != 1 || updateCalls != 1 {
		t.Fatalf("repository calls: get=%d update=%d, want 1 each", getCalls, updateCalls)
	}
	if stored.Title != title || stored.Publisher != publisher || stored.ExePath != "/Applications/GAME.APP" {
		t.Fatalf("unexpected stored input: %#v", stored)
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

func TestGameServiceUpdatePlayTimeRefreshesFromSessions(t *testing.T) {
	t.Parallel()

	lastPlayed := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	repository := &fakeGameRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return &domain.Game{ID: gameID, Title: "Game", TotalPlayTime: 120, LastPlayed: &lastPlayed}, nil
		},
	}
	service := NewGameService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	updated, err := service.UpdatePlayTime(context.Background(), "game-1", 9999, time.Time{})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if repository.refreshGamePlayTimeCalls != 1 {
		t.Fatalf("expected session-derived refresh, got %d calls", repository.refreshGamePlayTimeCalls)
	}
	if updated == nil || updated.TotalPlayTime != 120 {
		t.Fatalf("expected refreshed game totals, got %#v", updated)
	}
}
