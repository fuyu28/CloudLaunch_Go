// @fileoverview データエクスポートとバックアップ復元APIを提供する。
package app

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"CloudLaunch_Go/internal/db"
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/services"
)

const (
	backupTypeV1 = "appdata-zip-v1"
)

// GameExportStatistic はゲーム単位の統計情報を表す。
type GameExportStatistic struct {
	GameID                 string     `json:"gameId"`
	Title                  string     `json:"title"`
	SessionCount           int        `json:"sessionCount"`
	TotalSessionDuration   int64      `json:"totalSessionDuration"`
	AverageSessionDuration float64    `json:"averageSessionDuration"`
	LastSessionAt          *time.Time `json:"lastSessionAt,omitempty"`
}

// GameExportPayload はJSONエクスポート内容を表す。
type GameExportPayload struct {
	ExportedAt  time.Time             `json:"exportedAt"`
	Games       []models.Game         `json:"games"`
	Statistics  []GameExportStatistic `json:"statistics"`
	SessionRows []models.PlaySession  `json:"sessions"`
}

// GameExportResult はデータエクスポートの出力情報を表す。
type GameExportResult struct {
	JSONPath string `json:"jsonPath"`
	CSVPath  string `json:"csvPath"`
}

// BackupManifest はバックアップのメタ情報を表す。
type BackupManifest struct {
	CreatedAt             time.Time `json:"createdAt"`
	AppDataDir            string    `json:"appDataDir"`
	DatabaseRelativePath  string    `json:"databaseRelativePath"`
	CredentialNotice      string    `json:"credentialNotice"`
	CloudLaunchBackupType string    `json:"cloudLaunchBackupType"`
	BackupVersion         int       `json:"backupVersion"`
}

// ExportGameData はゲーム情報・統計データをCSV/JSONで出力する。
func (app *App) ExportGameData(outputDir string) result.ApiResult[GameExportResult] {
	trimmed := strings.TrimSpace(outputDir)
	if trimmed == "" {
		return result.ErrorResult[GameExportResult]("出力先フォルダが不正です", "outputDir is empty")
	}
	if err := os.MkdirAll(trimmed, 0o700); err != nil {
		return errorResultWithLog[GameExportResult](app, "出力先フォルダの作成に失敗しました", err, "operation", "ExportGameData.mkdir", "outputDir", trimmed)
	}

	ctx := app.context()
	games, err := app.Database.ListGames(ctx, "", models.PlayStatus(""), "title", "asc")
	if err != nil {
		return errorResultWithLog[GameExportResult](app, "ゲーム一覧の取得に失敗しました", err, "operation", "ExportGameData.listGames")
	}

	stats := make([]GameExportStatistic, 0, len(games))
	sessionRows := make([]models.PlaySession, 0, len(games)*2)
	for _, game := range games {
		sessions, err := app.Database.ListPlaySessionsByGame(ctx, game.ID)
		if err != nil {
			return errorResultWithLog[GameExportResult](app, "セッション取得に失敗しました", err, "operation", "ExportGameData.listSessions", "gameId", game.ID)
		}
		sessionRows = append(sessionRows, sessions...)

		var total int64
		for _, session := range sessions {
			total += session.Duration
		}
		average := float64(0)
		if len(sessions) > 0 {
			average = float64(total) / float64(len(sessions))
		}
		var lastSessionAt *time.Time
		if len(sessions) > 0 {
			last := sessions[0].PlayedAt
			lastSessionAt = &last
		}
		stats = append(stats, GameExportStatistic{
			GameID:                 game.ID,
			Title:                  game.Title,
			SessionCount:           len(sessions),
			TotalSessionDuration:   total,
			AverageSessionDuration: average,
			LastSessionAt:          lastSessionAt,
		})
	}

	now := time.Now()
	stamp := now.Format("20060102_150405")
	jsonPath := filepath.Join(trimmed, fmt.Sprintf("cloudlaunch_export_%s.json", stamp))
	csvPath := filepath.Join(trimmed, fmt.Sprintf("cloudlaunch_export_%s.csv", stamp))

	payload := GameExportPayload{
		ExportedAt:  now,
		Games:       games,
		Statistics:  stats,
		SessionRows: sessionRows,
	}
	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return errorResultWithLog[GameExportResult](app, "JSONの生成に失敗しました", err, "operation", "ExportGameData.marshal")
	}
	if err := os.WriteFile(jsonPath, jsonData, 0o600); err != nil {
		return errorResultWithLog[GameExportResult](app, "JSONファイルの保存に失敗しました", err, "operation", "ExportGameData.writeJSON", "path", jsonPath)
	}

	if err := writeExportCSV(csvPath, games, stats); err != nil {
		return errorResultWithLog[GameExportResult](app, "CSVファイルの保存に失敗しました", err, "operation", "ExportGameData.writeCSV", "path", csvPath)
	}

	return result.OkResult(GameExportResult{JSONPath: jsonPath, CSVPath: csvPath})
}

