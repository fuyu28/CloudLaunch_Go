//go:build windows

// Windows向け screencap-cli によるスクリーンショット撮影を実装する。
package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"CloudLaunch_Go/internal/config"
)

const (
	// screencapTimeout は screencap-cli.exe 実行のタイムアウト。
	screencapTimeout = 10 * time.Second
	// screencapBlackWarnRatio を超える真っ黒率で警告する（最小化中の可能性）。
	screencapBlackWarnRatio = 0.98
)

// screencapPathCache は解決済み CLI パスのメモ化（プロセス生存中は不変）。
var (
	screencapPathMu    sync.Mutex
	screencapPathCache string
)

// captureWithScreencap は同梱の screencap-cli.exe を呼び出して outPath に画像を保存する。
// pid が 0 のときはフォアグラウンドウィンドウを対象にする。
func (service *ScreenshotService) captureWithScreencap(ctx context.Context, pid int, outPath string) error {
	cliPath, err := resolveScreencapCLIPath()
	if err != nil {
		return err
	}

	args := buildScreencapArgs(pid, outPath, service.localJpeg, service.jpegQuality, service.clientOnly)

	runCtx, cancel := context.WithTimeout(ctx, screencapTimeout)
	defer cancel()

	command := execCommandHidden(runCtx, cliPath, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	runErr := command.Run()
	if runErr == nil {
		result, parseErr := parseScreencapResult(stdout.Bytes())
		if parseErr != nil {
			// exit 0 でも結果JSONを解釈できない場合は、出力ファイルの実在で成否を判断する。
			if _, statErr := os.Stat(outPath); statErr != nil {
				return fmt.Errorf(
					"screencap-cli は正常終了しましたが出力ファイルが確認できません: %v (parse=%v)",
					statErr,
					parseErr,
				)
			}
			service.logCapture(slog.LevelWarn, "screencap-cli の結果JSONを解析できませんでした", "error", parseErr)
			return nil
		}
		if !result.OK {
			return screencapErrorFromResult(result)
		}
		if result.ImageStats.BlackRatio > screencapBlackWarnRatio {
			service.logCapture(
				slog.LevelWarn,
				"キャプチャがほぼ真っ黒（最小化中の可能性）",
				"blackRatio", result.ImageStats.BlackRatio,
				"outPath", result.OutPath,
			)
		}
		service.logCapture(slog.LevelDebug, "screencap-cli 実行成功", "outPath", result.OutPath)
		return nil
	}

	// タイムアウトは他の失敗と区別して明示する。
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("screencap-cli がタイムアウトしました (%s)", screencapTimeout)
	}

	// 失敗時は結果JSONの error 情報を優先し、解析できなければ生の出力を含めて返す。
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		if result, parseErr := parseScreencapResult(stdout.Bytes()); parseErr == nil && result.Error != nil {
			return screencapErrorFromResult(result)
		}
		return screencapExitError(exitErr.ExitCode(), stdout.String(), stderr.String())
	}
	return fmt.Errorf("screencap-cli の起動に失敗しました: %w", runErr)
}

// resolveScreencapCLIPath は実行ファイルと同じディレクトリの screencap-cli.exe のみを解決する。
// ピン留めしたバージョン以外の野良バイナリを拾わないよう PATH フォールバックは行わない。
// 解決結果はプロセス生存中不変なため成功時のみメモ化する（エラーはキャッシュしない）。
func resolveScreencapCLIPath() (string, error) {
	screencapPathMu.Lock()
	defer screencapPathMu.Unlock()
	if screencapPathCache != "" {
		return screencapPathCache, nil
	}

	dir := config.ExecutableDir()
	if dir == "" {
		return "", errors.New("screencap-cli.exe の配置先を特定できません（実行ファイルパスを解決できません）")
	}
	candidate := filepath.Join(dir, "screencap-cli.exe")
	if _, statErr := os.Stat(candidate); statErr != nil {
		return "", fmt.Errorf("screencap-cli.exe が見つかりません: %s", candidate)
	}
	screencapPathCache = candidate
	return candidate, nil
}
