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

	if err := repo.DeleteGameAndQueueMemoCleanup(ctx, created.ID); err != nil {
		t.Fatalf("DeleteGameAndQueueMemoCleanup: %v", err)
	}
	gone, err := repo.GetGameByID(ctx, created.ID)
	if err != nil || gone != nil {
		t.Fatalf("expected nil after delete, got %v, err=%v", gone, err)
	}
}

func TestDeleteGameAndQueueMemoCleanupCommitsPendingMarker(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newTestRepo(t)
	game, err := repo.CreateGame(ctx, newGame("Delete Game", "/delete.exe"))
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	if err := repo.DeleteGameAndQueueMemoCleanup(ctx, game.ID); err != nil {
		t.Fatalf("DeleteGameAndQueueMemoCleanup: %v", err)
	}
	if got, err := repo.GetGameByID(ctx, game.ID); err != nil || got != nil {
		t.Fatalf("game was not deleted: game=%#v err=%v", got, err)
	}
	pending, err := repo.ListPendingMemoCleanup(ctx)
	if err != nil {
		t.Fatalf("ListPendingMemoCleanup: %v", err)
	}
	if len(pending) != 1 || pending[0] != game.ID {
		t.Fatalf("pending cleanup = %#v, want [%q]", pending, game.ID)
	}
	if err := repo.ClearPendingMemoCleanup(ctx, game.ID); err != nil {
		t.Fatalf("ClearPendingMemoCleanup: %v", err)
	}
	pending, err = repo.ListPendingMemoCleanup(ctx)
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending cleanup was not cleared: pending=%#v err=%v", pending, err)
	}
}

func TestDeleteGameAndQueueMemoCleanupRollsBackMarkerAndPreservesGameOnDeleteFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, conn := newTestRepoWithConnection(t)
	game, err := repo.CreateGame(ctx, newGame("Preserved Game", "/preserved.exe"))
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `
		CREATE TRIGGER fail_game_delete
		BEFORE DELETE ON "Game"
		BEGIN
			SELECT RAISE(ABORT, 'game delete failed');
		END
	`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	if err := repo.DeleteGameAndQueueMemoCleanup(ctx, game.ID); err == nil {
		t.Fatal("expected game deletion failure")
	}
	if got, err := repo.GetGameByID(ctx, game.ID); err != nil || got == nil {
		t.Fatalf("game should be preserved: game=%#v err=%v", got, err)
	}
	pending, err := repo.ListPendingMemoCleanup(ctx)
	if err != nil {
		t.Fatalf("ListPendingMemoCleanup: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending marker should roll back: %#v", pending)
	}
}

// --- PlaySession 正本 + Game 派生キャッシュ ---

func TestCreatePlaySessionAndRefreshGameUpdatesTotals(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newTestRepo(t)
	game, _ := repo.CreateGame(ctx, newGame("Game", "/game.exe"))

	older := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	if _, err := repo.CreatePlaySessionAndRefreshGame(ctx, domain.PlaySession{
		GameID: game.ID, PlayedAt: older, Duration: 100,
	}); err != nil {
		t.Fatalf("create older: %v", err)
	}
	if _, err := repo.CreatePlaySessionAndRefreshGame(ctx, domain.PlaySession{
		GameID: game.ID, PlayedAt: newer, Duration: 50,
	}); err != nil {
		t.Fatalf("create newer: %v", err)
	}

	got, err := repo.GetGameByID(ctx, game.ID)
	if err != nil || got == nil {
		t.Fatalf("GetGameByID: %v", err)
	}
	if got.TotalPlayTime != 150 {
		t.Fatalf("totalPlayTime = %d, want 150", got.TotalPlayTime)
	}
	if got.LastPlayed == nil || !got.LastPlayed.Equal(newer) {
		t.Fatalf("lastPlayed = %v, want %v", got.LastPlayed, newer)
	}
}