// CreateFullBackup はアプリデータ一式のバックアップZIPを作成する。
func (app *App) CreateFullBackup(outputDir string) result.ApiResult[string] {
	trimmed := strings.TrimSpace(outputDir)
	if trimmed == "" {
		return result.ErrorResult[string]("出力先フォルダが不正です", "outputDir is empty")
	}
	if err := os.MkdirAll(trimmed, 0o700); err != nil {
		return errorResultWithLog[string](app, "出力先フォルダの作成に失敗しました", err, "operation", "CreateFullBackup.mkdir", "outputDir", trimmed)
	}

	appDataDir := strings.TrimSpace(app.Config.AppDataDir)
	if appDataDir == "" {
		return result.ErrorResult[string]("バックアップ元ディレクトリが不正です", "AppDataDir is empty")
	}

	relDBPath, err := filepath.Rel(appDataDir, app.Config.DatabasePath)
	if err != nil {
		return errorResultWithLog[string](app, "DB相対パスの解決に失敗しました", err, "operation", "CreateFullBackup.relDBPath")
	}
	if strings.HasPrefix(relDBPath, "..") {
		return result.ErrorResult[string]("バックアップ対象DBが不正です", "database path is outside AppDataDir")
	}

	stagingDir, err := os.MkdirTemp("", "cloudlaunch-backup-")
	if err != nil {
		return errorResultWithLog[string](app, "バックアップ準備に失敗しました", err, "operation", "CreateFullBackup.mktemp")
	}
	defer func() {
		_ = os.RemoveAll(stagingDir)
	}()

	if err := copyDirectoryTree(appDataDir, stagingDir); err != nil {
		return errorResultWithLog[string](app, "バックアップ準備に失敗しました", err, "operation", "CreateFullBackup.copyAppData")
	}

	snapshotPath := filepath.Join(stagingDir, relDBPath)
	if err := app.createDatabaseSnapshot(snapshotPath); err != nil {
		return errorResultWithLog[string](app, "DBスナップショットの取得に失敗しました", err, "operation", "CreateFullBackup.snapshot")
	}
	_ = os.Remove(snapshotPath + "-wal")
	_ = os.Remove(snapshotPath + "-shm")

	stamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(trimmed, fmt.Sprintf("cloudlaunch_backup_%s.zip", stamp))
	manifest := BackupManifest{
		CreatedAt:             time.Now(),
		AppDataDir:            appDataDir,
		DatabaseRelativePath:  filepath.ToSlash(relDBPath),
		CredentialNotice:      "OS credential store (Windows Credential Manager) is not included.",
		CloudLaunchBackupType: backupTypeV1,
		BackupVersion:         1,
	}
	if err := writeBackupZip(stagingDir, backupPath, manifest); err != nil {
		return errorResultWithLog[string](app, "バックアップ作成に失敗しました", err, "operation", "CreateFullBackup.writeZip", "path", backupPath)
	}
	return result.OkResult(backupPath)
}

