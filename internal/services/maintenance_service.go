package services

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/models"
)

const BackupTypeV1 = "appdata-zip-v1"

type GameExportStatistic struct {
	GameID                 string     `json:"gameId"`
	Title                  string     `json:"title"`
	SessionCount           int        `json:"sessionCount"`
	TotalSessionDuration   int64      `json:"totalSessionDuration"`
	AverageSessionDuration float64    `json:"averageSessionDuration"`
	LastSessionAt          *time.Time `json:"lastSessionAt,omitempty"`
}

type GameExportPayload struct {
	ExportedAt  time.Time             `json:"exportedAt"`
	Games       []models.Game         `json:"games"`
	Routes      []models.PlayRoute    `json:"routes"`
	Statistics  []GameExportStatistic `json:"statistics"`
	SessionRows []models.PlaySession  `json:"sessions"`
}

type GameExportResult struct {
	JSONPath string `json:"jsonPath"`
	CSVPath  string `json:"csvPath"`
}

type BackupManifest struct {
	CreatedAt             time.Time `json:"createdAt"`
	AppDataDir            string    `json:"appDataDir"`
	DatabaseRelativePath  string    `json:"databaseRelativePath"`
	CredentialNotice      string    `json:"credentialNotice"`
	CloudLaunchBackupType string    `json:"cloudLaunchBackupType"`
	BackupVersion         int       `json:"backupVersion"`
}

type MaintenanceRepository interface {
	ListGames(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error)
	ListPlaySessionsByGame(ctx context.Context, gameID string) ([]models.PlaySession, error)
	ListPlayRoutesByGame(ctx context.Context, gameID string) ([]models.PlayRoute, error)
}

type MaintenanceRuntimeHooks struct {
	CreateDatabaseSnapshot    func(destinationPath string) error
	StopRuntimeServices       func()
	CloseDatabaseConnection   func() error
	ReopenDatabaseAndServices func() error
	ResumeRuntimeServices     func() error
}

type MaintenanceService struct {
	config     config.Config
	repository MaintenanceRepository
	logger     *slog.Logger
	hooks      MaintenanceRuntimeHooks
}

func NewMaintenanceService(
	cfg config.Config,
	repository MaintenanceRepository,
	logger *slog.Logger,
	hooks MaintenanceRuntimeHooks,
) *MaintenanceService {
	return &MaintenanceService{
		config:     cfg,
		repository: repository,
		logger:     logger,
		hooks:      hooks,
	}
}

func (service *MaintenanceService) ExportGameData(ctx context.Context, outputDir string) (GameExportResult, error) {
	trimmed := strings.TrimSpace(outputDir)
	if trimmed == "" {
		return GameExportResult{}, newServiceError("出力先フォルダが不正です", "outputDir is empty")
	}
	if err := os.MkdirAll(trimmed, 0o700); err != nil {
		service.logger.Error("出力先フォルダの作成に失敗しました", "error", err, "operation", "ExportGameData.mkdir", "outputDir", trimmed)
		return GameExportResult{}, newServiceError("出力先フォルダの作成に失敗しました", err.Error())
	}

	games, err := service.repository.ListGames(ctx, "", models.PlayStatus(""), "title", "asc")
	if err != nil {
		service.logger.Error("ゲーム一覧の取得に失敗しました", "error", err, "operation", "ExportGameData.listGames")
		return GameExportResult{}, newServiceError("ゲーム一覧の取得に失敗しました", err.Error())
	}

	stats := make([]GameExportStatistic, 0, len(games))
	routes := make([]models.PlayRoute, 0, len(games))
	sessionRows := make([]models.PlaySession, 0, len(games)*2)
	for _, game := range games {
		gameRoutes, err := service.repository.ListPlayRoutesByGame(ctx, game.ID)
		if err != nil {
			service.logger.Error("プレイルート取得に失敗しました", "error", err, "operation", "ExportGameData.listPlayRoutes", "gameId", game.ID)
			return GameExportResult{}, newServiceError("プレイルート取得に失敗しました", err.Error())
		}
		routes = append(routes, gameRoutes...)

		sessions, err := service.repository.ListPlaySessionsByGame(ctx, game.ID)
		if err != nil {
			service.logger.Error("セッション取得に失敗しました", "error", err, "operation", "ExportGameData.listSessions", "gameId", game.ID)
			return GameExportResult{}, newServiceError("セッション取得に失敗しました", err.Error())
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
		Routes:      routes,
		Statistics:  stats,
		SessionRows: sessionRows,
	}
	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		service.logger.Error("JSONの生成に失敗しました", "error", err, "operation", "ExportGameData.marshal")
		return GameExportResult{}, newServiceError("JSONの生成に失敗しました", err.Error())
	}
	if err := os.WriteFile(jsonPath, jsonData, 0o600); err != nil {
		service.logger.Error("JSONファイルの保存に失敗しました", "error", err, "operation", "ExportGameData.writeJSON", "path", jsonPath)
		return GameExportResult{}, newServiceError("JSONファイルの保存に失敗しました", err.Error())
	}
	if err := writeExportCSV(csvPath, games, stats); err != nil {
		service.logger.Error("CSVファイルの保存に失敗しました", "error", err, "operation", "ExportGameData.writeCSV", "path", csvPath)
		return GameExportResult{}, newServiceError("CSVファイルの保存に失敗しました", err.Error())
	}

	return GameExportResult{JSONPath: jsonPath, CSVPath: csvPath}, nil
}

