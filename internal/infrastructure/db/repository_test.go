package db_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/infrastructure/db"
)

func newTestRepo(t *testing.T) *db.Repository {
	t.Helper()
	repository, _ := newTestRepoWithConnection(t)
	return repository
}

func newTestRepoWithConnection(t *testing.T) (*db.Repository, *sql.DB) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := db.ApplyMigrations(conn); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return db.NewRepository(conn), conn
}

func newGame(title, exePath string) domain.Game {
	return domain.Game{Title: title, Publisher: "Pub", ExePath: exePath, PlayStatus: domain.PlayStatusUnplayed}
}

// --- playStatus 保存・読み取り ---

func TestRepositoryPlayStatusStoredAndRead(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		playStatus domain.PlayStatus
	}{
		{"unplayed", domain.PlayStatusUnplayed},
		{"playing", domain.PlayStatusPlaying},
		{"played", domain.PlayStatusPlayed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo := newTestRepo(t)
			ctx := context.Background()

			g := newGame("Game", "/game.exe")
			g.PlayStatus = tc.playStatus

			created, err := repo.CreateGame(ctx, g)
			if err != nil {
				t.Fatalf("CreateGame: %v", err)
			}

			got, err := repo.GetGameByID(ctx, created.ID)
			if err != nil || got == nil {
				t.Fatalf("GetGameByID: %v", err)
			}
			if got.PlayStatus != tc.playStatus {
				t.Errorf("PlayStatus = %q, want %q", got.PlayStatus, tc.playStatus)
			}
		})
	}
}

// --- ListGames フィルタ ---

func TestRepositoryListGamesFilterByPlayStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newTestRepo(t)

	unplayed, _ := repo.CreateGame(ctx, domain.Game{Title: "Unplayed", Publisher: "Pub", ExePath: "/a.exe", PlayStatus: domain.PlayStatusUnplayed})
	playing, _ := repo.CreateGame(ctx, domain.Game{Title: "Playing", Publisher: "Pub", ExePath: "/b.exe", PlayStatus: domain.PlayStatusPlaying})
	played, _ := repo.CreateGame(ctx, domain.Game{Title: "Played", Publisher: "Pub", ExePath: "/c.exe", PlayStatus: domain.PlayStatusPlayed})

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

// --- ApplyPullResult ---

func TestApplyPullResultNormalizesMissingRouteRefs(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	ctx := context.Background()

	created, err := repo.CreateGame(ctx, newGame("RouteGame", "/route.exe"))
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	missingRoute := "nonexistent-route-id"
	game := *created
	game.CurrentRouteID = &missingRoute
	sessions := []domain.PlaySession{
		{ID: "sess-1", GameID: created.ID, PlayedAt: time.Now().UTC(), Duration: 60, RouteID: &missingRoute, UpdatedAt: time.Now().UTC()},
	}

	if err := repo.ApplyPullResult(ctx, game, sessions, "head-1", "{\"files\":{}}"); err != nil {
		t.Fatalf("ApplyPullResult should not fail on missing route refs: %v", err)
	}

	got, err := repo.GetGameByID(ctx, created.ID)
	if err != nil || got == nil {
		t.Fatalf("GetGameByID: %v", err)
	}
	if got.CurrentRouteID != nil {
		t.Fatalf("currentRouteId should be normalized to NULL, got %v", *got.CurrentRouteID)
	}

	savedSessions, err := repo.ListPlaySessionsByGame(ctx, created.ID)
	if err != nil {
		t.Fatalf("ListPlaySessionsByGame: %v", err)
	}
	if len(savedSessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(savedSessions))
	}
	if savedSessions[0].RouteID != nil {
		t.Fatalf("session routeId should be normalized to NULL, got %v", *savedSessions[0].RouteID)
	}
}

func TestApplyPullResultPersistsHeadAndTree(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	ctx := context.Background()

	created, err := repo.CreateGame(ctx, newGame("HeadGame", "/head.exe"))
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	if err := repo.ApplyPullResult(ctx, *created, nil, "head-xyz", "{\"files\":{\"a.sav\":\"h\"}}"); err != nil {
		t.Fatalf("ApplyPullResult: %v", err)
	}

	got, err := repo.GetGameByID(ctx, created.ID)
	if err != nil || got == nil {
		t.Fatalf("GetGameByID: %v", err)
	}
	if got.LocalSyncHead == nil || *got.LocalSyncHead != "head-xyz" {
		t.Fatalf("localSyncHead not persisted, got %v", got.LocalSyncHead)
	}
	tree, err := repo.GetLocalSaveTree(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetLocalSaveTree: %v", err)
	}
	if tree != "{\"files\":{\"a.sav\":\"h\"}}" {
		t.Fatalf("localSaveTree not persisted, got %q", tree)
	}
}

// --- CreateGameWithInitialRoute ---

func TestCreateGameWithInitialRouteCreatesExactlyOneRoute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newTestRepo(t)

	created, err := repo.CreateGameWithInitialRoute(
		ctx,
		newGame("Atomic Game", "/atomic.exe"),
		domain.Route{Name: "メインルート", Order: 1},
	)
	if err != nil {
		t.Fatalf("CreateGameWithInitialRoute: %v", err)
	}
	if created == nil || created.ID == "" {
		t.Fatalf("expected created game, got %#v", created)
	}

	routes, err := repo.ListRoutesByGame(ctx, created.ID)
	if err != nil {
		t.Fatalf("ListRoutesByGame: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("routes count = %d, want 1", len(routes))
	}
	if routes[0].Name != "メインルート" || routes[0].Order != 1 || routes[0].GameID != created.ID {
		t.Fatalf("unexpected initial route: %#v", routes[0])
	}
	if routes[0].ID == "" || routes[0].CreatedAt.IsZero() {
		t.Fatalf("expected generated route id and timestamp: %#v", routes[0])
	}
}

func TestCreateGameWithInitialRouteRollsBackGameWhenRouteInsertFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, conn := newTestRepoWithConnection(t)
	if _, err := conn.ExecContext(ctx, `
		CREATE TRIGGER fail_initial_route
		BEFORE INSERT ON "Route"
		BEGIN
			SELECT RAISE(ABORT, 'route insert failed');
		END
	`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	created, err := repo.CreateGameWithInitialRoute(
		ctx,
		newGame("Rollback Game", "/rollback.exe"),
		domain.Route{Name: "メインルート", Order: 1},
	)
	if err == nil || created != nil {
		t.Fatalf("expected route insert failure, got created=%#v err=%v", created, err)
	}

	games, err := repo.ListGames(ctx, "", "", "title", "asc")
	if err != nil {
		t.Fatalf("ListGames: %v", err)
	}
	if len(games) != 0 {
		t.Fatalf("game insert was not rolled back: %#v", games)
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

// TestOpenSetsBusyTimeout は Open が busy_timeout を設定し、瞬間的なロック競合を
// 即 SQLITE_BUSY で失敗させず待機させることを確認する。
func TestOpenSetsBusyTimeout(t *testing.T) {
	t.Parallel()
	conn, err := db.Open(filepath.Join(t.TempDir(), "busy.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	var timeout int
	if err := conn.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if timeout < 5000 {
		t.Fatalf("busy_timeout should be >= 5000ms, got %d", timeout)
	}
}
