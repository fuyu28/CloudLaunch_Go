//go:build windows

package services

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

const (
	screenClipTimeout  = 30 * time.Second
	screenClipPollWait = 250 * time.Millisecond
)

var (
	errClipboardImageNotFound = errors.New("clipboard image not found")
	procClipboardSeqNumber    = user32.NewProc("GetClipboardSequenceNumber")
)

// CaptureForegroundWindow はホットキー経由で前面ウィンドウを撮影する。
func (service *ScreenshotService) CaptureForegroundWindow(ctx context.Context, target CaptureTarget) (string, string, error) {
	return service.captureForegroundWindowWithMode(ctx, target, true)
}

// CaptureForegroundWindowFull はホットキー経由で前面ウィンドウ全体を撮影する。
func (service *ScreenshotService) CaptureForegroundWindowFull(ctx context.Context, target CaptureTarget) (string, string, error) {
	return service.captureForegroundWindowWithMode(ctx, target, false)
}

func (service *ScreenshotService) captureForegroundWindowWithMode(ctx context.Context, target CaptureTarget, clientOnly bool) (string, string, error) {
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

	service.logCapture(
		slog.LevelInfo,
		"ホットキーキャプチャ開始",
		"gameId", game.ID,
		"title", game.Title,
		"exePath", game.ExePath,
		"pid", pid,
		"hwnd", target.HWND,
		"fallback", target.FromFallback,
		"clientOnly", clientOnly,
		"procPath", proc.Cmd,
		"output", fullPath,
	)

	if err := service.captureWithScreenClip(ctx, fullPath, tmpPath); err != nil {
		if errors.Is(err, ErrNoNewScreenshot) {
			service.logCapture(slog.LevelInfo, "スクリーンショットが取得されなかったため保存をスキップ", "gameId", game.ID)
			return "", "", err
		}
		service.logCapture(slog.LevelWarn, "スクリーンショット取得に失敗", "error", err)
		return "", "", err
	}

	return game.ID, fullPath, nil
}

func (service *ScreenshotService) captureWithScreenClip(ctx context.Context, fullPath string, tmpPath string) error {
	beforeSeq := clipboardSequenceNumber()
	beforeHash, _ := readClipboardImageHash(ctx)

	command := execCommandHidden(ctx, "explorer.exe", "ms-screenclip:")
	if err := command.Start(); err != nil {
		return fmt.Errorf("failed to start ms-screenclip: %w", err)
	}

	imageBytes, err := waitForNewClipboardImage(ctx, beforeSeq, beforeHash, screenClipTimeout)
	if err != nil {
		return err
	}

	outputPath := fullPath
	if tmpPath != "" {
		outputPath = tmpPath
	}
	if err := os.WriteFile(outputPath, imageBytes, 0o600); err != nil {
		return err
	}
	if tmpPath == "" {
		return nil
	}
	if err := service.convertFileToJpeg(tmpPath, fullPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	_ = os.Remove(tmpPath)
	return nil
}

func waitForNewClipboardImage(ctx context.Context, beforeSeq uint32, beforeHash string, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if time.Now().After(deadline) {
			return nil, ErrNoNewScreenshot
		}

		seq := clipboardSequenceNumber()
		if seq == beforeSeq {
			time.Sleep(screenClipPollWait)
			continue
		}

		imageBytes, err := readClipboardImageBytes(ctx)
		if err != nil {
			if errors.Is(err, errClipboardImageNotFound) {
				time.Sleep(screenClipPollWait)
				continue
			}
			return nil, err
		}
		hash := hashBytes(imageBytes)
		if hash == beforeHash {
			time.Sleep(screenClipPollWait)
			continue
		}
		return imageBytes, nil
	}
}

func clipboardSequenceNumber() uint32 {
	value, _, _ := procClipboardSeqNumber.Call()
	return uint32(value)
}

func readClipboardImageHash(ctx context.Context) (string, error) {
	imageBytes, err := readClipboardImageBytes(ctx)
	if err != nil {
		return "", err
	}
	return hashBytes(imageBytes), nil
}

func hashBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func readClipboardImageBytes(ctx context.Context) ([]byte, error) {
	command := execCommandHidden(
		ctx,
		"powershell",
		"-NoProfile",
		"-NonInteractive",
		"-Command",
		`Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $img=[Windows.Forms.Clipboard]::GetImage(); if ($null -eq $img) { exit 3 }; $ms=New-Object System.IO.MemoryStream; $img.Save($ms,[System.Drawing.Imaging.ImageFormat]::Png); [Convert]::ToBase64String($ms.ToArray())`,
	)
	output, err := command.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 3 {
			return nil, errClipboardImageNotFound
		}
		return nil, err
	}
	encoded := strings.TrimSpace(string(output))
	if encoded == "" {
		return nil, errClipboardImageNotFound
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	if len(decoded) == 0 {
		return nil, errClipboardImageNotFound
	}
	return decoded, nil
}

func captureWindowByPID(pid int, outputPath string) (CaptureMeta, error) {
	if pid <= 0 {
		return CaptureMeta{}, errors.New("pid is invalid")
	}
	service := ScreenshotService{}
	if err := service.captureWithScreenClip(context.Background(), outputPath, ""); err != nil {
		return CaptureMeta{}, err
	}
	return CaptureMeta{}, nil
}
