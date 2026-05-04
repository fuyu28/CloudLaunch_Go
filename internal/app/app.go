// @fileoverview アプリケーション全体の初期化とサービス公開を行う。
package app

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/credentials"
	"CloudLaunch_Go/internal/db"
	"CloudLaunch_Go/internal/logging"
	"CloudLaunch_Go/internal/memo"
	"CloudLaunch_Go/internal/services"
)

// App はWailsと連携するアプリケーション本体を表す。
type App struct {
	ctx                 context.Context
	Config              config.Config
	Logger              *slog.Logger
	GameService         *services.GameService
	SessionService      *services.SessionService
	MemoService         *services.MemoService
	MemoFiles           *memo.FileManager
	CredentialService   *services.CredentialService
	CloudService        *services.CloudService
	CloudSyncService    *services.CloudSyncService
	ErogameScapeService *services.ErogameScapeService
	ProcessMonitor      *services.ProcessMonitorService
	ScreenshotService   *services.ScreenshotService
	MemoCloudService    *services.MemoCloudService
	MaintenanceService  *services.MaintenanceService
	HotkeyService       services.HotkeyService
	dbConnection        *sql.DB
	autoTracking        bool
	isMonitoring        bool
}

// NewApp はアプリケーションを初期化する。
func NewApp(ctx context.Context) (*App, error) {
	cfg := config.LoadFromEnv()
	logger := logging.NewLogger(cfg.AppDataDir, cfg.LogLevel)

	if error := os.MkdirAll(cfg.AppDataDir, 0o700); error != nil {
		return nil, error
	}
	if error := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o700); error != nil {
		return nil, error
	}

	connection, error := db.Open(cfg.DatabasePath)
	if error != nil {
		return nil, error
	}
	if error := db.ApplyMigrations(connection); error != nil {
		_ = connection.Close()
		return nil, error
	}

	repository := db.NewRepository(connection)
	credentialStore := newCredentialStore(cfg)
	memoFiles := memo.NewFileManager(cfg.AppDataDir)
	if error := memoFiles.EnsureBaseDir(); error != nil {
		_ = connection.Close()
		return nil, error
	}

	app := &App{
		Config:       cfg,
		Logger:       logger,
		MemoFiles:    memoFiles,
		dbConnection: connection,
		autoTracking: true,
		isMonitoring: false,
	}
	app.configureServices(repository, credentialStore)

	logger.Info("CloudLaunch backend initialized")
	return app, nil
}

// Startup はWailsの起動時に呼ばれる。
func (app *App) Startup(ctx context.Context) {
	app.ctx = ctx
	if app.ProcessMonitor != nil {
		app.ProcessMonitor.StartMonitoring()
		app.isMonitoring = app.ProcessMonitor.IsMonitoring()
	}
	if err := app.startHotkey(); err != nil {
		app.Logger.Warn("ホットキーの開始に失敗しました", "error", err)
	}
}

func (app *App) context() context.Context {
	if app.ctx != nil {
		return app.ctx
	}
	return context.Background()
}

// Shutdown はアプリケーションの終了処理を行う。
func (app *App) Shutdown(ctx context.Context) error {
	app.Logger.Info("CloudLaunch backend shutting down")
	if app.ProcessMonitor != nil {
		app.ProcessMonitor.StopMonitoring()
	}
	app.stopHotkey()
	if app.ScreenshotService != nil {
		if err := app.ScreenshotService.Close(); err != nil {
			app.Logger.Warn("スクリーンショットログのクローズに失敗しました", "error", err)
		}
	}
	if app.dbConnection != nil {
		return app.dbConnection.Close()
	}
	return nil
}

func (app *App) configureServices(repository *db.Repository, credentialStore credentials.Store) {
	app.GameService = services.NewGameService(repository, app.Logger)
	app.SessionService = services.NewSessionService(repository, app.Logger)
	app.MemoService = services.NewMemoService(repository, app.MemoFiles, app.Logger)
	app.CredentialService = services.NewCredentialService(credentialStore, app.Logger)
	app.CloudService = services.NewCloudService(app.Config, credentialStore, app.Logger)
	app.CloudSyncService = services.NewCloudSyncService(app.Config, credentialStore, repository, app.Logger)
	app.ErogameScapeService = services.NewErogameScapeService(app.Config, app.Logger)
	app.ProcessMonitor = services.NewProcessMonitorService(repository, app.Logger, app.CloudSyncService)
	app.ScreenshotService = services.NewScreenshotService(app.Config, repository, app.Logger)
	app.MemoCloudService = services.NewMemoCloudService(app.Config, credentialStore, app.GameService, app.MemoService, app.Logger)
	app.MaintenanceService = services.NewMaintenanceService(
		app.Config,
		repository,
		app.Logger,
		services.MaintenanceRuntimeHooks{
			CreateDatabaseSnapshot:    app.createDatabaseSnapshot,
			StopRuntimeServices:       app.stopRuntimeServicesForRestore,
			CloseDatabaseConnection:   app.closeDatabaseConnection,
			ReopenDatabaseAndServices: app.reopenDatabaseAndServices,
			ResumeRuntimeServices:     app.resumeRuntimeServicesAfterRestore,
		},
	)
}
