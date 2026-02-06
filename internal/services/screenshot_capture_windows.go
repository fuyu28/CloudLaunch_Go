//go:build windows

package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
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

	result, err := captureWindowWithDXGI(hwnd, crop, outputPath)
	monitor := parseDXGIMonitorIndex(result.Stdout)
	if err != nil {
		service.logCapture(
			slog.LevelWarn,
			"DXGIキャプチャに失敗",
			"error", err,
			"stderr", result.Stderr,
			"monitor", monitor,
		)
		if tmpPath != "" {
			_ = os.Remove(tmpPath)
		}
		return "", "", err
	}
	if strings.TrimSpace(result.Stdout) != "" {
		service.logCapture(slog.LevelInfo, "DXGIキャプチャ情報", "stdout", result.Stdout, "monitor", monitor)
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

type dxgiCaptureResult struct {
	Stdout string
	Stderr string
}

func captureWindowWithDXGI(hwnd windows.Handle, crop windowRect, outputPath string) (dxgiCaptureResult, error) {
	ensureDpiAwareness()
	if hwnd == 0 {
		return dxgiCaptureResult{}, errors.New("hwnd is zero")
	}
	width := int(crop.Right - crop.Left)
	height := int(crop.Bottom - crop.Top)
	if width <= 0 || height <= 0 {
		return dxgiCaptureResult{}, errors.New("invalid crop size")
	}
	helperPath, err := dxgiHelperPath()
	if err != nil {
		return dxgiCaptureResult{}, err
	}
	if _, statErr := os.Stat(helperPath); statErr != nil {
		return dxgiCaptureResult{}, fmt.Errorf("dxgi helper not found: %w", statErr)
	}
	args := []string{
		"--out",
		outputPath,
		"--crop",
		strconv.Itoa(int(crop.Left)),
		strconv.Itoa(int(crop.Top)),
		strconv.Itoa(width),
		strconv.Itoa(height),
		"--format",
		"png",
	}
	command := execCommandHidden(context.Background(), helperPath, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	result := dxgiCaptureResult{
		Stdout: strings.TrimSpace(stdout.String()),
		Stderr: strings.TrimSpace(stderr.String()),
	}
	if err != nil {
		if result.Stderr != "" {
			return result, fmt.Errorf("%w: %s", err, result.Stderr)
		}
		if result.Stdout != "" {
			return result, fmt.Errorf("%w: %s", err, result.Stdout)
		}
		return result, err
	}
	return result, nil
}

func dxgiHelperPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, "dxgi_screenshot.exe"), nil
}

func captureWindowByPID(pid int, outputPath string) (CaptureMeta, error) {
	if pid <= 0 {
		return CaptureMeta{}, errors.New("pid is invalid")
	}
	hwnd, err := findBestWindowForPID(uint32(pid))
	if err != nil {
		return CaptureMeta{}, err
	}
	crop, err := getClientRectOnScreen(hwnd)
	if err != nil {
		return CaptureMeta{}, err
	}
	result, err := captureWindowWithDXGI(hwnd, crop, outputPath)
	meta := CaptureMeta{
		HWND:       uintptr(hwnd),
		CropX:      int(crop.Left),
		CropY:      int(crop.Top),
		CropW:      int(crop.Right - crop.Left),
		CropH:      int(crop.Bottom - crop.Top),
		Monitor:    parseDXGIMonitorIndex(result.Stdout),
		DXGIStdout: result.Stdout,
		DXGIStderr: result.Stderr,
	}
	if err != nil {
		return meta, err
	}
	return meta, nil
}

func parseDXGIMonitorIndex(output string) int {
	output = strings.TrimSpace(output)
	if output == "" {
		return -1
	}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "monitor=") {
			continue
		}
		value := strings.TrimPrefix(line, "monitor=")
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return parsed
		}
	}
	return -1
}