// RestoreFullBackup はバックアップZIPから全データを復元する。
func (app *App) RestoreFullBackup(backupPath string) result.ApiResult[bool] {
	trimmed := strings.TrimSpace(backupPath)
	if trimmed == "" {
		return result.ErrorResult[bool]("バックアップファイルが不正です", "backupPath is empty")
	}
	if _, err := os.Stat(trimmed); err != nil {
		if os.IsNotExist(err) {
			return result.ErrorResult[bool]("バックアップファイルが見つかりません", err.Error())
		}
		return errorResultWithLog[bool](app, "バックアップファイルの確認に失敗しました", err, "operation", "RestoreFullBackup.stat", "path", trimmed)
	}

	tmpDir, err := os.MkdirTemp("", "cloudlaunch-restore-")
	if err != nil {
		return errorResultWithLog[bool](app, "復元用一時ディレクトリの作成に失敗しました", err, "operation", "RestoreFullBackup.mktemp")
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	if err := unzipToDirectory(trimmed, tmpDir); err != nil {
		return errorResultWithLog[bool](app, "バックアップ展開に失敗しました", err, "operation", "RestoreFullBackup.unzip", "path", trimmed)
	}

	if err := app.restoreAppDataFrom(tmpDir); err != nil {
		return errorResultWithLog[bool](app, "バックアップ復元に失敗しました", err, "operation", "RestoreFullBackup.restore")
	}

	return result.OkResult(true)
}

func writeExportCSV(path string, games []models.Game, stats []GameExportStatistic) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	writer := csv.NewWriter(file)
	if err := writer.Write([]string{
		"gameId",
		"title",
		"publisher",
		"playStatus",
		"totalPlayTime",
		"lastPlayed",
		"createdAt",
		"updatedAt",
		"sessionCount",
		"totalSessionDuration",
		"averageSessionDuration",
		"lastSessionAt",
	}); err != nil {
		return err
	}

	statMap := make(map[string]GameExportStatistic, len(stats))
	for _, stat := range stats {
		statMap[stat.GameID] = stat
	}

	for _, game := range games {
		stat := statMap[game.ID]
		row := []string{
			game.ID,
			game.Title,
			game.Publisher,
			string(game.PlayStatus),
			fmt.Sprintf("%d", game.TotalPlayTime),
			formatTimePtr(game.LastPlayed),
			game.CreatedAt.Format(time.RFC3339),
			game.UpdatedAt.Format(time.RFC3339),
			fmt.Sprintf("%d", stat.SessionCount),
			fmt.Sprintf("%d", stat.TotalSessionDuration),
			fmt.Sprintf("%.2f", stat.AverageSessionDuration),
			formatTimePtr(stat.LastSessionAt),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func formatTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func writeBackupZip(sourceRoot string, outputPath string, manifest BackupManifest) error {
	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	zipWriter := zip.NewWriter(file)
	defer func() {
		_ = zipWriter.Close()
	}()

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := writeZipBytes(zipWriter, "_manifest.json", manifestBytes); err != nil {
		return err
	}

	return filepath.WalkDir(sourceRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relPath)
		if name == "" || name == "_manifest.json" || strings.HasPrefix(name, "../") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = name
		header.Method = zip.Deflate

		dest, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		return copyFileContent(path, dest)
	})
}

func writeZipBytes(writer *zip.Writer, name string, payload []byte) error {
	dest, err := writer.Create(name)
	if err != nil {
		return err
	}
	_, err = dest.Write(payload)
	return err
}

func unzipToDirectory(zipPath string, destRoot string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = reader.Close()
	}()

	for _, file := range reader.File {
		cleanName := filepath.Clean(file.Name)
		if cleanName == "." || cleanName == "" {
			continue
		}
		destPath := filepath.Join(destRoot, cleanName)
		if !strings.HasPrefix(destPath, filepath.Clean(destRoot)+string(os.PathSeparator)) {
			return errors.New("invalid backup archive path")
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, 0o700); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o700); err != nil {
			return err
		}

		src, err := file.Open()
		if err != nil {
			return err
		}

		dst, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			_ = src.Close()
			return err
		}

		if _, err := io.Copy(dst, src); err != nil {
			_ = dst.Close()
			_ = src.Close()
			return err
		}
		_ = dst.Close()
		_ = src.Close()
	}
	return nil
}

