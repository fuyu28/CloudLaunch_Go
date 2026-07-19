// Wails バインディング用の API メソッドを提供する。
package app

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	_ "image/png"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/infrastructure/credentials"
	"CloudLaunch_Go/internal/infrastructure/storage"
	"CloudLaunch_Go/internal/logging"
	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/services"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ListGames はゲーム一覧を取得する。
func (app *App) ListGames(searchText string, filter string, sortBy string, sortDirection string) result.ApiResult[[]domain.Game] {
	ctx := app.context()
	status := normalizePlayStatus(filter)
	games, err := app.GameService.ListGames(ctx, searchText, status, sortBy, sortDirection)
	return serviceResult(games, err, "ゲーム一覧取得に失敗しました")
}

// GetGameByID はゲームを取得する。
func (app *App) GetGameByID(gameID string) result.ApiResult[*domain.Game] {
	game, err := app.GameService.GetGameByID(app.context(), gameID)
	return serviceResult(game, err, "ゲーム取得に失敗しました")
}

// CreateGame はゲームを作成する。
func (app *App) CreateGame(input services.GameInput) result.ApiResult[*domain.Game] {
	created, err := app.GameService.CreateGame(app.context(), input)
	if err != nil {
		return serviceErrorResult[*domain.Game](err, "ゲーム作成に失敗しました")
	}
	if created != nil {
		app.syncGameAsync(created.ID)
	}
	return result.OkResult(created)
}

// UpdateGame はゲームを更新する。
func (app *App) UpdateGame(gameID string, input services.GameUpdateInput) result.ApiResult[*domain.Game] {
	updated, err := app.GameService.UpdateGame(app.context(), gameID, input)
	if err != nil {
		return serviceErrorResult[*domain.Game](err, "ゲーム更新に失敗しました")
	}
	if updated != nil {
		app.syncGameAsync(updated.ID)
	}
	return result.OkResult(updated)
}

// UpdatePlayTime はセッション集計からプレイ時間キャッシュを再構築する（互換 API）。
func (app *App) UpdatePlayTime(gameID string, totalPlayTime int64, lastPlayed time.Time) result.ApiResult[*domain.Game] {
	game, err := app.GameService.UpdatePlayTime(app.context(), gameID, totalPlayTime, lastPlayed)
	return serviceResult(game, err, "プレイ時間更新に失敗しました")
}

// DeleteGame はゲームを削除する。
func (app *App) DeleteGame(gameID string) result.ApiResult[bool] {
	return boolResult(app.GameService.DeleteGame(app.context(), gameID), "ゲーム削除に失敗しました")
}

// ListRoutesByGame はルート一覧を取得する。
func (app *App) ListRoutesByGame(gameID string) result.ApiResult[[]domain.Route] {
	routes, err := app.RouteService.ListRoutesByGame(app.context(), gameID)
	return serviceResult(routes, err, "ルート取得に失敗しました")
}

// CreateRoute はルートを作成する。
func (app *App) CreateRoute(input services.RouteInput) result.ApiResult[*domain.Route] {
	route, err := app.RouteService.CreateRoute(app.context(), input)
	return serviceResult(route, err, "ルート作成に失敗しました")
}

// UpdateRoute はルートを更新する。
func (app *App) UpdateRoute(routeID string, input services.RouteUpdateInput) result.ApiResult[*domain.Route] {
	route, err := app.RouteService.UpdateRoute(app.context(), routeID, input)
	return serviceResult(route, err, "ルート更新に失敗しました")
}

// UpdateRouteOrders はルートの並び順を更新する。
func (app *App) UpdateRouteOrders(gameID string, orders []services.RouteOrderUpdate) result.ApiResult[bool] {
	return boolResult(app.RouteService.UpdateRouteOrders(app.context(), gameID, orders), "ルート順序更新に失敗しました")
}

// GetRouteStats はルートの統計を取得する。
func (app *App) GetRouteStats(gameID string) result.ApiResult[[]domain.RouteStat] {
	stats, err := app.RouteService.GetRouteStats(app.context(), gameID)
	return serviceResult(stats, err, "ルート統計取得に失敗しました")
}

