package db_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/infrastructure/db"
)

func newTestRepo(t *testing.T) *db.Repository {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.ApplyMigrations(conn); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return db.NewRepository(conn)
}

func newGame(title, exePath string) domain.Game {
	return domain.Game{Title: title, Publisher: "Pub", ExePath: exePath}
}

// --- playStatus 導出 ---

func TestRepositoryPlayStatusDerivedOnRead(t *testing.T) {
	t.Parallel()

	lastPlayed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clearedAt := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name       string
		lastPlayed *time.Time
		clearedAt  *time.Time
		want       domain.PlayStatus
	}{
		{"unplayed when both nil", nil, nil, domain.PlayStatusUnplayed},
		{"playing when lastPlayed set", &lastPlayed, nil, domain.PlayStatusPlaying},
		{"played when clearedAt set", nil, &clearedAt, domain.PlayStatusPlayed},
		{"played takes priority over lastPlayed", &lastPlayed, &clearedAt, domain.PlayStatusPlayed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo := newTestRepo(t)
			ctx := context.Background()

			g := newGame("Game", "/game.exe")
			g.LastPlayed = tc.lastPlayed
			g.ClearedAt = tc.clearedAt

			created, err := repo.CreateGame(ctx, g)
			if err != nil {
				t.Fatalf("CreateGame: %v", err)
			}

			got, err := repo.GetGameByID(ctx, created.ID)
			if err != nil || got == nil {
				t.Fatalf("GetGameByID: %v", err)
			}
			if got.PlayStatus != tc.want {
				t.Errorf("PlayStatus = %q, want %q", got.PlayStatus, tc.want)
			}
		})
	}
}

// --- ListGames フィルタ ---

func TestRepositoryListGamesFilterByPlayStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newTestRepo(t)

	lastPlayed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clearedAt := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	unplayed, _ := repo.CreateGame(ctx, newGame("Unplayed", "/a.exe"))
	playing, _ := repo.CreateGame(ctx, domain.Game{Title: "Playing", Publisher: "Pub", ExePath: "/b.exe", LastPlayed: &lastPlayed})
	played, _ := repo.CreateGame(ctx, domain.Game{Title: "Played", Publisher: "Pub", ExePath: "/c.exe", ClearedAt: &clearedAt})

	cases := []struct {
		name    string
		filter  domain.PlayStatus
		wantIDs []string
	}{
		{"unplayed filter", domain.PlayStatusUnplayed, []string{unplayed.ID}},
		{"playing filter", domain.PlayStatusPlaying, []string{playing.ID}},
		{"played filter", domain.PlayStatusPlayed, []string{played.ID}},
		{"no filter returns all", "", []string{unplayed.ID, playing.ID, played.ID}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			games, err := repo.ListGames(ctx, "", tc.filter, "title", "asc")
			if err != nil {
				t.Fatalf("ListGames: %v", err)
			}
			if len(games) != len(tc.wantIDs) {
				t.Fatalf("got %d games, want %d", len(games), len(tc.wantIDs))
			}
			ids := make(map[string]bool, len(games))
			for _, g := range games {
				ids[g.ID] = true
			}
			for _, id := range tc.wantIDs {
				if !ids[id] {
					t.Errorf("expected game %q in results", id)
				}
			}
		})
	}
}

// --- Game CRUD ---

func TestRepositoryGameCRUDRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newTestRepo(t)

	created, err := repo.CreateGame(ctx, newGame("My Game", "/game.exe"))
	if err != nil || created == nil || created.ID == "" {
		t.Fatalf("CreateGame: %v", err)
	}

	got, err := repo.GetGameByID(ctx, created.ID)
	if err != nil || got == nil || got.Title != "My Game" {
		t.Fatalf("GetGameByID: %v (got %v)", err, got)
	}

	got.Title = "Updated"
	updated, err := repo.UpdateGame(ctx, *got)
	if err != nil || updated == nil || updated.Title != "Updated" {
		t.Fatalf("UpdateGame: %v (got %v)", err, updated)
	}

	if err := repo.DeleteGame(ctx, created.ID); err != nil {
		t.Fatalf("DeleteGame: %v", err)
	}
	gone, err := repo.GetGameByID(ctx, created.ID)
	if err != nil || gone != nil {
		t.Fatalf("expected nil after delete, got %v, err=%v", gone, err)
	}
}

