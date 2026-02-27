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

	"CloudLaunch_Go/internal/models"
)

const (
	screenClipTimeout  = 30 * time.Second
	screenClipPollWait = 250 * time.Millisecond
	hotkeyDefaultDirID = "default"
)

var (
	errClipboardImageNotFound = errors.New("clipboard image not found")
	procClipboardSeqNumber    = user32.NewProc("GetClipboardSequenceNumber")
)

// CaptureHotkey はホットキー経由でSnipping Toolを起動し、画像を保存する。
func (service *ScreenshotService) CaptureHotkey(ctx context.Context) (string, string, error) {
	game, err := service.findCurrentPlayingGame(ctx)
	if err != nil {
		return "", "", err
	}
	gameID := hotkeyDefaultDirID
	gameTitle := "default"
	gameExePath := ""
	if game != nil {
		gameID = game.ID
		gameTitle = game.Title
		gameExePath = game.ExePath
	}

	baseDir := strings.TrimSpace(service.appDataDir)
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	saveDir := filepath.Join(baseDir, "screenshots", gameID)
	fullPath, tmpPath, err := service.buildScreenshotPaths(gameID, saveDir)
	if err != nil {
		return "", "", err
	}

	service.logCapture(
		slog.LevelInfo,
		"ホットキーキャプチャ開始",
		"gameId", gameID,
		"title", gameTitle,
		"exePath", gameExePath,
		"output", fullPath,
	)

	if err := service.captureWithScreenClip(ctx, fullPath, tmpPath); err != nil {
		if errors.Is(err, ErrNoNewScreenshot) {
			service.logCapture(slog.LevelInfo, "スクリーンショットが取得されなかったため保存をスキップ", "gameId", gameID)
			return "", "", err
		}
		service.logCapture(slog.LevelWarn, "スクリーンショット取得に失敗", "error", err)
		return "", "", err
	}

	if game == nil {
		return "", fullPath, nil
	}
	return game.ID, fullPath, nil
}

func (service *ScreenshotService) findCurrentPlayingGame(ctx context.Context) (*models.Game, error) {
	games, err := service.repository.ListGames(ctx, "", models.PlayStatusPlaying, "lastPlayed", "desc")
	if err != nil {
		return nil, err
	}
	if len(games) == 0 {
		return nil, nil
	}
	game := games[0]
	return &game, nil
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