func TestDeletePlaySessionAndRefreshGameRecalculatesLastPlayed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newTestRepo(t)
	game, _ := repo.CreateGame(ctx, newGame("Game", "/game.exe"))

	older := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	first, err := repo.CreatePlaySessionAndRefreshGame(ctx, domain.PlaySession{
		GameID: game.ID, PlayedAt: older, Duration: 100,
	})
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	second, err := repo.CreatePlaySessionAndRefreshGame(ctx, domain.PlaySession{
		GameID: game.ID, PlayedAt: newer, Duration: 50,
	})
	if err != nil {
		t.Fatalf("create second: %v", err)
	}

	gameID, err := repo.DeletePlaySessionAndRefreshGame(ctx, second.ID)
	if err != nil || gameID != game.ID {
		t.Fatalf("delete newest: gameID=%q err=%v", gameID, err)
	}

	got, _ := repo.GetGameByID(ctx, game.ID)
	if got.TotalPlayTime != 100 {
		t.Fatalf("totalPlayTime = %d, want 100", got.TotalPlayTime)
	}
	if got.LastPlayed == nil || !got.LastPlayed.Equal(older) {
		t.Fatalf("lastPlayed should fall back to older session, got %v", got.LastPlayed)
	}

	if _, err := repo.DeletePlaySessionAndRefreshGame(ctx, first.ID); err != nil {
		t.Fatalf("delete remaining: %v", err)
	}
	got, _ = repo.GetGameByID(ctx, game.ID)
	if got.TotalPlayTime != 0 || got.LastPlayed != nil {
		t.Fatalf("expected empty play cache, got total=%d last=%v", got.TotalPlayTime, got.LastPlayed)
	}
}

func TestCreatePlaySessionAndRefreshGameRollsBackOnGameUpdateFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo, conn := newTestRepoWithConnection(t)
	game, _ := repo.CreateGame(ctx, newGame("Game", "/game.exe"))

	if _, err := conn.ExecContext(ctx, `
		CREATE TRIGGER fail_game_playtime_update
		BEFORE UPDATE OF totalPlayTime ON "Game"
		BEGIN
			SELECT RAISE(ABORT, 'playtime update failed');
		END
	`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	created, err := repo.CreatePlaySessionAndRefreshGame(ctx, domain.PlaySession{
		GameID: game.ID, PlayedAt: time.Now().UTC(), Duration: 60,
	})
	if err == nil || created != nil {
		t.Fatalf("expected rollback failure, got created=%#v err=%v", created, err)
	}

	sessions, err := repo.ListPlaySessionsByGame(ctx, game.ID)
	if err != nil {
		t.Fatalf("ListPlaySessionsByGame: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("session insert was not rolled back: %#v", sessions)
	}
}

func TestCreatePlaySessionAndRefreshGameConcurrentSum(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newTestRepo(t)
	game, _ := repo.CreateGame(ctx, newGame("Game", "/game.exe"))

	const workers = 20
	const duration int64 = 7
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func(i int) {
			_, err := repo.CreatePlaySessionAndRefreshGame(ctx, domain.PlaySession{
				GameID:   game.ID,
				PlayedAt: time.Date(2026, 1, 1, 0, 0, i, 0, time.UTC),
				Duration: duration,
			})
			errCh <- err
		}(i)
	}
	for i := 0; i < workers; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("concurrent create: %v", err)
		}
	}

	got, err := repo.GetGameByID(ctx, game.ID)
	if err != nil || got == nil {
		t.Fatalf("GetGameByID: %v", err)
	}
	want := int64(workers) * duration
	if got.TotalPlayTime != want {
		t.Fatalf("totalPlayTime = %d, want %d", got.TotalPlayTime, want)
	}
	sessions, err := repo.ListPlaySessionsByGame(ctx, game.ID)
	if err != nil || int64(len(sessions)) != workers {
		t.Fatalf("sessions=%d err=%v", len(sessions), err)
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

func TestApplyPullResultDerivesPlayTimeFromSessionsNotGameJSON(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	ctx := context.Background()

	created, err := repo.CreateGame(ctx, newGame("PullGame", "/pull.exe"))
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	wrongLast := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	realLast := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	game := *created
	game.TotalPlayTime = 99999
	game.LastPlayed = &wrongLast

	sessions := []domain.PlaySession{
		{ID: "sess-a", GameID: created.ID, PlayedAt: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), Duration: 100, UpdatedAt: time.Now().UTC()},
		{ID: "sess-b", GameID: created.ID, PlayedAt: realLast, Duration: 40, UpdatedAt: time.Now().UTC()},
	}
	if err := repo.ApplyPullResult(ctx, game, sessions, "head-1", "{}"); err != nil {
		t.Fatalf("ApplyPullResult: %v", err)
	}

	got, err := repo.GetGameByID(ctx, created.ID)
	if err != nil || got == nil {
		t.Fatalf("GetGameByID: %v", err)
	}
	if got.TotalPlayTime != 140 {
		t.Fatalf("totalPlayTime = %d, want 140 from sessions", got.TotalPlayTime)
	}
	if got.LastPlayed == nil || !got.LastPlayed.Equal(realLast) {
		t.Fatalf("lastPlayed = %v, want %v", got.LastPlayed, realLast)
	}
}