// --- UpdateGameTotalPlayTimeWithLastPlayed ---

func TestRepositoryLastPlayedOnlyAdvances(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newTestRepo(t)

	older := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	game, _ := repo.CreateGame(ctx, newGame("Game", "/game.exe"))

	if err := repo.UpdateGameTotalPlayTimeWithLastPlayed(ctx, game.ID, 100, older); err != nil {
		t.Fatalf("first update: %v", err)
	}
	got, _ := repo.GetGameByID(ctx, game.ID)
	if got.LastPlayed == nil || !got.LastPlayed.Equal(older) {
		t.Fatalf("want lastPlayed=%v, got %v", older, got.LastPlayed)
	}

	if err := repo.UpdateGameTotalPlayTimeWithLastPlayed(ctx, game.ID, 200, newer); err != nil {
		t.Fatalf("newer update: %v", err)
	}
	got, _ = repo.GetGameByID(ctx, game.ID)
	if got.LastPlayed == nil || !got.LastPlayed.Equal(newer) {
		t.Fatalf("want lastPlayed advanced to %v, got %v", newer, got.LastPlayed)
	}

	if err := repo.UpdateGameTotalPlayTimeWithLastPlayed(ctx, game.ID, 300, older); err != nil {
		t.Fatalf("older update: %v", err)
	}
	got, _ = repo.GetGameByID(ctx, game.ID)
	if got.LastPlayed == nil || !got.LastPlayed.Equal(newer) {
		t.Fatalf("want lastPlayed unchanged at %v, got %v", newer, got.LastPlayed)
	}
}

// --- Session CRUD ---

func TestRepositorySessionCRUD(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newTestRepo(t)

	game, _ := repo.CreateGame(ctx, newGame("Game", "/game.exe"))
	playedAt := time.Date(2026, 1, 1, 20, 0, 0, 0, time.UTC)

	session, err := repo.CreatePlaySession(ctx, domain.PlaySession{
		GameID:   game.ID,
		PlayedAt: playedAt,
		Duration: 3600,
	})
	if err != nil || session == nil {
		t.Fatalf("CreatePlaySession: %v", err)
	}

	sessions, err := repo.ListPlaySessionsByGame(ctx, game.ID)
	if err != nil || len(sessions) != 1 || sessions[0].Duration != 3600 {
		t.Fatalf("ListPlaySessionsByGame: got %v, err=%v", sessions, err)
	}

	if err := repo.DeletePlaySession(ctx, session.ID); err != nil {
		t.Fatalf("DeletePlaySession: %v", err)
	}
	sessions, _ = repo.ListPlaySessionsByGame(ctx, game.ID)
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions after delete, got %d", len(sessions))
	}
}

// --- Route カスケード削除 ---

func TestRepositoryRoutesDeletedWithGame(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newTestRepo(t)

	game, _ := repo.CreateGame(ctx, newGame("Game", "/game.exe"))
	_, err := repo.CreateRoute(ctx, domain.Route{
		Name:   "Route A",
		Order:  1,
		GameID: game.ID,
	})
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}

	if err := repo.DeleteGame(ctx, game.ID); err != nil {
		t.Fatalf("DeleteGame: %v", err)
	}

	routes, err := repo.ListRoutesByGame(ctx, game.ID)
	if err != nil || len(routes) != 0 {
		t.Fatalf("expected routes cascade deleted, got %d, err=%v", len(routes), err)
	}
}