// SetCurrentRoute はゲームの現在ルートを設定する。
func (app *App) SetCurrentRoute(gameID string, routeID string) result.ApiResult[bool] {
	return boolResult(app.RouteService.SetCurrentRoute(app.context(), gameID, routeID), "現在ルート更新に失敗しました")
}

// DeleteRoute はルートを削除する。
func (app *App) DeleteRoute(routeID string) result.ApiResult[bool] {
	return boolResult(app.RouteService.DeleteRoute(app.context(), routeID), "ルート削除に失敗しました")
}

// CreateSession はセッションを作成する。
func (app *App) CreateSession(input services.SessionInput) result.ApiResult[*domain.PlaySession] {
	created, err := app.SessionService.CreateSession(app.context(), input)
	if err != nil {
		return serviceErrorResult[*domain.PlaySession](err, "セッション作成に失敗しました")
	}
	if created != nil {
		app.syncGameAsync(created.GameID)
	}
	return result.OkResult(created)
}

// ListSessionsByGame はセッション一覧を取得する。
func (app *App) ListSessionsByGame(gameID string) result.ApiResult[[]domain.PlaySession] {
	sessions, err := app.SessionService.ListSessionsByGame(app.context(), gameID)
	return serviceResult(sessions, err, "セッション取得に失敗しました")
}

// DeleteSession はセッションを削除する。
func (app *App) DeleteSession(sessionID string) result.ApiResult[bool] {
	return app.mutateSessionAndSync(sessionID, "セッション削除に失敗しました", func(ctx context.Context) error {
		return app.SessionService.DeleteSession(ctx, sessionID)
	})
}

// UpdateSessionRoute はセッションのルートを更新する。
func (app *App) UpdateSessionRoute(sessionID string, routeID *string) result.ApiResult[bool] {
	return app.mutateSessionAndSync(sessionID, "セッションルート更新に失敗しました", func(ctx context.Context) error {
		return app.SessionService.UpdateSessionRoute(ctx, sessionID, routeID)
	})
}

// UpdateSessionName はセッション名を更新する。
func (app *App) UpdateSessionName(sessionID string, sessionName string) result.ApiResult[bool] {
	return app.mutateSessionAndSync(sessionID, "セッション名更新に失敗しました", func(ctx context.Context) error {
		return app.SessionService.UpdateSessionName(ctx, sessionID, sessionName)
	})
}

// mutateSessionAndSync は mutation 前にセッションを引き、成功時のみ保持した gameID で同期する。
// セッションが無い場合（nil, nil）は mutation 自体は実行し、同期はスキップする（既存挙動）。
func (app *App) mutateSessionAndSync(sessionID string, fallbackMessage string, mutate func(ctx context.Context) error) result.ApiResult[bool] {
	ctx := app.context()
	trimmedID := strings.TrimSpace(sessionID)
	session, err := app.playSessionLookup.GetPlaySessionByID(ctx, trimmedID)
	if err != nil {
		return serviceErrorResult[bool](err, "セッション取得に失敗しました")
	}
	gameID := ""
	if session != nil {
		gameID = strings.TrimSpace(session.GameID)
	}
	if err := mutate(ctx); err != nil {
		return serviceErrorResult[bool](err, fallbackMessage)
	}
	if gameID != "" {
		app.syncGameAsync(gameID)
	}
	return result.OkResult(true)
}

// CreateMemo はメモを作成する。
func (app *App) CreateMemo(input services.MemoInput) result.ApiResult[*domain.Memo] {
	memo, err := app.MemoService.CreateMemo(app.context(), input)
	return serviceResult(memo, err, "メモ作成に失敗しました")
}

// UpdateMemo はメモを更新する。
func (app *App) UpdateMemo(memoID string, input services.MemoUpdateInput) result.ApiResult[*domain.Memo] {
	memo, err := app.MemoService.UpdateMemo(app.context(), memoID, input)
	return serviceResult(memo, err, "メモ更新に失敗しました")
}

// GetMemoByID はメモを取得する。
func (app *App) GetMemoByID(memoID string) result.ApiResult[*domain.Memo] {
	memo, err := app.MemoService.GetMemoByID(app.context(), memoID)
	return serviceResult(memo, err, "メモ取得に失敗しました")
}

