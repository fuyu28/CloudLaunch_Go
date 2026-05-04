package app

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/db"
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/services"
)

func TestAppExportGameData_WritesUserVisibleArtifacts(t *testing.T) {
	t.Parallel()

	app, repository := newMaintenanceTestApp(t)
	game, sessions := seedExportFixture(t, repository)

	outputDir := t.TempDir()
	exported := app.ExportGameData(outputDir)
	if !exported.Success {
		t.Fatalf("ExportGameData failed: %#v", exported.Error)
	}

	if exported.Data.JSONPath == "" {
		t.Fatal("expected JSONPath to be set")
	}
	if exported.Data.CSVPath == "" {
		t.Fatal("expected CSVPath to be set")
	}

	jsonBytes, err := os.ReadFile(exported.Data.JSONPath)
	if err != nil {
		t.Fatalf("failed to read export json: %v", err)
	}

	var payload services.GameExportPayload
	if err := json.Unmarshal(jsonBytes, &payload); err != nil {
		t.Fatalf("failed to unmarshal export json: %v", err)
	}

	if len(payload.Games) != 1 {
		t.Fatalf("expected 1 game in export payload, got %d", len(payload.Games))
	}
	if payload.Games[0].ID != game.ID {
		t.Fatalf("expected exported game id %q, got %q", game.ID, payload.Games[0].ID)
	}
	if len(payload.Routes) != 1 || payload.Routes[0].GameID != game.ID {
		t.Fatalf("expected 1 route in export payload for %q, got %#v", game.ID, payload.Routes)
	}
	if len(payload.SessionRows) != len(sessions) {
		t.Fatalf("expected %d sessions in export payload, got %d", len(sessions), len(payload.SessionRows))
	}
	if len(payload.Statistics) != 1 {
		t.Fatalf("expected 1 statistics row, got %d", len(payload.Statistics))
	}

	stat := payload.Statistics[0]
	if stat.GameID != game.ID {
		t.Fatalf("expected statistics game id %q, got %q", game.ID, stat.GameID)
	}
	if stat.SessionCount != 2 {
		t.Fatalf("expected session count 2, got %d", stat.SessionCount)
	}
	if stat.TotalSessionDuration != 5400 {
		t.Fatalf("expected total session duration 5400, got %d", stat.TotalSessionDuration)
	}
	if stat.AverageSessionDuration != 2700 {
		t.Fatalf("expected average session duration 2700, got %f", stat.AverageSessionDuration)
	}
	if stat.LastSessionAt == nil || !stat.LastSessionAt.Equal(sessions[1].PlayedAt) {
		t.Fatalf("expected latest session time %v, got %v", sessions[1].PlayedAt, stat.LastSessionAt)
	}

	csvFile, err := os.Open(exported.Data.CSVPath)
	if err != nil {
		t.Fatalf("failed to open export csv: %v", err)
	}
	defer func() {
		_ = csvFile.Close()
	}()

	rows, err := csv.NewReader(csvFile).ReadAll()
	if err != nil {
		t.Fatalf("failed to read export csv: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected header + 1 row in csv, got %d rows", len(rows))
	}
	if rows[1][0] != game.ID {
		t.Fatalf("expected csv game id %q, got %q", game.ID, rows[1][0])
	}
	if rows[1][8] != "2" {
		t.Fatalf("expected csv session count 2, got %q", rows[1][8])
	}
	if rows[1][9] != "5400" {
		t.Fatalf("expected csv total session duration 5400, got %q", rows[1][9])
	}
	if rows[1][10] != "2700.00" {
		t.Fatalf("expected csv average session duration 2700.00, got %q", rows[1][10])
	}
}