func readBackupManifest(extractedRoot string) (*BackupManifest, error) {
	manifestPath := filepath.Join(extractedRoot, "_manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("backup manifest not found")
		}
		return nil, err
	}
	var manifest BackupManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	if manifest.CloudLaunchBackupType != backupTypeV1 {
		return nil, errors.New("unsupported backup type")
	}
	if manifest.BackupVersion != 0 && manifest.BackupVersion != 1 {
		return nil, errors.New("unsupported backup version")
	}
	if strings.TrimSpace(manifest.DatabaseRelativePath) == "" {
		return nil, errors.New("databaseRelativePath is empty")
	}
	return &manifest, nil
}

func sanitizeRelativePath(pathValue string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(strings.TrimSpace(pathValue)))
	if cleaned == "." || cleaned == "" {
		return "", errors.New("relative path is empty")
	}
	if filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "..") {
		return "", errors.New("relative path is invalid")
	}
	return cleaned, nil
}

func (app *App) restoreAppDataFrom(extractedRoot string) (restoreErr error) {
	appDataDir := strings.TrimSpace(app.Config.AppDataDir)
	if appDataDir == "" {
		return errors.New("appDataDir is empty")
	}

	manifest, err := readBackupManifest(extractedRoot)
	if err != nil {
		return err
	}
	relDBPath, err := sanitizeRelativePath(manifest.DatabaseRelativePath)
	if err != nil {
		return err
	}
	backupDBPath := filepath.Join(extractedRoot, relDBPath)
	if _, err := os.Stat(backupDBPath); err != nil {
		if os.IsNotExist(err) {
			return errors.New("backup database not found")
		}
		return err
	}

	rollbackDir, err := os.MkdirTemp("", "cloudlaunch-rollback-")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(rollbackDir)
	}()

	hasCurrentData, err := directoryHasAnyEntry(appDataDir)
	if err != nil {
		return err
	}
	if hasCurrentData {
		if err := copyDirectoryTree(appDataDir, rollbackDir); err != nil {
			return err
		}
	}

	app.stopRuntimeServicesForRestore()
	if app.dbConnection != nil {
		if err := app.dbConnection.Close(); err != nil {
			return err
		}
		app.dbConnection = nil
	}

	defer func() {
		if restoreErr == nil {
			return
		}
		recoverErr := app.recoverAppDataFromRollback(rollbackDir, appDataDir, hasCurrentData)
		if recoverErr != nil {
			restoreErr = fmt.Errorf("%w (rollback failed: %v)", restoreErr, recoverErr)
		}
	}()

	if err := clearDirectory(appDataDir); err != nil {
		return err
	}
	if err := copyDirectoryTree(extractedRoot, appDataDir); err != nil {
		return err
	}

	if err := app.reopenDatabaseAndServices(); err != nil {
		return err
	}
	if err := app.resumeRuntimeServicesAfterRestore(); err != nil {
		return err
	}
	return nil
}

func directoryHasAnyEntry(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return len(entries) > 0, nil
}

func clearDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(path, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyDirectoryTree(sourceRoot string, destRoot string) error {
	if err := os.MkdirAll(destRoot, 0o700); err != nil {
		return err
	}
	return filepath.WalkDir(sourceRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relPath, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		destPath := filepath.Join(destRoot, relPath)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0o700)
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0o700); err != nil {
			return err
		}
		dest, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		if err := copyFileContent(path, dest); err != nil {
			_ = dest.Close()
			return err
		}
		return dest.Close()
	})
}