// ListAllMemos は全メモを取得する。
func (app *App) ListAllMemos() result.ApiResult[[]domain.Memo] {
	memos, err := app.MemoService.ListAllMemos(app.context())
	return serviceResult(memos, err, "メモ取得に失敗しました")
}

// ListMemosByGame はメモ一覧を取得する。
func (app *App) ListMemosByGame(gameID string) result.ApiResult[[]domain.Memo] {
	memos, err := app.MemoService.ListMemosByGame(app.context(), gameID)
	return serviceResult(memos, err, "メモ取得に失敗しました")
}

// DeleteMemo はメモを削除する。
func (app *App) DeleteMemo(memoID string) result.ApiResult[bool] {
	return boolResult(app.MemoService.DeleteMemo(app.context(), memoID), "メモ削除に失敗しました")
}

// FileFilterInput はファイル選択フィルタを表す。
type FileFilterInput struct {
	Name       string   `json:"name"`
	Extensions []string `json:"extensions"`
}

// FrontendLogPayload はレンダラープロセスから送信されるログ情報。
type FrontendLogPayload struct {
	Level     string `json:"level"`
	Message   string `json:"message"`
	Component string `json:"component"`
	Function  string `json:"function"`
	Context   string `json:"context"`
	Data      any    `json:"data"`
	Timestamp string `json:"timestamp"`
}

// FrontendErrorPayload はレンダラープロセスから送信されるエラー情報。
type FrontendErrorPayload struct {
	Level     string `json:"level"`
	Message   string `json:"message"`
	Stack     string `json:"stack"`
	Context   string `json:"context"`
	Component string `json:"component"`
	Function  string `json:"function"`
	Data      any    `json:"data"`
	Timestamp string `json:"timestamp"`
}

// appendIfNonEmpty は trim 後に非空ならキー・値を attrs に追加する。
func appendIfNonEmpty(attrs []any, key, value string) []any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return attrs
	}
	return append(attrs, key, trimmed)
}

// ReportLog はフロントエンドログをバックエンドログに統合する。
func (app *App) ReportLog(payload FrontendLogPayload) {
	if app == nil || app.Logger == nil {
		return
	}
	message := strings.TrimSpace(payload.Message)
	if message == "" {
		message = "frontend log (empty message)"
	}
	attrs := []any{"origin", "renderer"}
	attrs = appendIfNonEmpty(attrs, "component", payload.Component)
	attrs = appendIfNonEmpty(attrs, "function", payload.Function)
	attrs = appendIfNonEmpty(attrs, "context", payload.Context)
	attrs = appendIfNonEmpty(attrs, "sourceTimestamp", payload.Timestamp)
	if payload.Data != nil {
		attrs = append(attrs, "data", payload.Data)
	}
	app.Logger.Log(app.context(), logging.ParseLevel(payload.Level), message, attrs...)
}

// ReportError はフロントエラーをバックエンド側へ送信する。
func (app *App) ReportError(payload FrontendErrorPayload) {
	if app == nil || app.Logger == nil {
		return
	}
	message := strings.TrimSpace(payload.Message)
	if message == "" {
		message = "frontend error (empty message)"
	}
	attrs := []any{"origin", "renderer", "kind", "error"}
	attrs = appendIfNonEmpty(attrs, "stack", payload.Stack)
	attrs = appendIfNonEmpty(attrs, "context", payload.Context)
	attrs = appendIfNonEmpty(attrs, "component", payload.Component)
	attrs = appendIfNonEmpty(attrs, "function", payload.Function)
	attrs = appendIfNonEmpty(attrs, "sourceTimestamp", payload.Timestamp)
	if payload.Data != nil {
		attrs = append(attrs, "data", payload.Data)
	}
	level := logging.ParseLevel(payload.Level)
	if level < slog.LevelError {
		level = slog.LevelError
	}
	app.Logger.Log(app.context(), level, message, attrs...)
}

// UpdateAutoTracking は自動計測設定を更新する。
func (app *App) UpdateAutoTracking(enabled bool) result.ApiResult[bool] {
	app.autoTracking = enabled
	if app.ProcessMonitor != nil {
		app.ProcessMonitor.UpdateAutoTracking(enabled)
		app.isMonitoring = app.ProcessMonitor.IsMonitoring()
	}
	return result.OkResult(true)
}

