// @fileoverview ゲームウィンドウのスクリーンショットを保存するサービス。
package services

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
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
	repository     *db.Repository
	processMonitor *ProcessMonitorService
	logger         *slog.Logger
	appDataDir     string
	clientOnly     bool
	localJpeg      bool
	jpegQuality    int
	fileLogger     *slog.Logger
}

// NewScreenshotService は ScreenshotService を生成する。
func NewScreenshotService(
	cfg config.Config,
	repository *db.Repository,
	processMonitor *ProcessMonitorService,
	logger *slog.Logger,
) *ScreenshotService {
	return &ScreenshotService{
		repository:     repository,
		processMonitor: processMonitor,
		logger:         logger,
		appDataDir:     cfg.AppDataDir,
		clientOnly:     cfg.ScreenshotClientOnly,
		localJpeg:      cfg.ScreenshotLocalJpeg,
		jpegQuality:    cfg.ScreenshotJpegQuality,
		fileLogger:     newScreenshotFileLogger(cfg.AppDataDir, cfg.LogLevel),
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

// CaptureGameWindow は指定ゲームのウィンドウを撮影して保存し、保存先パスを返す。
func (service *ScreenshotService) CaptureGameWindow(ctx context.Context, gameID string) (string, error) {
	if service.processMonitor == nil {
		return "", errors.New("process monitor is nil")
	}
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
	exePath := strings.TrimSpace(game.ExePath)
	if exePath == "" || exePath == UnconfiguredExePath {
		return "", errors.New("exePath is invalid")
	}

	pids, err := service.processMonitor.FindProcessIDsByExe(exePath)
	if err != nil {
		service.logCapture(slog.LevelWarn, "プロセス検索に失敗", "gameId", trimmed, "error", err)
		return "", err
	}
	if len(pids) == 0 {
		service.logCapture(slog.LevelInfo, "ゲームプロセスが見つかりません", "gameId", trimmed, "exePath", exePath)
		return "", errors.New("game process not found")
	}
	pids = rankPidsForCapture(pids)

	baseDir := strings.TrimSpace(service.appDataDir)
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	saveDir := filepath.Join(baseDir, "screenshots", game.ID)
	if err := os.MkdirAll(saveDir, 0o700); err != nil {
		return "", err
	}

	ext := ".png"
	if service.localJpeg {
		ext = ".jpg"
	}
	fileName := fmt.Sprintf("%s_%s%s", time.Now().Format("20060102_150405"), game.ID, ext)
	fullPath := filepath.Join(saveDir, fileName)
	service.logCapture(
		slog.LevelInfo,
		"スクリーンショット開始",
		"gameId", game.ID,
		"title", game.Title,
		"exePath", exePath,
		"pids", pids,
		"output", fullPath,
		"clientOnly", service.clientOnly,
		"localJpeg", service.localJpeg,
	)

	for _, pid := range pids {
		outputPath := fullPath
		tmpPath := ""
		if service.localJpeg {
			outputPath = filepath.Join(saveDir, fmt.Sprintf("%s_%s.tmp.png", time.Now().Format("20060102_150405"), game.ID))
			tmpPath = outputPath
		}

		service.logCapture(slog.LevelDebug, "WGCキャプチャ開始", "pid", pid, "output", outputPath)
		ok, err := captureWindowWithWGC(pid, outputPath, service.clientOnly)
		if err == nil && ok {
			if tmpPath != "" {
				if convertErr := service.convertFileToJpeg(tmpPath, fullPath); convertErr != nil {
					_ = os.Remove(tmpPath)
					service.logCapture(slog.LevelWarn, "WGC JPEG変換に失敗", "pid", pid, "error", convertErr)
					return "", convertErr
				}
				_ = os.Remove(tmpPath)
			}
			service.logCapture(slog.LevelInfo, "WGCキャプチャ成功", "pid", pid, "output", fullPath)
			return fullPath, nil
		}
		if err != nil {
			service.logCapture(slog.LevelWarn, "WGCキャプチャに失敗", "pid", pid, "error", err)
		}
		if tmpPath != "" {
			_ = os.Remove(tmpPath)
		}
	}

	var captured image.Image
	var captureErr error
	for _, pid := range pids {
		service.logCapture(slog.LevelDebug, "フォールバックキャプチャ開始", "pid", pid)
		captured, captureErr = captureWindowImageByPID(pid, service.clientOnly)
		if captureErr == nil && captured != nil {
			service.logCapture(slog.LevelInfo, "フォールバックキャプチャ成功", "pid", pid, "output", fullPath)
			break
		}
		if captureErr != nil {
			service.logCapture(slog.LevelWarn, "フォールバックキャプチャに失敗", "pid", pid, "error", captureErr)
		}
	}
	if captureErr != nil {
		return "", captureErr
	}
	if captured == nil {
		return "", errors.New("failed to capture window")
	}

	if err := service.saveImage(fullPath, captured); err != nil {
		service.logCapture(slog.LevelWarn, "スクリーンショット保存に失敗", "output", fullPath, "error", err)
		return "", err
	}

	return fullPath, nil
}

func (service *ScreenshotService) saveImage(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			service.logger.Warn("スクリーンショットの保存に失敗", "error", closeErr)
		}
	}()

	if service.localJpeg {
		quality := normalizeJpegQuality(service.jpegQuality)
		return jpeg.Encode(file, img, &jpeg.Options{Quality: quality})
	}
	return png.Encode(file, img)
}

func (service *ScreenshotService) convertFileToJpeg(sourcePath string, destPath string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := out.Close(); closeErr != nil {
			service.logger.Warn("スクリーンショットの保存に失敗", "error", closeErr)
		}
	}()

	quality := normalizeJpegQuality(service.jpegQuality)
	return jpeg.Encode(out, img, &jpeg.Options{Quality: quality})
}

func normalizeJpegQuality(value int) int {
	if value < 1 || value > 100 {
		return 85
	}
	return value
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