func (service *MaintenanceService) CreateFullBackup(outputDir string) (string, error) {
	trimmed := strings.TrimSpace(outputDir)
	if trimmed == "" {
		return "", newServiceError("出力先フォルダが不正です", "outputDir is empty")
	}
	if err := os.MkdirAll(trimmed, 0o700); err != nil {
		service.logger.Error("出力先フォルダの作成に失敗しました", "error", err, "operation", "CreateFullBackup.mkdir", "outputDir", trimmed)
		return "", newServiceError("出力先フォルダの作成に失敗しました", err.Error())
	}

	appDataDir := strings.TrimSpace(service.config.AppDataDir)
	if appDataDir == "" {
		return "", newServiceError("バックアップ元ディレクトリが不正です", "AppDataDir is empty")
	}

	relDBPath, err := filepath.Rel(appDataDir, service.config.DatabasePath)
	if err != nil {
		service.logger.Error("DB相対パスの解決に失敗しました", "error", err, "operation", "CreateFullBackup.relDBPath")
		return "", newServiceError("DB相対パスの解決に失敗しました", err.Error())
	}
	if strings.HasPrefix(relDBPath, "..") {
		return "", newServiceError("バックアップ対象DBが不正です", "database path is outside AppDataDir")
	}

	stagingDir, err := os.MkdirTemp("", "cloudlaunch-backup-")
	if err != nil {
		service.logger.Error("バックアップ準備に失敗しました", "error", err, "operation", "CreateFullBackup.mktemp")
		return "", newServiceError("バックアップ準備に失敗しました", err.Error())
	}
	defer func() {
		_ = os.RemoveAll(stagingDir)
	}()

	if err := copyDirectoryTree(appDataDir, stagingDir); err != nil {
		service.logger.Error("バックアップ準備に失敗しました", "error", err, "operation", "CreateFullBackup.copyAppData")
		return "", newServiceError("バックアップ準備に失敗しました", err.Error())
	}

	snapshotPath := filepath.Join(stagingDir, relDBPath)
	if service.hooks.CreateDatabaseSnapshot == nil {
		return "", newServiceError("DBスナップショットの取得に失敗しました", "snapshot hook is nil")
	}
	if err := service.hooks.CreateDatabaseSnapshot(snapshotPath); err != nil {
		service.logger.Error("DBスナップショットの取得に失敗しました", "error", err, "operation", "CreateFullBackup.snapshot")
		return "", newServiceError("DBスナップショットの取得に失敗しました", err.Error())
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
		CloudLaunchBackupType: BackupTypeV1,
		BackupVersion:         1,
	}
	if err := writeBackupZip(stagingDir, backupPath, manifest); err != nil {
		service.logger.Error("バックアップ作成に失敗しました", "error", err, "operation", "CreateFullBackup.writeZip", "path", backupPath)
		return "", newServiceError("バックアップ作成に失敗しました", err.Error())
	}
	return backupPath, nil
}

