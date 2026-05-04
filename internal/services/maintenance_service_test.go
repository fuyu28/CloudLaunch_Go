package services

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
)

type maintenanceTestRuntime struct {
	cfg        config.Config
	repository *db.Repository
	service    *MaintenanceService
}

func TestMaintenanceServiceExportGameDataWritesArtifacts(t *testing.T) {
	t.Parallel()

	runtime := newMaintenanceServiceRuntime(t)
	game, sessions := seedMaintenanceFixture(t, runtime.repository)

	outputDir := t.TempDir()
	exported, err := runtime.service.ExportGameData(context.Background(), outputDir)
	if err != nil {
		t.Fatalf("ExportGameData failed: %v", err)
	}

	jsonBytes, err := os.ReadFile(exported.JSONPath)
	if err != nil {
		t.Fatalf("failed to read export json: %v", err)
	}

	var payload GameExportPayload
	if err := json.Unmarshal(jsonBytes, &payload); err != nil {
		t.Fatalf("failed to unmarshal export json: %v", err)
	}
	if len(payload.Games) != 1 || payload.Games[0].ID != game.ID {
		t.Fatalf("unexpected exported games: %#v", payload.Games)
	}
	if len(payload.SessionRows) != len(sessions) {
		t.Fatalf("expected %d sessions, got %d", len(sessions), len(payload.SessionRows))
	}
	if len(payload.Statistics) != 1 {
		t.Fatalf("expected 1 statistic row, got %d", len(payload.Statistics))
	}

	stat := payload.Statistics[0]
	if stat.SessionCount != 2 || stat.TotalSessionDuration != 5400 || stat.AverageSessionDuration != 2700 {
		t.Fatalf("unexpected statistic row: %#v", stat)
	}
	if stat.LastSessionAt == nil || !stat.LastSessionAt.Equal(sessions[1].PlayedAt) {
		t.Fatalf("expected latest session time %v, got %v", sessions[1].PlayedAt, stat.LastSessionAt)
	}

	csvFile, err := os.Open(exported.CSVPath)
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
	if len(rows) != 2 || rows[1][0] != game.ID || rows[1][8] != "2" || rows[1][9] != "5400" || rows[1][10] != "2700.00" {
		t.Fatalf("unexpected csv rows: %#v", rows)
	}
}

func TestMaintenanceServiceCreateFullBackupCapturesDatabaseAndFiles(t *testing.T) {
	t.Parallel()

	runtime := newMaintenanceServiceRuntime(t)
	game, _ := seedMaintenanceFixture(t, runtime.repository)
	writeMaintenanceFile(t, filepath.Join(runtime.cfg.AppDataDir, "notes", "readme.txt"), "keep me")

	backupResult, err := runtime.service.CreateFullBackup(t.TempDir())
	if err != nil {
		t.Fatalf("CreateFullBackup failed: %v", err)
	}

	restoreRoot := t.TempDir()
	if err := UnzipToDirectory(backupResult, restoreRoot); err != nil {
		t.Fatalf("failed to unzip backup: %v", err)
	}

	manifest, err := ReadBackupManifest(restoreRoot)
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}
	if manifest.DatabaseRelativePath != filepath.Base(runtime.cfg.DatabasePath) {
		t.Fatalf("unexpected databaseRelativePath: %q", manifest.DatabaseRelativePath)
	}

	assertMaintenanceFileContent(t, filepath.Join(restoreRoot, "notes", "readme.txt"), "keep me")

	connection, err := db.Open(filepath.Join(restoreRoot, manifest.DatabaseRelativePath))
	if err != nil {
		t.Fatalf("failed to open backup db: %v", err)
	}
	defer func() {
		_ = connection.Close()
	}()

	repository := db.NewRepository(connection)
	games, err := repository.ListGames(context.Background(), "", models.PlayStatus(""), "title", "asc")
	if err != nil {
		t.Fatalf("failed to list backup games: %v", err)
	}
	if len(games) != 1 || games[0].ID != game.ID {
		t.Fatalf("unexpected backup games: %#v", games)
	}
}