// UpdateOfflineMode はオフラインモードの ON/OFF を切り替える。
// ON の間は ContentSyncService.Push/Pull/DeleteFromCloud が ErrOffline を返し、
// process_monitor からの自動同期も静かにスキップされる。フロントエンドの atom
// 状態は永続化されているため、起動時にもこの API を再度呼んでバックエンドへ同期させる。
func (app *App) UpdateOfflineMode(enabled bool) result.ApiResult[bool] {
	if app.ContentSyncService != nil {
		app.ContentSyncService.SetOfflineMode(enabled)
	}
	return result.OkResult(true)
}

// UpdateUploadConcurrency はアップロード同時実行数を更新する。
func (app *App) UpdateUploadConcurrency(value int) result.ApiResult[bool] {
	if value <= 0 {
		app.Logger.Warn("同時実行数が不正です", "operation", "UpdateUploadConcurrency", "value", value)
		return result.ErrorResult[bool]("同時実行数が不正です", "valueが不正です")
	}
	app.Config.S3UploadConcurrency = value
	if app.ContentSyncService != nil {
		app.ContentSyncService.SetUploadConcurrency(value)
	}
	return result.OkResult(true)
}

// UpdateS3ForcePathStyle は S3 path-style アドレス指定を更新する（MinIO 等向け）。
func (app *App) UpdateS3ForcePathStyle(enabled bool) result.ApiResult[bool] {
	app.Config.S3ForcePathStyle = enabled
	if app.ContentSyncService != nil {
		app.ContentSyncService.SetS3ForcePathStyle(enabled)
	}
	if app.MemoCloudService != nil {
		app.MemoCloudService.SetS3ForcePathStyle(enabled)
	}
	return result.OkResult(true)
}

// UpdateS3UseTLS は S3 通信の TLS 有効/無効を更新する。
func (app *App) UpdateS3UseTLS(enabled bool) result.ApiResult[bool] {
	app.Config.S3UseTLS = enabled
	if app.ContentSyncService != nil {
		app.ContentSyncService.SetS3UseTLS(enabled)
	}
	if app.MemoCloudService != nil {
		app.MemoCloudService.SetS3UseTLS(enabled)
	}
	return result.OkResult(true)
}

// UpdateLogLevel はバックエンドのログレベルを実行時に変更する。
// 受け付ける値: debug / info / warn / error（大文字小文字・空白は無視）。
func (app *App) UpdateLogLevel(level string) result.ApiResult[bool] {
	normalized := strings.ToLower(strings.TrimSpace(level))
	switch normalized {
	case "debug", "info", "warn", "warning", "error":
	default:
		app.Logger.Warn("ログレベルが不正です", "operation", "UpdateLogLevel", "level", level)
		return result.ErrorResult[bool]("ログレベルが不正です", "level must be debug|info|warn|error")
	}
	if normalized == "warning" {
		normalized = "warn"
	}
	app.Config.LogLevel = normalized
	if app.logLevel != nil {
		app.logLevel.Set(logging.ParseLevel(normalized))
	}
	app.Logger.Info("ログレベルを更新しました", "level", normalized)
	return result.OkResult(true)
}

// UpdateScreenshotSyncEnabled はスクリーンショット同期の有効/無効を更新する。
func (app *App) UpdateScreenshotSyncEnabled(enabled bool) result.ApiResult[bool] {
	app.Config.ScreenshotSyncEnabled = enabled
	return result.OkResult(true)
}

// UpdateScreenshotUploadJpeg はスクリーンショットをJPEG変換してアップロードするか更新する。
func (app *App) UpdateScreenshotUploadJpeg(enabled bool) result.ApiResult[bool] {
	app.Config.ScreenshotUploadJpeg = enabled
	return result.OkResult(true)
}

// UpdateScreenshotJpegQuality はスクリーンショットJPEGの品質を更新する。
func (app *App) UpdateScreenshotJpegQuality(value int) result.ApiResult[bool] {
	if value < 1 || value > 100 {
		app.Logger.Warn("JPEG品質が不正です", "operation", "UpdateScreenshotJpegQuality", "value", value)
		return result.ErrorResult[bool]("JPEG品質が不正です", "value must be 1-100")
	}
	app.Config.ScreenshotJpegQuality = value
	if app.ScreenshotService != nil {
		app.ScreenshotService.SetJpegQuality(value)
	}
	return result.OkResult(true)
}

