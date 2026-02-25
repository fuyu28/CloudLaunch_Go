// @fileoverview アプリケーション全体の初期化とサービス公開を行う。
package app

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"

	"CloudLaunch_Go/internal/config"
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
	Database            *db.Repository
	GameService         *services.GameService
	SessionService      *services.SessionService
	ChapterService      *services.ChapterService
	MemoService         *services.MemoService
	MemoFiles           *memo.FileManager
	UploadService       *services.UploadService
	CredentialService   *services.CredentialService
	CloudService        *services.CloudService
	CloudSyncService    *services.CloudSyncService
	ErogameScapeService *services.ErogameScapeService
	ProcessMonitor      *services.ProcessMonitorService
	ScreenshotService   *services.ScreenshotService
	HotkeyService       services.HotkeyService
	dbConnection        *sql.DB
	autoTracking        bool
	isMonitoring        bool
}

// NewApp はアプリケーションを初期化する。
func NewApp(ctx context.Context) (*App, error) {
	cfg := config.LoadFromEnv()
	logger := logging.NewLogger(cfg.LogLevel)

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

	cloudService := services.NewCloudService(cfg, credentialStore, logger)
	cloudSync := services.NewCloudSyncService(cfg, credentialStore, repository, logger)
	erogameScapeService := services.NewErogameScapeService(cfg, logger)
	processMonitor := services.NewProcessMonitorService(repository, logger, cloudSync)
	screenshotService := services.NewScreenshotService(cfg, repository, processMonitor, logger)

	app := &App{
		Config:              cfg,
		Logger:              logger,
		Database:            repository,
		GameService:         services.NewGameService(repository, logger),
		SessionService:      services.NewSessionService(repository, logger),
		ChapterService:      services.NewChapterService(repository, logger),
		MemoService:         services.NewMemoService(repository, memoFiles, logger),
		MemoFiles:           memoFiles,
		UploadService:       services.NewUploadService(repository, logger),
		CredentialService:   services.NewCredentialService(credentialStore, logger),
		CloudService:        cloudService,
		CloudSyncService:    cloudSync,
		ErogameScapeService: erogameScapeService,
		ProcessMonitor:      processMonitor,
		ScreenshotService:   screenshotService,
		dbConnection:        connection,
		autoTracking:        true,
		isMonitoring:        false,
	}

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
	app.startHotkey()
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
	if app.dbConnection != nil {
		return app.dbConnection.Close()
	}
	return nil
}
