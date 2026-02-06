//go:build windows

package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// CaptureForegroundWindow はホットキー経由で前面ウィンドウを撮影する。
func (service *ScreenshotService) CaptureForegroundWindow(ctx context.Context, target CaptureTarget) (string, string, error) {
	if target.HWND == 0 {
		return "", "", errors.New("hwnd is empty")
	}
	hwnd := windows.Handle(target.HWND)
	pid := int(windowProcessID(hwnd))
	if pid <= 0 {
		return "", "", errors.New("failed to resolve pid")
	}
	game, proc, err := service.findGameByPID(ctx, pid)
	if err != nil {
		return "", "", err
	}
	if game == nil {
		return "", "", errors.New("game not found")
	}

	baseDir := strings.TrimSpace(service.appDataDir)
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	saveDir := filepath.Join(baseDir, "screenshots", game.ID)
	fullPath, tmpPath, err := service.buildScreenshotPaths(game.ID, saveDir)
	if err != nil {
		return "", "", err
	}

	crop, err := getClientRectOnScreen(hwnd)
	if err != nil {
		return "", "", err
	}

	service.logCapture(
		slog.LevelInfo,
		"ホットキーキャプチャ開始",
		"gameId", game.ID,
		"title", game.Title,
		"exePath", game.ExePath,
		"pid", pid,
		"hwnd", target.HWND,
		"fallback", target.FromFallback,
		"procPath", proc.Cmd,
		"crop", fmt.Sprintf("%d,%d %dx%d", crop.Left, crop.Top, crop.Right-crop.Left, crop.Bottom-crop.Top),
		"output", fullPath,
	)

	outputPath := fullPath
	if service.localJpeg {
		outputPath = tmpPath
	}

	if err := captureWindowWithDXGI(hwnd, crop, outputPath); err != nil {
		service.logCapture(slog.LevelWarn, "DXGIキャプチャに失敗", "error", err)
		if tmpPath != "" {
			_ = os.Remove(tmpPath)
		}
		return "", "", err
	}

	if tmpPath != "" {
		if convertErr := service.convertFileToJpeg(tmpPath, fullPath); convertErr != nil {
			_ = os.Remove(tmpPath)
			service.logCapture(slog.LevelWarn, "DXGI JPEG変換に失敗", "error", convertErr)
			return "", "", convertErr
		}
		_ = os.Remove(tmpPath)
	}

	return game.ID, fullPath, nil
}

func captureWindowWithDXGI(hwnd windows.Handle, crop windowRect, outputPath string) error {
	return errors.New("dxgi helper is not configured")
}