// UpdateScreenshotClientOnly はスクリーンショットをクライアント領域のみ取得するか更新する。
func (app *App) UpdateScreenshotClientOnly(enabled bool) result.ApiResult[bool] {
	app.Config.ScreenshotClientOnly = enabled
	if app.ScreenshotService != nil {
		app.ScreenshotService.SetClientOnly(enabled)
	}
	return result.OkResult(true)
}

// UpdateScreenshotLocalJpeg はローカル保存形式をJPEGにするか更新する。
func (app *App) UpdateScreenshotLocalJpeg(enabled bool) result.ApiResult[bool] {
	app.Config.ScreenshotLocalJpeg = enabled
	if app.ScreenshotService != nil {
		app.ScreenshotService.SetLocalJpeg(enabled)
	}
	return result.OkResult(true)
}

// applyHotkeyChange は Config を書き換えた後にホットキーを再起動し、
// 失敗時は呼び出し側の rollback を呼んで旧設定に戻す。
// rollback は新設定を旧設定へ戻すクロージャ。errMessage はユーザー向けメッセージ。
func (app *App) applyHotkeyChange(operation, errMessage string, rollback func(), attrs ...any) result.ApiResult[bool] {
	app.hotkeyMu.Lock()
	defer app.hotkeyMu.Unlock()
	app.stopHotkeyLocked()
	if err := app.startHotkeyLocked(); err != nil {
		rollback()
		_ = app.startHotkeyLocked()
		logArgs := append([]any{"operation", operation, "error", err}, attrs...)
		app.Logger.Error(errMessage, logArgs...)
		return result.ErrorResult[bool](errMessage, err.Error())
	}
	return result.OkResult(true)
}

// UpdateScreenshotHotkey はスクリーンショットのホットキーを更新する。
func (app *App) UpdateScreenshotHotkey(combo string) result.ApiResult[bool] {
	trimmed := strings.TrimSpace(combo)
	if trimmed == "" {
		app.Logger.Warn("ホットキーが不正です", "operation", "UpdateScreenshotHotkey", "reason", "empty combo")
		return result.ErrorResult[bool]("ホットキーが不正です", "combo is empty")
	}
	if err := services.ValidateHotkeyCombo(trimmed); err != nil {
		app.Logger.Warn("ホットキーが不正です", "operation", "UpdateScreenshotHotkey", "combo", trimmed, "error", err)
		return result.ErrorResult[bool]("ホットキーが不正です", err.Error())
	}
	if app.Config.ScreenshotHotkey == trimmed {
		// Startup 済みの同一コンボを boot sync が再登録しようとして
		// already registered になるのを防ぐ。
		return result.OkResult(true)
	}
	prev := app.Config.ScreenshotHotkey
	app.Config.ScreenshotHotkey = trimmed
	return app.applyHotkeyChange("UpdateScreenshotHotkey", "ホットキーの更新に失敗しました",
		func() { app.Config.ScreenshotHotkey = prev }, "combo", trimmed)
}

// UpdateScreenshotHotkeyNotify はホットキー通知の有効/無効を更新する。
// OS ホットキーの再登録は不要なので、実行中サービスのフラグだけ更新する。
func (app *App) UpdateScreenshotHotkeyNotify(enabled bool) result.ApiResult[bool] {
	if app.Config.ScreenshotHotkeyNotify == enabled {
		return result.OkResult(true)
	}
	app.Config.ScreenshotHotkeyNotify = enabled
	app.hotkeyMu.Lock()
	if app.HotkeyService != nil {
		app.HotkeyService.SetNotify(enabled)
	}
	app.hotkeyMu.Unlock()
	return result.OkResult(true)
}

// GetMonitoringStatus は監視状態を取得する。
func (app *App) GetMonitoringStatus() result.ApiResult[[]domain.MonitoringGameStatus] {
	if app.ProcessMonitor == nil {
		return result.OkResult([]domain.MonitoringGameStatus{})
	}
	status := app.ProcessMonitor.GetMonitoringStatus()
	return result.OkResult(status)
}

