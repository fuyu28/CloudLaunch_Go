// アプリケーション全体の初期化とサービス公開を行う。
package app

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/infrastructure/credentials"
	"CloudLaunch_Go/internal/infrastructure/db"
	"CloudLaunch_Go/internal/logging"
	"CloudLaunch_Go/internal/memo"
	"CloudLaunch_Go/internal/services"
)

// playSessionLookup はセッション mutation 前に gameID を確保するための最小ポート。
// 具象の db.Repository に依存せず、GetPlaySessionByID だけを要求する。
type playSessionLookup interface {
	GetPlaySessionByID(ctx context.Context, sessionID string) (*domain.PlaySession, error)
}

// routeLookup はルート mutation（特に Delete）前に gameID を確保するための最小ポート。
type routeLookup interface {
	GetRouteByID(ctx context.Context, routeID string) (*domain.Route, error)
}

// App はWailsと連携するアプリケーション本体を表す。
type App struct {
	ctx                 context.Context
	Config              config.Config
	Logger              *slog.Logger
	logLevel            *slog.LevelVar
	GameService         *services.GameService
	SessionService      *services.SessionService
	RouteService        *services.RouteService
	MemoService         *services.MemoService
	MemoFiles           *memo.FileManager
	CredentialService   *services.CredentialService
	ContentSyncService  *services.ContentSyncService
	ErogameScapeService *services.ErogameScapeService
	ProcessMonitor      *services.ProcessMonitorService
	ScreenshotService   *services.ScreenshotService
	MemoCloudService    *services.MemoCloudService
	MaintenanceService  *services.MaintenanceService
	HotkeyService       services.HotkeyService
	hotkeyMu            sync.Mutex
	dbConnection        *sql.DB
	autoTracking        bool
	isMonitoring        bool
	syncCoalescer       *asyncCoalescer
	playSessionLookup   playSessionLookup
	routeLookup         routeLookup
}

// NewApp はアプリケーションを初期化する。
func NewApp(ctx context.Context) (*App, error) {
	cfg := config.LoadFromEnv()
	logger, logLevel := logging.NewLogger(cfg.AppDataDir, cfg.LogLevel)

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
		logLevel:     logLevel,
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
	if app.GameService != nil {
		if err := app.GameService.RetryPendingMemoCleanup(ctx); err != nil {
			app.Logger.Warn("保留中のローカルメモ削除に失敗しました", "error", err)
		}
	}
	// 同期 API / 自動 Push が動く前に、未完了の Push baseline を回復する。
	// オフライン・ネットワーク不通は起動を止めず、pending を残して次回に委ねる。
	if app.ContentSyncService != nil {
		if err := app.ContentSyncService.RecoverPendingPushes(ctx); err != nil {
			app.Logger.Warn("保留中の Push baseline 回復に失敗しました", "error", err)
		}
	}
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
	app.GameService = services.NewGameService(repository, app.Logger, app.MemoFiles)
	app.SessionService = services.NewSessionService(repository, app.Logger)
	app.playSessionLookup = repository
	app.RouteService = services.NewRouteService(repository, app.Logger)
	app.routeLookup = repository
	app.MemoService = services.NewMemoService(repository, app.MemoFiles, app.Logger)
	app.CredentialService = services.NewCredentialService(credentialStore, app.Logger)
	app.ContentSyncService = services.NewContentSyncService(app.Config, credentialStore, repository, app.Logger)
	app.syncCoalescer = newAsyncCoalescer(func(id string) {
		if err := app.ContentSyncService.Push(app.context(), id, nil); err != nil {
			app.Logger.Warn("クラウド同期に失敗", "gameId", id, "detail", err)
		}
	})
	app.syncCoalescer.onPanic = func(id string, recovered any) {
		app.Logger.Error("クラウド同期中に panic を回収", "gameId", id, "recovered", recovered)
	}
	app.ErogameScapeService = services.NewErogameScapeService(app.Config, app.Logger)
	app.ProcessMonitor = services.NewProcessMonitorService(repository, app.Logger, app.ContentSyncService)
	app.ScreenshotService = services.NewScreenshotService(app.Config, repository, app.ProcessMonitor, app.Logger)
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
