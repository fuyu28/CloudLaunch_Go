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
		return "", err
	}
	if len(pids) == 0 {
		return "", errors.New("game process not found")
	}

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

	for _, pid := range pids {
		outputPath := fullPath
		tmpPath := ""
		if service.localJpeg {
			outputPath = filepath.Join(saveDir, fmt.Sprintf("%s_%s.tmp.png", time.Now().Format("20060102_150405"), game.ID))
			tmpPath = outputPath
		}

		ok, err := captureWindowWithWGC(pid, outputPath, service.clientOnly)
		if err == nil && ok {
			if tmpPath != "" {
				if convertErr := service.convertFileToJpeg(tmpPath, fullPath); convertErr != nil {
					_ = os.Remove(tmpPath)
					return "", convertErr
				}
				_ = os.Remove(tmpPath)
			}
			return fullPath, nil
		}
		if tmpPath != "" {
			_ = os.Remove(tmpPath)
		}
	}

	var captured image.Image
	var captureErr error
	for _, pid := range pids {
		captured, captureErr = captureWindowImageByPID(pid, service.clientOnly)
		if captureErr == nil && captured != nil {
			break
		}
	}
	if captureErr != nil {
		return "", captureErr
	}
	if captured == nil {
		return "", errors.New("failed to capture window")
	}

	if err := service.saveImage(fullPath, captured); err != nil {
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
