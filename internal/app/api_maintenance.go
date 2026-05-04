// @fileoverview データエクスポートとバックアップ復元APIを提供する。
package app

import (
	"fmt"
	"os"
	"strings"

	"CloudLaunch_Go/internal/db"
	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/services"
)

// ExportGameData はゲーム情報・統計データをCSV/JSONで出力する。
func (app *App) ExportGameData(outputDir string) result.ApiResult[services.GameExportResult] {
	exported, err := app.MaintenanceService.ExportGameData(app.context(), outputDir)
	return serviceResult(exported, err, "ゲーム一覧の出力に失敗しました")
}

// CreateFullBackup はアプリデータ一式のバックアップZIPを作成する。
func (app *App) CreateFullBackup(outputDir string) result.ApiResult[string] {
	path, err := app.MaintenanceService.CreateFullBackup(outputDir)
	return serviceResult(path, err, "バックアップ作成に失敗しました")
}

// RestoreFullBackup はバックアップZIPから全データを復元する。
func (app *App) RestoreFullBackup(backupPath string) result.ApiResult[bool] {
	if err := app.MaintenanceService.RestoreFullBackup(backupPath); err != nil {
		return serviceErrorResult[bool](err, "バックアップ復元に失敗しました")
	}
	return result.OkResult(true)
}

func (app *App) createDatabaseSnapshot(destinationPath string) error {
	_ = os.Remove(destinationPath)
	if app.dbConnection == nil {
		return services.CopyFilePath(app.Config.DatabasePath, destinationPath)
	}
	escaped := strings.ReplaceAll(destinationPath, "'", "''")
	statement := fmt.Sprintf("VACUUM INTO '%s'", escaped)
	if _, err := app.dbConnection.Exec(statement); err == nil {
		return nil
	}
	return services.CopyFilePath(app.Config.DatabasePath, destinationPath)
}

func (app *App) closeDatabaseConnection() error {
	if app.dbConnection == nil {
		return nil
	}
	if err := app.dbConnection.Close(); err != nil {
		return err
	}
	app.dbConnection = nil
	return nil
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

	app.dbConnection = connection
	repository := db.NewRepository(connection)
	credentialStore := newCredentialStore(app.Config)
	app.configureServices(repository, credentialStore)
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