func TestMigration0010PlaytimeAdjustmentIsIdempotent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo, conn := newTestRepoWithConnection(t)

	game, err := repo.CreateGame(ctx, newGame("Legacy", "/legacy.exe"))
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	// マイグレーション前の不整合を再現: セッション合計より大きい totalPlayTime。
	if _, err := conn.ExecContext(ctx, `
		UPDATE "Game" SET totalPlayTime = 1000, lastPlayed = ? WHERE id = ?
	`, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), game.ID); err != nil {
		t.Fatalf("seed game total: %v", err)
	}
	if _, err := repo.CreatePlaySession(ctx, domain.PlaySession{
		GameID: game.ID, PlayedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), Duration: 200,
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	insertAdjustment := `
INSERT INTO "PlaySession" (id, gameId, playedAt, duration, sessionName, routeId)
SELECT
  'h4adj-' || g.id,
  g.id,
  COALESCE(g.lastPlayed, g.createdAt, CURRENT_TIMESTAMP),
  g.totalPlayTime - COALESCE((
    SELECT SUM(ps.duration) FROM "PlaySession" ps WHERE ps.gameId = g.id
  ), 0),
  'プレイ時間の移行調整',
  NULL
FROM "Game" g
WHERE g.totalPlayTime > COALESCE((
  SELECT SUM(ps.duration) FROM "PlaySession" ps WHERE ps.gameId = g.id
), 0)
ON CONFLICT(id) DO NOTHING`
	refreshTotals := `
UPDATE "Game"
SET
  totalPlayTime = (
    SELECT COALESCE(SUM(duration), 0) FROM "PlaySession" WHERE gameId = "Game".id
  ),
  lastPlayed = (
    SELECT MAX(playedAt) FROM "PlaySession" WHERE gameId = "Game".id
  )`
	for i := 0; i < 2; i++ {
		if _, err := conn.ExecContext(ctx, insertAdjustment); err != nil {
			t.Fatalf("adjustment insert pass %d: %v", i+1, err)
		}
		if _, err := conn.ExecContext(ctx, refreshTotals); err != nil {
			t.Fatalf("adjustment refresh pass %d: %v", i+1, err)
		}
	}

	sessions, err := repo.ListPlaySessionsByGame(ctx, game.ID)
	if err != nil {
		t.Fatalf("ListPlaySessionsByGame: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions after idempotent adjustment, got %d", len(sessions))
	}
	var adjCount int
	var adjDuration int64
	for _, s := range sessions {
		if s.ID == "h4adj-"+game.ID {
			adjCount++
			adjDuration = s.Duration
			if s.SessionName == nil || *s.SessionName != "プレイ時間の移行調整" {
				t.Fatalf("unexpected adjustment name: %#v", s.SessionName)
			}
		}
	}
	if adjCount != 1 || adjDuration != 800 {
		t.Fatalf("adjustment session count=%d duration=%d", adjCount, adjDuration)
	}

	got, _ := repo.GetGameByID(ctx, game.ID)
	if got.TotalPlayTime != 1000 {
		t.Fatalf("totalPlayTime = %d, want 1000 preserved", got.TotalPlayTime)
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

	if err := repo.DeleteGameAndQueueMemoCleanup(ctx, game.ID); err != nil {
		t.Fatalf("DeleteGameAndQueueMemoCleanup: %v", err)
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

func TestSetLocalSyncStateUpdatesHeadAndTreeAtomically(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	ctx := context.Background()

	created, err := repo.CreateGame(ctx, newGame("SyncStateGame", "/sync-state.exe"))
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	if err := repo.SetLocalSyncState(ctx, created.ID, "fp-1", `{"files":{"a.sav":"h1"}}`); err != nil {
		t.Fatalf("SetLocalSyncState: %v", err)
	}

	got, err := repo.GetGameByID(ctx, created.ID)
	if err != nil || got == nil {
		t.Fatalf("GetGameByID: %v", err)
	}
	if got.LocalSyncHead == nil || *got.LocalSyncHead != "fp-1" {
		t.Fatalf("localSyncHead = %v, want fp-1", got.LocalSyncHead)
	}
	tree, err := repo.GetLocalSaveTree(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetLocalSaveTree: %v", err)
	}
	if tree != `{"files":{"a.sav":"h1"}}` {
		t.Fatalf("localSaveTree = %q", tree)
	}
}

func TestPendingPushLifecycleFinalizeClearsPending(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	ctx := context.Background()

	created, err := repo.CreateGame(ctx, newGame("PendingPushGame", "/pending.exe"))
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}

	pending := domain.PendingPush{
		GameID:             created.ID,
		ExpectedRemoteHead: "old-head",
		NewCommitHash:      "new-commit",
		ContentFingerprint: "fp-new",
		SaveTree:           `{"files":{"b.sav":"h2"}}`,
	}
	if err := repo.BeginPendingPush(ctx, pending); err != nil {
		t.Fatalf("BeginPendingPush: %v", err)
	}
	listed, err := repo.ListPendingPushes(ctx)
	if err != nil {
		t.Fatalf("ListPendingPushes: %v", err)
	}
	if len(listed) != 1 || listed[0].NewCommitHash != "new-commit" {
		t.Fatalf("listed pending = %#v", listed)
	}

	if err := repo.FinalizePendingPush(ctx, created.ID, pending.ContentFingerprint, pending.SaveTree); err != nil {
		t.Fatalf("FinalizePendingPush: %v", err)
	}

	got, err := repo.GetGameByID(ctx, created.ID)
	if err != nil || got == nil {
		t.Fatalf("GetGameByID: %v", err)
	}
	if got.LocalSyncHead == nil || *got.LocalSyncHead != "fp-new" {
		t.Fatalf("localSyncHead = %v, want fp-new", got.LocalSyncHead)
	}
	tree, err := repo.GetLocalSaveTree(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetLocalSaveTree: %v", err)
	}
	if tree != pending.SaveTree {
		t.Fatalf("localSaveTree = %q, want %q", tree, pending.SaveTree)
	}
	listed, err = repo.ListPendingPushes(ctx)
	if err != nil {
		t.Fatalf("ListPendingPushes after finalize: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("pending should be cleared, got %#v", listed)
	}
}

func TestClearPendingPushDoesNotTouchBaseline(t *testing.T) {
	t.Parallel()
	repo := newTestRepo(t)
	ctx := context.Background()

	created, err := repo.CreateGame(ctx, newGame("ClearPendingGame", "/clear-pending.exe"))
	if err != nil {
		t.Fatalf("CreateGame: %v", err)
	}
	if err := repo.SetLocalSyncState(ctx, created.ID, "fp-keep", `{"files":{}}`); err != nil {
		t.Fatalf("SetLocalSyncState: %v", err)
	}
	if err := repo.BeginPendingPush(ctx, domain.PendingPush{
		GameID:             created.ID,
		ExpectedRemoteHead: "",
		NewCommitHash:      "abandoned",
		ContentFingerprint: "fp-abandoned",
		SaveTree:           `{"files":{"x":"y"}}`,
	}); err != nil {
		t.Fatalf("BeginPendingPush: %v", err)
	}

	if err := repo.ClearPendingPush(ctx, created.ID); err != nil {
		t.Fatalf("ClearPendingPush: %v", err)
	}

	got, err := repo.GetGameByID(ctx, created.ID)
	if err != nil || got == nil {
		t.Fatalf("GetGameByID: %v", err)
	}
	if got.LocalSyncHead == nil || *got.LocalSyncHead != "fp-keep" {
		t.Fatalf("baseline should remain fp-keep, got %v", got.LocalSyncHead)
	}
	listed, err := repo.ListPendingPushes(ctx)
	if err != nil {
		t.Fatalf("ListPendingPushes: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("pending should be cleared, got %#v", listed)
	}
}
