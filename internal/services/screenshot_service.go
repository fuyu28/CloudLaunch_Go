// @fileoverview ゲームウィンドウのスクリーンショットを保存するサービス。
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
	"CloudLaunch_Go/internal/db"
)

// ScreenshotService はゲームウィンドウのスクリーンショット取得を提供する。
type ScreenshotService struct {
	repository  *db.Repository
	logger      *slog.Logger
	appDataDir  string
	clientOnly  bool
	localJpeg   bool
	jpegQuality int
	fileLogger  *slog.Logger
}

// NewScreenshotService は ScreenshotService を生成する。
func NewScreenshotService(
	cfg config.Config,
	repository *db.Repository,
	logger *slog.Logger,
) *ScreenshotService {
	return &ScreenshotService{
		repository:  repository,
		logger:      logger,
		appDataDir:  cfg.AppDataDir,
		clientOnly:  cfg.ScreenshotClientOnly,
		localJpeg:   cfg.ScreenshotLocalJpeg,
		jpegQuality: cfg.ScreenshotJpegQuality,
		fileLogger:  newScreenshotFileLogger(cfg.AppDataDir, cfg.LogLevel),
	}
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

	baseDir := strings.TrimSpace(service.appDataDir)
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	saveDir := filepath.Join(baseDir, "screenshots", game.ID)
	if err := os.MkdirAll(saveDir, 0o700); err != nil {
		return "", err
	}

	fullPath, tmpPath, err := service.buildScreenshotPaths(game.ID, saveDir)
	if err != nil {
		return "", err
	}
	service.logCapture(
		slog.LevelInfo,
		"スクリーンショット開始",
		"gameId", game.ID,
		"title", game.Title,
		"output", fullPath,
		"clientOnly", service.clientOnly,
		"localJpeg", service.localJpeg,
	)

	if err := service.captureWithScreenClip(ctx, fullPath, tmpPath); err != nil {
		if errors.Is(err, ErrNoNewScreenshot) {
			service.logCapture(slog.LevelInfo, "スクリーンショットが取得されなかったため保存をスキップ", "gameId", game.ID)
			return "", err
		}
		service.logCapture(slog.LevelWarn, "スクリーンショット取得に失敗", "gameId", game.ID, "error", err)
		return "", err
	}
	service.logCapture(slog.LevelInfo, "スクリーンショット保存完了", "gameId", game.ID, "output", fullPath)
	return fullPath, nil
}

func (service *ScreenshotService) buildScreenshotPaths(gameID string, saveDir string) (string, string, error) {
	if strings.TrimSpace(gameID) == "" {
		return "", "", errors.New("gameID is empty")
	}
	if strings.TrimSpace(saveDir) == "" {
		return "", "", errors.New("saveDir is empty")
	}
	if err := os.MkdirAll(saveDir, 0o700); err != nil {
		return "", "", err
	}
	timestamp := time.Now().Format("20060102_150405")
	ext := ".png"
	if service.localJpeg {
		ext = ".jpg"
	}
	fullPath := filepath.Join(saveDir, fmt.Sprintf("%s_%s%s", timestamp, gameID, ext))
	tmpPath := ""
	if service.localJpeg {
		tmpPath = filepath.Join(saveDir, fmt.Sprintf("%s_%s.tmp.png", timestamp, gameID))
	}
	return fullPath, tmpPath, nil
}

func newScreenshotFileLogger(appDataDir string, level string) *slog.Logger {
	baseDir := strings.TrimSpace(appDataDir)
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	logDir := filepath.Join(baseDir, "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return nil
	}
	logPath := filepath.Join(logDir, "screenshot.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil
	}

	logLevel := parseLogLevel(level)
	return slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{Level: logLevel}))
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