func TestMaintenanceServiceRestoreFullBackupReplacesAppData(t *testing.T) {
	t.Parallel()

	source := newMaintenanceServiceRuntime(t)
	game, _ := seedMaintenanceFixture(t, source.repository)
	writeMaintenanceFile(t, filepath.Join(source.cfg.AppDataDir, "screenshots", "latest.txt"), "from backup")

	backupResult, err := source.service.CreateFullBackup(t.TempDir())
	if err != nil {
		t.Fatalf("CreateFullBackup failed: %v", err)
	}

	target := newMaintenanceServiceRuntime(t)
	writeMaintenanceFile(t, filepath.Join(target.cfg.AppDataDir, "obsolete.txt"), "remove me")
	createMaintenanceGame(t, target.repository, models.Game{
		Title:      "Old Game",
		Publisher:  "Legacy",
		ExePath:    "/games/old.exe",
		PlayStatus: models.PlayStatusUnplayed,
	})

	if err := target.service.RestoreFullBackup(backupResult); err != nil {
		t.Fatalf("RestoreFullBackup failed: %v", err)
	}

	connection, err := db.Open(target.cfg.DatabasePath)
	if err != nil {
		t.Fatalf("failed to reopen restored db: %v", err)
	}
	defer func() {
		_ = connection.Close()
	}()
	repository := db.NewRepository(connection)
	games, err := repository.ListGames(context.Background(), "", models.PlayStatus(""), "title", "asc")
	if err != nil {
		t.Fatalf("failed to list restored games: %v", err)
	}
	if len(games) != 1 || games[0].ID != game.ID {
		t.Fatalf("unexpected restored games: %#v", games)
	}

	assertMaintenanceFileContent(t, filepath.Join(target.cfg.AppDataDir, "screenshots", "latest.txt"), "from backup")
	if _, err := os.Stat(filepath.Join(target.cfg.AppDataDir, "obsolete.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected obsolete file to be removed, stat err=%v", err)
	}
}

func TestUnzipToDirectoryRejectsPathTraversalArchive(t *testing.T) {
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

	if err := UnzipToDirectory(zipPath, t.TempDir()); err == nil {
		t.Fatal("expected path traversal archive to be rejected")
	}
}

func TestReadBackupManifestRejectsUnsupportedBackupType(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	data, err := json.Marshal(BackupManifest{
		CloudLaunchBackupType: "unknown",
		BackupVersion:         1,
		DatabaseRelativePath:  "app.db",
	})
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "_manifest.json"), data, 0o600); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	if _, err := ReadBackupManifest(root); err == nil {
		t.Fatal("expected unsupported backup type to fail")
	}
}

func newMaintenanceServiceRuntime(t *testing.T) *maintenanceTestRuntime {
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
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := config.Config{
		AppDataDir:   appDataDir,
		DatabasePath: databasePath,
	}

	runtime := &maintenanceTestRuntime{
		cfg:        cfg,
		repository: repository,
	}

	runtime.service = NewMaintenanceService(cfg, repository, logger, MaintenanceRuntimeHooks{
		CreateDatabaseSnapshot: func(destinationPath string) error {
			return CopyFilePath(databasePath, destinationPath)
		},
		StopRuntimeServices: func() {},
		CloseDatabaseConnection: func() error {
			return connection.Close()
		},
		ReopenDatabaseAndServices: func() error {
			reopened, err := db.Open(databasePath)
			if err != nil {
				return err
			}
			if err := db.ApplyMigrations(reopened); err != nil {
				_ = reopened.Close()
				return err
			}
			connection = reopened
			runtime.repository = db.NewRepository(reopened)
			runtime.service.repository = runtime.repository
			return nil
		},
		ResumeRuntimeServices: func() error { return nil },
	})

	t.Cleanup(func() {
		_ = connection.Close()
	})

	return runtime
}

func seedMaintenanceFixture(t *testing.T, repository *db.Repository) (*models.Game, []models.PlaySession) {
	t.Helper()

	lastPlayed := time.Date(2026, 4, 28, 21, 0, 0, 0, time.UTC)
	game := createMaintenanceGame(t, repository, models.Game{
		Title:         "Test Game",
		Publisher:     "Test Publisher",
		ExePath:       "/games/test.exe",
		PlayStatus:    models.PlayStatusPlaying,
		TotalPlayTime: 5400,
		LastPlayed:    &lastPlayed,
	})

	firstPlayedAt := time.Date(2026, 4, 27, 20, 0, 0, 0, time.UTC)
	secondPlayedAt := time.Date(2026, 4, 28, 21, 0, 0, 0, time.UTC)
	session1 := createMaintenanceSession(t, repository, models.PlaySession{
		GameID:   game.ID,
		PlayedAt: firstPlayedAt,
		Duration: 1800,
	})
	session2 := createMaintenanceSession(t, repository, models.PlaySession{
		GameID:   game.ID,
		PlayedAt: secondPlayedAt,
		Duration: 3600,
	})

	return game, []models.PlaySession{*session1, *session2}
}

func createMaintenanceGame(t *testing.T, repository *db.Repository, game models.Game) *models.Game {
	t.Helper()

	created, err := repository.CreateGame(context.Background(), game)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}
	return created
}

func createMaintenanceSession(t *testing.T, repository *db.Repository, session models.PlaySession) *models.PlaySession {
	t.Helper()

	created, err := repository.CreatePlaySession(context.Background(), session)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	return created
}

func writeMaintenanceFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

func assertMaintenanceFileContent(t *testing.T, path string, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != want {
		t.Fatalf("expected %q, got %q", want, string(data))
	}
}