// GetProcessSnapshot はプロセス一覧のデバッグ情報を取得する。
func (app *App) GetProcessSnapshot() result.ApiResult[domain.ProcessSnapshot] {
	if app.ProcessMonitor == nil {
		return result.OkResult(domain.ProcessSnapshot{Source: "none", Items: []domain.ProcessSnapshotItem{}})
	}
	snapshot := app.ProcessMonitor.GetProcessSnapshot()
	return result.OkResult(snapshot)
}

// requireProcessMonitor は ProcessMonitor が未設定ならログを残し、無効化エラーを返す。
// Success=true のときは続行、false のときはその結果をそのまま return する。
func (app *App) requireProcessMonitor(operation string) result.ApiResult[bool] {
	if app.ProcessMonitor == nil {
		app.Logger.Warn("監視が無効です", "operation", operation, "reason", "process monitor is nil")
		return result.ErrorResult[bool]("監視が無効です", "process monitor is nil")
	}
	return result.OkResult(true)
}

// PauseMonitoringSession はセッションを中断する。
func (app *App) PauseMonitoringSession(gameID string) result.ApiResult[bool] {
	if errResult := app.requireProcessMonitor("PauseMonitoringSession"); !errResult.Success {
		return errResult
	}
	trimmedGameID := strings.TrimSpace(gameID)
	if ok := app.ProcessMonitor.PauseSession(trimmedGameID); !ok {
		app.Logger.Warn("中断に失敗しました", "operation", "PauseMonitoringSession", "gameId", trimmedGameID)
		return result.ErrorResult[bool]("中断に失敗しました", "session not found")
	}
	return result.OkResult(true)
}

// ResumeMonitoringSession は中断中セッションを再開する。
func (app *App) ResumeMonitoringSession(gameID string) result.ApiResult[bool] {
	if errResult := app.requireProcessMonitor("ResumeMonitoringSession"); !errResult.Success {
		return errResult
	}
	trimmedGameID := strings.TrimSpace(gameID)
	if ok := app.ProcessMonitor.ResumeSession(trimmedGameID); !ok {
		app.Logger.Warn("再開に失敗しました", "operation", "ResumeMonitoringSession", "gameId", trimmedGameID)
		return result.ErrorResult[bool]("再開に失敗しました", "session not running")
	}
	return result.OkResult(true)
}

// EndMonitoringSession はセッションを終了して保存する。
func (app *App) EndMonitoringSession(gameID string) result.ApiResult[bool] {
	if errResult := app.requireProcessMonitor("EndMonitoringSession"); !errResult.Success {
		return errResult
	}
	trimmedGameID := strings.TrimSpace(gameID)
	if ok := app.ProcessMonitor.EndSession(trimmedGameID); !ok {
		app.Logger.Warn("終了に失敗しました", "operation", "EndMonitoringSession", "gameId", trimmedGameID)
		return result.ErrorResult[bool]("終了に失敗しました", "session not found")
	}
	return result.OkResult(true)
}

// SelectFile はファイル選択ダイアログを開く。
func (app *App) SelectFile(filters []FileFilterInput) result.ApiResult[string] {
	dialogContext := app.runtimeContext()
	fileFilters := buildFileFilters(filters)
	path, error := runtime.OpenFileDialog(dialogContext, runtime.OpenDialogOptions{
		Filters: fileFilters,
	})
	if error != nil {
		app.Logger.Error("ファイル選択に失敗", "error", error)
		return result.ErrorResult[string]("ファイル選択に失敗しました", error.Error())
	}
	if path == "" {
		app.Logger.Info("ファイル選択がキャンセルされました", "operation", "SelectFile")
		return result.ErrorResult[string]("ファイルが選択されませんでした", "")
	}
	return result.OkResult(path)
}

// SelectFolder はフォルダ選択ダイアログを開く。
func (app *App) SelectFolder() result.ApiResult[string] {
	dialogContext := app.runtimeContext()
	path, error := runtime.OpenDirectoryDialog(dialogContext, runtime.OpenDialogOptions{})
	if error != nil {
		app.Logger.Error("フォルダ選択に失敗", "error", error)
		return result.ErrorResult[string]("フォルダ選択に失敗しました", error.Error())
	}
	if path == "" {
		app.Logger.Info("フォルダ選択がキャンセルされました", "operation", "SelectFolder")
		return result.ErrorResult[string]("フォルダが選択されませんでした", "")
	}
	return result.OkResult(path)
}

