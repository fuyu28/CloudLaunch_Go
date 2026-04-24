package services

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"CloudLaunch_Go/internal/models"
)

type fakeProcessMonitorRepository struct {
	createPlaySessionFn func(ctx context.Context, session models.PlaySession) (*models.PlaySession, error)
	getGameByIDFn       func(ctx context.Context, gameID string) (*models.Game, error)
	updateGameFn        func(ctx context.Context, game models.Game) (*models.Game, error)
	listGamesFn         func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error)
}

func (repository fakeProcessMonitorRepository) CreatePlaySession(ctx context.Context, session models.PlaySession) (*models.PlaySession, error) {
	return repository.createPlaySessionFn(ctx, session)
}

func (repository fakeProcessMonitorRepository) GetGameByID(ctx context.Context, gameID string) (*models.Game, error) {
	return repository.getGameByIDFn(ctx, gameID)
}

func (repository fakeProcessMonitorRepository) UpdateGame(ctx context.Context, game models.Game) (*models.Game, error) {
	return repository.updateGameFn(ctx, game)
}

func (repository fakeProcessMonitorRepository) ListGames(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
	return repository.listGamesFn(ctx, searchText, filter, sortBy, sortDirection)
}

func TestProcessMonitorServiceAutoAddGamesFromDatabaseAddsMatchingGame(t *testing.T) {
	t.Parallel()

	service := NewProcessMonitorService(fakeProcessMonitorRepository{
		createPlaySessionFn: func(ctx context.Context, session models.PlaySession) (*models.PlaySession, error) {
			return &session, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) { return nil, nil },
		updateGameFn:  func(ctx context.Context, game models.Game) (*models.Game, error) { return &game, nil },
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return []models.Game{{
				ID:      "game-1",
				Title:   "Game",
				ExePath: `C:\games\game.exe`,
			}}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)

	processes := []ProcessInfo{{Name: "game.exe", Pid: 123, Cmd: `C:\games\game.exe`}}
	normalized := []normalizedProcess{{
		info:          processes[0],
		normalized:    normalizeProcessToken(processes[0].Name),
		normalizedCmd: normalizeProcessPathToken(processes[0].Cmd),
	}}

	service.autoAddGamesFromDatabase(processes, normalized)

	if _, ok := service.monitoredGames["game-1"]; !ok {
		t.Fatalf("expected matching game to be added to monitor list")
	}
}

func TestProcessMonitorServiceSaveSessionUpdatesGameTotals(t *testing.T) {
	t.Parallel()

	var updatedGame models.Game
	service := NewProcessMonitorService(fakeProcessMonitorRepository{
		createPlaySessionFn: func(ctx context.Context, session models.PlaySession) (*models.PlaySession, error) {
			return &session, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) {
			return &models.Game{ID: gameID, Title: "Game", TotalPlayTime: 100}, nil
		},
		updateGameFn: func(ctx context.Context, game models.Game) (*models.Game, error) {
			updatedGame = game
			return &game, nil
		},
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)

	endedAt := time.Date(2026, 4, 24, 20, 0, 0, 0, time.UTC)
	service.saveSession(MonitoringGame{
		GameID:          "game-1",
		GameTitle:       "Game",
		ExeName:         "game.exe",
		AccumulatedTime: 30,
	}, endedAt)

	if updatedGame.TotalPlayTime != 130 {
		t.Fatalf("expected total play time to be updated, got %d", updatedGame.TotalPlayTime)
	}
	if updatedGame.LastPlayed == nil || !updatedGame.LastPlayed.Equal(endedAt) {
		t.Fatalf("expected last played to be updated")
	}
}

func TestProcessMonitorServicePauseSessionMarksGamePaused(t *testing.T) {
	t.Parallel()

	service := NewProcessMonitorService(fakeProcessMonitorRepository{
		createPlaySessionFn: func(ctx context.Context, session models.PlaySession) (*models.PlaySession, error) {
			return &session, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) { return nil, nil },
		updateGameFn:  func(ctx context.Context, game models.Game) (*models.Game, error) { return &game, nil },
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	start := time.Now().Add(-30 * time.Second)
	service.monitoredGames["game-1"] = &MonitoringGame{
		GameID:        "game-1",
		ExeName:       "game.exe",
		PlayStartTime: &start,
	}

	ok := service.PauseSession("game-1")
	if !ok {
		t.Fatalf("expected pause to succeed")
	}
	game := service.monitoredGames["game-1"]
	if !game.IsPaused || game.PlayStartTime != nil {
		t.Fatalf("expected game to be paused")
	}
	if game.AccumulatedTime <= 0 {
		t.Fatalf("expected accumulated time to increase")
	}
}

func TestProcessMonitorServiceEndSessionResetsAccumulatedTime(t *testing.T) {
	t.Parallel()

	service := NewProcessMonitorService(fakeProcessMonitorRepository{
		createPlaySessionFn: func(ctx context.Context, session models.PlaySession) (*models.PlaySession, error) {
			return &session, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) {
			return &models.Game{ID: gameID, Title: "Game"}, nil
		},
		updateGameFn: func(ctx context.Context, game models.Game) (*models.Game, error) { return &game, nil },
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	start := time.Now().Add(-10 * time.Second)
	service.monitoredGames["game-1"] = &MonitoringGame{
		GameID:        "game-1",
		GameTitle:     "Game",
		ExeName:       "game.exe",
		PlayStartTime: &start,
	}

	ok := service.EndSession("game-1")
	if !ok {
		t.Fatalf("expected end session to succeed")
	}
	game := service.monitoredGames["game-1"]
	if game.AccumulatedTime != 0 {
		t.Fatalf("expected accumulated time to reset after save")
	}
	if game.PlayStartTime != nil {
		t.Fatalf("expected play start time to be cleared")
	}
}

func TestProcessMonitorServiceMatchGameProcess(t *testing.T) {
	t.Parallel()

	service := NewProcessMonitorService(fakeProcessMonitorRepository{
		createPlaySessionFn: func(ctx context.Context, session models.PlaySession) (*models.PlaySession, error) {
			return &session, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) { return nil, nil },
		updateGameFn:  func(ctx context.Context, game models.Game) (*models.Game, error) { return &game, nil },
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)

	match := service.matchGameProcess("game.exe", `C:\games\game.exe`, normalizedProcess{
		info:          ProcessInfo{Name: "game.exe", Cmd: `C:\games\game.exe`},
		normalized:    normalizeProcessToken("game.exe"),
		normalizedCmd: normalizeProcessPathToken(`C:\games\game.exe`),
	})
	if !match {
		t.Fatalf("expected exact executable match")
	}

	noMatch := service.matchGameProcess("game.exe", `C:\games\game.exe`, normalizedProcess{
		info:          ProcessInfo{Name: "other.exe", Cmd: `C:\games\other.exe`},
		normalized:    normalizeProcessToken("other.exe"),
		normalizedCmd: normalizeProcessPathToken(`C:\games\other.exe`),
	})
	if noMatch {
		t.Fatalf("expected different executable to not match")
	}
}

func TestProcessMonitorServiceIsGameProcessRunning(t *testing.T) {
	t.Parallel()

	service := NewProcessMonitorService(fakeProcessMonitorRepository{
		createPlaySessionFn: func(ctx context.Context, session models.PlaySession) (*models.PlaySession, error) {
			return &session, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) { return nil, nil },
		updateGameFn:  func(ctx context.Context, game models.Game) (*models.Game, error) { return &game, nil },
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)

	processes := []normalizedProcess{
		{
			info:          ProcessInfo{Name: "other.exe", Cmd: `C:\games\other.exe`},
			normalized:    normalizeProcessToken("other.exe"),
			normalizedCmd: normalizeProcessPathToken(`C:\games\other.exe`),
		},
		{
			info:          ProcessInfo{Name: "game.exe", Cmd: `C:\games\game.exe`},
			normalized:    normalizeProcessToken("game.exe"),
			normalizedCmd: normalizeProcessPathToken(`C:\games\game.exe`),
		},
	}

	if !service.isGameProcessRunning("game.exe", `C:\games\game.exe`, processes) {
		t.Fatalf("expected running game to be detected")
	}
	if service.isGameProcessRunning("missing.exe", `C:\games\missing.exe`, processes) {
		t.Fatalf("expected missing game to not be detected")
	}
}
