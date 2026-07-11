// ゲームウィンドウのスクリーンショットを保存するサービス。
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/domain"
)

// hotkeyDefaultDirID は対象ゲームが特定できない場合のホットキー保存先ディレクトリID。
const hotkeyDefaultDirID = "default"

// ScreenshotService はゲームウィンドウのスクリーンショット取得を提供する。
type ScreenshotService struct {
	repository ScreenshotRepository
	resolver   ProcessIDResolver
	logger     *slog.Logger
	appDataDir string
	// screencap-cli 運用では自動適用できないため、設定互換性のために保持する（未使用）。
	clientOnly  bool
	localJpeg   bool
	jpegQuality int
	fileLogger  *slog.Logger
	logFile     *os.File
	// captureFunc はプラットフォーム依存のキャプチャ実装。テストで差し替え可能。
	// pid が 0 のときはフォアグラウンドウィンドウを対象にする。
	captureFunc func(ctx context.Context, pid int, outPath string) error
}

// NewScreenshotService は ScreenshotService を生成する。
// resolver は nil を許容する（nil の場合は PID 解決不可として扱う）。
func NewScreenshotService(
	cfg config.Config,
	repository ScreenshotRepository,
	resolver ProcessIDResolver,
	logger *slog.Logger,
) *ScreenshotService {
	fileLogger, logFile := newScreenshotFileLogger(cfg.AppDataDir, cfg.LogLevel)
	s := &ScreenshotService{
		repository:  repository,
		resolver:    resolver,
		logger:      logger,
		appDataDir:  cfg.AppDataDir,
		clientOnly:  cfg.ScreenshotClientOnly,
		localJpeg:   cfg.ScreenshotLocalJpeg,
		jpegQuality: cfg.ScreenshotJpegQuality,
		fileLogger:  fileLogger,
		logFile:     logFile,
	}
	s.captureFunc = s.captureWithScreencap
	return s
}

// resolvePID は exePath から稼働中プロセスIDを引く。pid が 0 のときは見つからなかったことを表す。
// resolver 未設定・パス空・プロセス不在は (0, nil)、プロセス一覧の取得失敗のみエラーを返す。
func (service *ScreenshotService) resolvePID(exePath string) (int, error) {
	trimmed := strings.TrimSpace(exePath)
	if service.resolver == nil || trimmed == "" {
		return 0, nil
	}
	pids, err := service.resolver.FindProcessIDsByExe(trimmed)
	if err != nil {
		return 0, newServiceError("プロセス一覧の取得に失敗しました", err.Error())
	}
	if len(pids) == 0 {
		return 0, nil
	}
	return pids[0], nil
}

// SetClientOnly はキャプチャ対象をクライアント領域のみにするか更新する。
func (service *ScreenshotService) SetClientOnly(enabled bool) {
	service.clientOnly = enabled
}

// SetLocalJpeg はローカル保存をJPEG形式にするか更新する。
func (service *ScreenshotService) SetLocalJpeg(enabled bool) {
	service.localJpeg = enabled
}

// SetJpegQuality はスクリーンショットJPEG品質を更新する。
func (service *ScreenshotService) SetJpegQuality(value int) {
	service.jpegQuality = value
}

// CaptureGameScreenshot は指定ゲームのスクリーンショットを保存し、保存先パスを返す。
func (service *ScreenshotService) CaptureGameScreenshot(ctx context.Context, gameID string) (string, error) {
	trimmed := strings.TrimSpace(gameID)
	if trimmed == "" {
		return "", errors.New("gameID is empty")
	}
	game, err := service.repository.GetGameByID(ctx, trimmed)
	if err != nil {
		return "", err
	}
	if game == nil {
		return "", errors.New("game not found")
	}

	// 明示キャプチャでは起動中プロセスの PID が必須。ディレクトリ作成より前に解決し、
	// 見つからなければ空ディレクトリを作らずにエラーで返す。
	pid, err := service.resolvePID(game.ExePath)
	if err != nil {
		return "", err
	}
	if pid == 0 {
		return "", newServiceError("ゲームのプロセスが見つかりません", "ゲームが起動しているか確認してください")
	}

	baseDir := strings.TrimSpace(service.appDataDir)
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	saveDir := filepath.Join(baseDir, "screenshots", game.ID)

	fullPath, err := service.buildScreenshotPaths(game.ID, saveDir)
	if err != nil {
		return "", err
	}
	service.logCapture(
		slog.LevelInfo,
		"スクリーンショット開始",
		"gameId", game.ID,
		"title", game.Title,
		"pid", pid,
		"output", fullPath,
		"clientOnly", service.clientOnly,
		"localJpeg", service.localJpeg,
	)

	if err := service.captureFunc(ctx, pid, fullPath); err != nil {
		service.logCapture(slog.LevelWarn, "スクリーンショット取得に失敗", "gameId", game.ID, "error", err)
		return "", err
	}
	service.logCapture(slog.LevelInfo, "スクリーンショット保存完了", "gameId", game.ID, "output", fullPath)
	return fullPath, nil
}

