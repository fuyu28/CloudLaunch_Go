// @fileoverview ゲームウィンドウのスクリーンショットを保存するサービス。
package services

import (
	"context"
	"errors"
	"fmt"
	"image"
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
	}
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

	var captured image.Image
	var captureErr error
	for _, pid := range pids {
		captured, captureErr = captureWindowImageByPID(pid)
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

	baseDir := strings.TrimSpace(service.appDataDir)
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	saveDir := filepath.Join(baseDir, "screenshots", game.ID)
	if err := os.MkdirAll(saveDir, 0o700); err != nil {
		return "", err
	}

	fileName := fmt.Sprintf("%s_%s.png", time.Now().Format("20060102_150405"), game.ID)
	fullPath := filepath.Join(saveDir, fileName)
	file, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			service.logger.Warn("スクリーンショットの保存に失敗", "error", closeErr)
		}
	}()

	if err := png.Encode(file, captured); err != nil {
		return "", err
	}

	return fullPath, nil
}