func (service *MaintenanceService) RestoreFullBackup(backupPath string) error {
	trimmed := strings.TrimSpace(backupPath)
	if trimmed == "" {
		return newServiceError("バックアップファイルが不正です", "backupPath is empty")
	}
	if _, err := os.Stat(trimmed); err != nil {
		if os.IsNotExist(err) {
			return newServiceError("バックアップファイルが見つかりません", err.Error())
		}
		service.logger.Error("バックアップファイルの確認に失敗しました", "error", err, "operation", "RestoreFullBackup.stat", "path", trimmed)
		return newServiceError("バックアップファイルの確認に失敗しました", err.Error())
	}

	tmpDir, err := os.MkdirTemp("", "cloudlaunch-restore-")
	if err != nil {
		service.logger.Error("復元用一時ディレクトリの作成に失敗しました", "error", err, "operation", "RestoreFullBackup.mktemp")
		return newServiceError("復元用一時ディレクトリの作成に失敗しました", err.Error())
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	if err := UnzipToDirectory(trimmed, tmpDir); err != nil {
		service.logger.Error("バックアップ展開に失敗しました", "error", err, "operation", "RestoreFullBackup.unzip", "path", trimmed)
		return newServiceError("バックアップ展開に失敗しました", err.Error())
	}

	if err := service.restoreAppDataFrom(tmpDir); err != nil {
		service.logger.Error("バックアップ復元に失敗しました", "error", err, "operation", "RestoreFullBackup.restore")
		return newServiceError("バックアップ復元に失敗しました", err.Error())
	}

	return nil
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
			formatTimePtr(statOrGameLastPlayed(game.LastPlayed)),
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

func statOrGameLastPlayed(value *time.Time) *time.Time {
	return value
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

func UnzipToDirectory(zipPath string, destRoot string) error {
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

func ReadBackupManifest(extractedRoot string) (*BackupManifest, error) {
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
	if manifest.CloudLaunchBackupType != BackupTypeV1 {
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

func (service *MaintenanceService) restoreAppDataFrom(extractedRoot string) (restoreErr error) {
	appDataDir := strings.TrimSpace(service.config.AppDataDir)
	if appDataDir == "" {
		return errors.New("appDataDir is empty")
	}

	manifest, err := ReadBackupManifest(extractedRoot)
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

	if service.hooks.StopRuntimeServices != nil {
		service.hooks.StopRuntimeServices()
	}
	if service.hooks.CloseDatabaseConnection != nil {
		if err := service.hooks.CloseDatabaseConnection(); err != nil {
			return err
		}
	}

	defer func() {
		if restoreErr == nil {
			return
		}
		recoverErr := service.recoverAppDataFromRollback(rollbackDir, appDataDir, hasCurrentData)
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

	if service.hooks.ReopenDatabaseAndServices == nil {
		return errors.New("reopen hook is nil")
	}
	if err := service.hooks.ReopenDatabaseAndServices(); err != nil {
		return err
	}
	if service.hooks.ResumeRuntimeServices == nil {
		return nil
	}
	return service.hooks.ResumeRuntimeServices()
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

func CopyFilePath(sourcePath string, destPath string) error {
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

func (service *MaintenanceService) recoverAppDataFromRollback(rollbackDir string, appDataDir string, hasRollback bool) error {
	if err := clearDirectory(appDataDir); err != nil {
		return err
	}
	if hasRollback {
		if err := copyDirectoryTree(rollbackDir, appDataDir); err != nil {
			return err
		}
	}
	if service.hooks.ReopenDatabaseAndServices == nil {
		return errors.New("reopen hook is nil")
	}
	if err := service.hooks.ReopenDatabaseAndServices(); err != nil {
		return err
	}
	if service.hooks.ResumeRuntimeServices == nil {
		return nil
	}
	return service.hooks.ResumeRuntimeServices()
}