// CheckFileExists はファイル存在を確認する。
func (app *App) CheckFileExists(filePath string) result.ApiResult[bool] {
	_, error := os.Stat(filePath)
	if error != nil {
		if os.IsNotExist(error) {
			return result.OkResult(false)
		}
		app.Logger.Error("ファイル存在チェックに失敗", "error", error)
		return result.ErrorResult[bool]("ファイル存在チェックに失敗しました", error.Error())
	}
	return result.OkResult(true)
}

// CheckDirectoryExists はディレクトリ存在を確認する。
func (app *App) CheckDirectoryExists(dirPath string) result.ApiResult[bool] {
	info, error := os.Stat(dirPath)
	if error != nil {
		if os.IsNotExist(error) {
			return result.OkResult(false)
		}
		app.Logger.Error("ディレクトリ存在チェックに失敗", "error", error)
		return result.ErrorResult[bool]("ディレクトリ存在チェックに失敗しました", error.Error())
	}
	return result.OkResult(info.IsDir())
}

// OpenFolder は指定パスをOSで開く。
func (app *App) OpenFolder(path string) result.ApiResult[bool] {
	if strings.TrimSpace(path) == "" {
		app.Logger.Warn("パスが不正です", "operation", "OpenFolder", "path", path)
		return result.ErrorResult[bool]("パスが不正です", "pathが空です")
	}
	if error := openPath(path); error != nil {
		app.Logger.Error("フォルダを開くのに失敗", "error", error)
		return result.ErrorResult[bool]("フォルダを開くのに失敗しました", error.Error())
	}
	return result.OkResult(true)
}

// OpenLogsDirectory はログ保存ディレクトリを開く。
func (app *App) OpenLogsDirectory() result.ApiResult[string] {
	path := app.Config.AppDataDir
	if path == "" {
		app.Logger.Warn("ログディレクトリが不明です", "operation", "OpenLogsDirectory", "reason", "AppDataDir is empty")
		return result.ErrorResult[string]("ログディレクトリが不明です", "AppDataDirが空です")
	}
	if error := openPath(path); error != nil {
		app.Logger.Error("ログディレクトリを開くのに失敗", "error", error)
		return result.ErrorResult[string]("ログディレクトリを開くのに失敗しました", error.Error())
	}
	return result.OkResult(path)
}

// SaveCredential は認証情報を保存する。
func (app *App) SaveCredential(key string, input services.CredentialInput) result.ApiResult[bool] {
	return boolResult(app.CredentialService.SaveCredential(app.context(), key, input), "認証情報保存に失敗しました")
}

// LoadCredential は認証情報を取得する。
func (app *App) LoadCredential(key string) result.ApiResult[*services.CredentialOutput] {
	return toCredentialOutput(app.CredentialService.LoadCredential(app.context(), key))
}

// DeleteCredential は認証情報を削除する。
func (app *App) DeleteCredential(key string) result.ApiResult[bool] {
	return boolResult(app.CredentialService.DeleteCredential(app.context(), key), "認証情報削除に失敗しました")
}

// LaunchGame は指定された実行ファイルを起動する。
func (app *App) LaunchGame(exePath string) result.ApiResult[bool] {
	if strings.TrimSpace(exePath) == "" || exePath == services.UnconfiguredExePath {
		app.Logger.Warn("実行ファイルが不正です", "operation", "LaunchGame", "exePath", exePath)
		return result.ErrorResult[bool]("実行ファイルが不正です", "exePathが空です")
	}
	command := exec.Command(exePath)
	command.Dir = filepath.Dir(exePath)
	if error := command.Start(); error != nil {
		app.Logger.Error("ゲーム起動に失敗", "error", error)
		return result.ErrorResult[bool]("ゲーム起動に失敗しました", error.Error())
	}
	return result.OkResult(true)
}