func TestAppCreateFullBackup_CapturesDatabaseAndFiles(t *testing.T) {
	t.Parallel()

	app, repository := newMaintenanceTestApp(t)
	game, _ := seedExportFixture(t, repository)
	writeTestFile(t, filepath.Join(app.Config.AppDataDir, "notes", "readme.txt"), "keep me")

	outputDir := t.TempDir()
	backupResult := app.CreateFullBackup(outputDir)
	if !backupResult.Success {
		t.Fatalf("CreateFullBackup failed: %#v", backupResult.Error)
	}

	restoreRoot := t.TempDir()
	if err := services.UnzipToDirectory(backupResult.Data, restoreRoot); err != nil {
		t.Fatalf("failed to unzip backup: %v", err)
	}

	manifest, err := services.ReadBackupManifest(restoreRoot)
	if err != nil {
		t.Fatalf("failed to read backup manifest: %v", err)
	}
	if manifest.DatabaseRelativePath != filepath.Base(app.Config.DatabasePath) {
		t.Fatalf("expected database relative path %q, got %q", filepath.Base(app.Config.DatabasePath), manifest.DatabaseRelativePath)
	}

	assertFileContent(t, filepath.Join(restoreRoot, "notes", "readme.txt"), "keep me")

	backupDBPath := filepath.Join(restoreRoot, manifest.DatabaseRelativePath)
	connection, err := db.Open(backupDBPath)
	if err != nil {
		t.Fatalf("failed to open backed up database: %v", err)
	}
	defer func() {
		_ = connection.Close()
	}()

	backupRepository := db.NewRepository(connection)
	games, err := backupRepository.ListGames(context.Background(), "", models.PlayStatus(""), "title", "asc")
	if err != nil {
		t.Fatalf("failed to list games from backup database: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("expected 1 game in backup database, got %d", len(games))
	}
	if games[0].ID != game.ID {
		t.Fatalf("expected backed up game id %q, got %q", game.ID, games[0].ID)
	}
}

func TestAppRestoreFullBackup_ReplacesAppDataWithBackupContents(t *testing.T) {
	t.Parallel()

	sourceApp, sourceRepository := newMaintenanceTestApp(t)
	game, _ := seedExportFixture(t, sourceRepository)
	writeTestFile(t, filepath.Join(sourceApp.Config.AppDataDir, "screenshots", "latest.txt"), "from backup")

	backupPath := sourceApp.CreateFullBackup(t.TempDir())
	if !backupPath.Success {
		t.Fatalf("CreateFullBackup failed: %#v", backupPath.Error)
	}

	targetApp, targetRepository := newMaintenanceTestApp(t)
	writeTestFile(t, filepath.Join(targetApp.Config.AppDataDir, "obsolete.txt"), "remove me")
	createGameForTest(t, targetRepository, models.Game{
		Title:      "Old Game",
		Publisher:  "Legacy",
		ExePath:    "/games/old.exe",
		PlayStatus: models.PlayStatusUnplayed,
	})

	restored := targetApp.RestoreFullBackup(backupPath.Data)
	if !restored.Success {
		t.Fatalf("RestoreFullBackup failed: %#v", restored.Error)
	}

	games, err := targetApp.GameService.ListGames(context.Background(), "", models.PlayStatus(""), "title", "asc")
	if err != nil {
		t.Fatalf("failed to list restored games: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("expected 1 restored game, got %d", len(games))
	}
	if games[0].ID != game.ID {
		t.Fatalf("expected restored game id %q, got %q", game.ID, games[0].ID)
	}

	assertFileContent(t, filepath.Join(targetApp.Config.AppDataDir, "screenshots", "latest.txt"), "from backup")
	if _, err := os.Stat(filepath.Join(targetApp.Config.AppDataDir, "obsolete.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected obsolete file to be removed during restore, stat err=%v", err)
	}
}

func TestUnzipToDirectory_RejectsPathTraversalArchive(t *testing.T) {
	t.Parallel()

	zipPath := filepath.Join(t.TempDir(), "malicious.zip")
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}

	writer := zip.NewWriter(file)
	entry, err := writer.Create("../evil.txt")
	if err != nil {
		t.Fatalf("failed to create zip entry: %v", err)
	}
	if _, err := entry.Write([]byte("malicious")); err != nil {
		t.Fatalf("failed to write zip entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("failed to close zip file: %v", err)
	}

	err = services.UnzipToDirectory(zipPath, t.TempDir())
	if err == nil {
		t.Fatal("expected unzipToDirectory to reject path traversal archive")
	}
}

func newMaintenanceTestApp(t *testing.T) (*App, *db.Repository) {
	t.Helper()

	appDataDir := t.TempDir()
	databasePath := filepath.Join(appDataDir, "app.db")
	connection, err := db.Open(databasePath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.ApplyMigrations(connection); err != nil {
		_ = connection.Close()
		t.Fatalf("failed to apply migrations: %v", err)
	}

	repository := db.NewRepository(connection)
	app := &App{
		Config: config.Config{
			AppDataDir:   appDataDir,
			DatabasePath: databasePath,
		},
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		dbConnection: connection,
		autoTracking: true,
	}
	app.configureServices(repository, newCredentialStore(app.Config))

	t.Cleanup(func() {
		if app.dbConnection != nil {
			_ = app.dbConnection.Close()
			app.dbConnection = nil
		}
	})

	return app, repository
}

func seedExportFixture(t *testing.T, repository *db.Repository) (*models.Game, []models.PlaySession) {
	t.Helper()

	lastPlayed := time.Date(2026, 4, 28, 21, 0, 0, 0, time.UTC)
	game := createGameForTest(t, repository, models.Game{
		Title:         "Test Game",
		Publisher:     "Test Publisher",
		ExePath:       "/games/test.exe",
		PlayStatus:    models.PlayStatusPlaying,
		TotalPlayTime: 5400,
		LastPlayed:    &lastPlayed,
	})
	route := createRouteForTest(t, repository, models.PlayRoute{
		GameID:    game.ID,
		Name:      "Common",
		SortOrder: 0,
	})

	firstPlayedAt := time.Date(2026, 4, 27, 20, 0, 0, 0, time.UTC)
	secondPlayedAt := time.Date(2026, 4, 28, 21, 0, 0, 0, time.UTC)
	session1 := createSessionForTest(t, repository, models.PlaySession{
		GameID:      game.ID,
		PlayRouteID: &route.ID,
		PlayedAt:    firstPlayedAt,
		Duration:    1800,
	})
	session2 := createSessionForTest(t, repository, models.PlaySession{
		GameID:   game.ID,
		PlayedAt: secondPlayedAt,
		Duration: 3600,
	})

	return game, []models.PlaySession{*session1, *session2}
}

func createGameForTest(t *testing.T, repository *db.Repository, game models.Game) *models.Game {
	t.Helper()

	created, err := repository.CreateGame(context.Background(), game)
	if err != nil {
		t.Fatalf("failed to create test game: %v", err)
	}
	return created
}

func createSessionForTest(t *testing.T, repository *db.Repository, session models.PlaySession) *models.PlaySession {
	t.Helper()

	created, err := repository.CreatePlaySession(context.Background(), session)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	return created
}

func createRouteForTest(t *testing.T, repository *db.Repository, route models.PlayRoute) *models.PlayRoute {
	t.Helper()

	created, err := repository.CreatePlayRoute(context.Background(), route)
	if err != nil {
		t.Fatalf("failed to create test route: %v", err)
	}
	return created
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("failed to create directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("expected %s to contain %q, got %q", path, want, string(data))
	}
}