// CaptureHotkey はホットキー経由でキャプチャし、(保存先ゲームID, 保存パス, error) を返す。
// 対象ゲームがある場合は PID が必須（プライバシー保護のため、PID が引けないときに
// 無関係なフォアグラウンドウィンドウを撮って当該ゲームのフォルダにアップロードしない）。
// 対象ゲームが無い場合のみ pid 0（フォアグラウンド）でキャプチャし default ディレクトリに保存する（アップロードなし）。
func (service *ScreenshotService) CaptureHotkey(ctx context.Context, preferredGameID string) (string, string, error) {
	game, err := service.resolveHotkeyGame(ctx, preferredGameID)
	if err != nil {
		return "", "", err
	}

	gameID := hotkeyDefaultDirID
	gameTitle := "default"
	gameExePath := ""
	pid := 0
	if game != nil {
		gameID = game.ID
		gameTitle = game.Title
		gameExePath = game.ExePath

		// 対象ゲームがある場合は PID 必須。フォアグラウンドへのフォールバックはしない。
		resolvedPID, err := service.resolvePID(gameExePath)
		if err != nil {
			return "", "", err
		}
		if resolvedPID == 0 {
			return "", "", newServiceError("ゲームのプロセスが見つかりません", "ゲームが起動しているか確認してください")
		}
		pid = resolvedPID
	}

	baseDir := strings.TrimSpace(service.appDataDir)
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	saveDir := filepath.Join(baseDir, "screenshots", gameID)
	fullPath, err := service.buildScreenshotPaths(gameID, saveDir)
	if err != nil {
		return "", "", err
	}

	service.logCapture(
		slog.LevelInfo,
		"ホットキーキャプチャ開始",
		"gameId", gameID,
		"title", gameTitle,
		"exePath", gameExePath,
		"pid", pid,
		"output", fullPath,
	)

	if err := service.captureFunc(ctx, pid, fullPath); err != nil {
		service.logCapture(slog.LevelWarn, "スクリーンショット取得に失敗", "error", err)
		return "", "", err
	}

	if game == nil {
		return "", fullPath, nil
	}
	return game.ID, fullPath, nil
}

func (service *ScreenshotService) resolveHotkeyGame(
	ctx context.Context,
	preferredGameID string,
) (*domain.Game, error) {
	trimmed := strings.TrimSpace(preferredGameID)
	if trimmed == "" {
		return nil, nil
	}
	game, err := service.repository.GetGameByID(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	return game, nil
}

func (service *ScreenshotService) buildScreenshotPaths(gameID string, saveDir string) (string, error) {
	if strings.TrimSpace(gameID) == "" {
		return "", errors.New("gameID is empty")
	}
	if strings.TrimSpace(saveDir) == "" {
		return "", errors.New("saveDir is empty")
	}
	if err := os.MkdirAll(saveDir, 0o700); err != nil {
		return "", err
	}
	// 同一秒内の連続キャプチャで --overwrite により上書き消失しないよう、ミリ秒まで含める。
	now := time.Now()
	timestamp := fmt.Sprintf("%s_%03d", now.Format("20060102_150405"), now.Nanosecond()/int(time.Millisecond))
	ext := ".png"
	if service.localJpeg {
		ext = ".jpg"
	}
	fullPath := filepath.Join(saveDir, fmt.Sprintf("%s_%s%s", timestamp, gameID, ext))
	return fullPath, nil
}

func (service *ScreenshotService) Close() error {
	if service == nil || service.logFile == nil {
		return nil
	}
	err := service.logFile.Close()
	service.logFile = nil
	return err
}

func newScreenshotFileLogger(appDataDir string, level string) (*slog.Logger, *os.File) {
	baseDir := strings.TrimSpace(appDataDir)
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	logDir := filepath.Join(baseDir, "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return nil, nil
	}
	logPath := filepath.Join(logDir, "screenshot.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil
	}

	logLevel := parseLogLevel(level)
	return slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{Level: logLevel})), file
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (service *ScreenshotService) logCapture(level slog.Level, msg string, attrs ...any) {
	if service.fileLogger != nil {
		service.fileLogger.Log(context.Background(), level, msg, attrs...)
	}
	if service.logger != nil {
		service.logger.Log(context.Background(), level, msg, attrs...)
	}
}