// CaptureGameScreenshot は指定されたゲームのスクリーンショットを保存する。
func (app *App) CaptureGameScreenshot(gameID string) result.ApiResult[string] {
	if app.ScreenshotService == nil {
		app.Logger.Warn("スクリーンショット機能が無効です", "operation", "CaptureGameScreenshot", "reason", "screenshot service is nil")
		return result.ErrorResult[string]("スクリーンショット機能が無効です", "screenshot service is nil")
	}
	path, err := app.ScreenshotService.CaptureGameScreenshot(app.context(), strings.TrimSpace(gameID))
	if err != nil {
		app.Logger.Error("スクリーンショット取得に失敗", "error", err)
		return serviceErrorResult[string](err, "スクリーンショットの取得に失敗しました")
	}
	if app.Config.ScreenshotSyncEnabled {
		if syncErr := app.uploadScreenshot(app.context(), strings.TrimSpace(gameID), path); syncErr != nil {
			app.Logger.Error("スクリーンショット同期に失敗", "error", syncErr)
			return result.ErrorResult[string]("スクリーンショットの同期に失敗しました", syncErr.Error())
		}
	}
	return result.OkResult(path)
}

func (app *App) uploadScreenshot(ctx context.Context, gameID string, filePath string) error {
	if gameID == "" {
		return errors.New("gameID is empty")
	}
	if strings.TrimSpace(filePath) == "" {
		return errors.New("filePath is empty")
	}

	client, bucket, err := app.getDefaultS3Client(ctx)
	if err != nil {
		return err
	}

	baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	key := filepath.ToSlash(filepath.Join("screenshots", gameID, baseName))

	if app.Config.ScreenshotUploadJpeg {
		quality := app.Config.ScreenshotJpegQuality
		if quality < 1 || quality > 100 {
			quality = 85
		}
		payload, err := convertImageToJpeg(filePath, quality)
		if err != nil {
			return err
		}
		return storage.UploadBytes(ctx, client, bucket, key+".jpg", payload, "image/jpeg")
	}

	payload, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
		key += ".jpg"
	case ".png":
		contentType = "image/png"
		key += ".png"
	default:
		key += ext
	}

	return storage.UploadBytes(ctx, client, bucket, key, payload, contentType)
}

func convertImageToJpeg(filePath string, quality int) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	buffer := bytes.NewBuffer(nil)
	if err := jpeg.Encode(buffer, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (app *App) runtimeContext() context.Context {
	return app.context()
}

func buildFileFilters(filters []FileFilterInput) []runtime.FileFilter {
	if len(filters) == 0 {
		return nil
	}
	fileFilters := make([]runtime.FileFilter, 0, len(filters))
	for _, filter := range filters {
		if len(filter.Extensions) == 0 {
			continue
		}
		patterns := make([]string, 0, len(filter.Extensions))
		for _, ext := range filter.Extensions {
			trimmed := strings.TrimSpace(ext)
			if trimmed == "" {
				continue
			}
			trimmed = strings.TrimPrefix(trimmed, ".")
			patterns = append(patterns, "*."+trimmed)
		}
		if len(patterns) == 0 {
			continue
		}
		fileFilters = append(fileFilters, runtime.FileFilter{
			DisplayName: filter.Name,
			Pattern:     strings.Join(patterns, ";"),
		})
	}
	return fileFilters
}

func openPath(path string) error {
	command := exec.Command("explorer.exe", path)
	return command.Start()
}

// normalizePlayStatus はUIのフィルタ文字列をモデル値へ変換する。
func normalizePlayStatus(filter string) domain.PlayStatus {
	value := strings.ToLower(strings.TrimSpace(filter))
	switch value {
	case "unplayed":
		return domain.PlayStatusUnplayed
	case "playing":
		return domain.PlayStatusPlaying
	case "played":
		return domain.PlayStatusPlayed
	default:
		return ""
	}
}

// toCredentialOutput は内部認証情報をUI向けに変換する。
func toCredentialOutput(credential *credentials.Credential, err error) result.ApiResult[*services.CredentialOutput] {
	if err != nil {
		return serviceErrorResult[*services.CredentialOutput](err, "認証情報取得に失敗しました")
	}
	if credential == nil {
		return result.OkResult[*services.CredentialOutput](nil)
	}
	return result.OkResult(&services.CredentialOutput{
		AccessKeyID: credential.AccessKeyID,
		BucketName:  credential.BucketName,
		Region:      credential.Region,
		Endpoint:    credential.Endpoint,
	})
}