func copyFileContent(sourcePath string, dest io.Writer) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = source.Close()
	}()
	_, err = io.Copy(dest, source)
	return err
}

func copyFilePath(sourcePath string, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o700); err != nil {
		return err
	}
	dest, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if err := copyFileContent(sourcePath, dest); err != nil {
		_ = dest.Close()
		return err
	}
	return dest.Close()
}

func (app *App) createDatabaseSnapshot(destinationPath string) error {
	_ = os.Remove(destinationPath)
	if app.dbConnection == nil {
		return copyFilePath(app.Config.DatabasePath, destinationPath)
	}
	escaped := strings.ReplaceAll(destinationPath, "'", "''")
	statement := fmt.Sprintf("VACUUM INTO '%s'", escaped)
	if _, err := app.dbConnection.Exec(statement); err == nil {
		return nil
	}
	return copyFilePath(app.Config.DatabasePath, destinationPath)
}

func (app *App) recoverAppDataFromRollback(rollbackDir string, appDataDir string, hasRollback bool) error {
	if err := clearDirectory(appDataDir); err != nil {
		return err
	}
	if hasRollback {
		if err := copyDirectoryTree(rollbackDir, appDataDir); err != nil {
			return err
		}
	}
	if err := app.reopenDatabaseAndServices(); err != nil {
		return err
	}
	return app.resumeRuntimeServicesAfterRestore()
}

func (app *App) stopRuntimeServicesForRestore() {
	if app.ProcessMonitor != nil {
		app.ProcessMonitor.StopMonitoring()
		app.isMonitoring = false
	}
	app.stopHotkey()
	if app.ScreenshotService != nil {
		_ = app.ScreenshotService.Close()
	}
}

func (app *App) reopenDatabaseAndServices() error {
	connection, err := db.Open(app.Config.DatabasePath)
	if err != nil {
		return err
	}
	if err := db.ApplyMigrations(connection); err != nil {
		_ = connection.Close()
		return err
	}

	repository := db.NewRepository(connection)
	credentialStore := newCredentialStore(app.Config)
	cloudService := services.NewCloudService(app.Config, credentialStore, app.Logger)
	cloudSync := services.NewCloudSyncService(app.Config, credentialStore, repository, app.Logger)

	var processMonitor *services.ProcessMonitorService
	if runtime.GOOS == "windows" {
		processMonitor = services.NewProcessMonitorService(repository, app.Logger, cloudSync)
	}

	app.dbConnection = connection
	app.Database = repository
	app.GameService = services.NewGameService(repository, app.Logger)
	app.SessionService = services.NewSessionService(repository, app.Logger)
	app.ChapterService = services.NewChapterService(repository, app.Logger)
	app.MemoService = services.NewMemoService(repository, app.MemoFiles, app.Logger)
	app.UploadService = services.NewUploadService(repository, app.Logger)
	app.CredentialService = services.NewCredentialService(credentialStore, app.Logger)
	app.CloudService = cloudService
	app.CloudSyncService = cloudSync
	app.ProcessMonitor = processMonitor
	app.ScreenshotService = services.NewScreenshotService(app.Config, repository, app.Logger)
	app.ErogameScapeService = services.NewErogameScapeService(app.Config, app.Logger)
	return nil
}

func (app *App) resumeRuntimeServicesAfterRestore() error {
	if app.ProcessMonitor != nil {
		app.ProcessMonitor.StartMonitoring()
		if !app.autoTracking {
			app.ProcessMonitor.UpdateAutoTracking(false)
		}
		app.isMonitoring = app.ProcessMonitor.IsMonitoring()
	}
	if err := app.startHotkey(); err != nil {
		return err
	}
	return nil
}
