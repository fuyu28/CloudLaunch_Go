// screencap-cli.exe 連携のうち、プラットフォームに依存しない純粋ロジックを提供する。
package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// buildScreencapArgs は screencap-cli.exe の cap サブコマンド引数を組み立てる。
func buildScreencapArgs(pid int, outPath string, localJpeg bool, jpegQuality int, clientOnly bool) []string {
	args := []string{"cap", "--method", "wgc-window"}
	if pid > 0 {
		args = append(args, "--pid", strconv.Itoa(pid))
	} else {
		args = append(args, "--foreground")
	}
	args = append(args, "--out", outPath, "--json", "--overwrite", "--no-log")
	if clientOnly {
		args = append(args, "--crop", "client")
	}
	if localJpeg {
		args = append(args, "--format", "jpg", "--quality", strconv.Itoa(normalizeJpegQuality(jpegQuality)))
	}
	return args
}

func normalizeJpegQuality(value int) int {
	if value < 1 || value > 100 {
		return 85
	}
	return value
}

// screencapResult は screencap-cli の --json 出力（成功/失敗共通）を表す。使用するフィールドのみ定義する。
type screencapResult struct {
	OK         bool                `json:"ok"`
	OutPath    string              `json:"out_path"`
	ImageStats screencapImageStats `json:"image_stats"`
	Error      *screencapError     `json:"error"`
}

// screencapImageStats はキャプチャ画像の統計情報。使用するフィールドのみ定義する。
// black_ratio は真っ黒キャプチャの検知に使う。
type screencapImageStats struct {
	BlackRatio float64 `json:"black_ratio"`
}

// screencapError は失敗時のエラー情報。hresult は数値/文字列いずれの表現もあり得るため RawMessage で安全に受ける。
type screencapError struct {
	Message string          `json:"message"`
	Where   string          `json:"where"`
	HResult json.RawMessage `json:"hresult"`
}

// parseScreencapResult は stdout の結果JSONを1オブジェクトとして解釈する。
func parseScreencapResult(stdout []byte) (*screencapResult, error) {
	trimmed := bytes.TrimSpace(stdout)
	if len(trimmed) == 0 {
		return nil, errors.New("screencap-cli の出力が空です")
	}
	var result screencapResult
	if err := json.Unmarshal(trimmed, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// screencapErrorFromResult は screencap-cli の結果JSONから失敗エラーを組み立てる。
// error 情報があれば message/where/hresult を含め、無ければ汎用文言を返す。
func screencapErrorFromResult(result *screencapResult) error {
	if result != nil && result.Error != nil {
		return fmt.Errorf(
			"screencap-cli に失敗しました: %s (where=%s, hresult=%s)",
			result.Error.Message,
			result.Error.Where,
			string(result.Error.HResult),
		)
	}
	return errors.New("screencap-cli に失敗しました")
}

// screencapExitError は screencap-cli の異常終了を表す。stderr を優先し、無ければ stdout を生テキストで含める。
func screencapExitError(exitCode int, stdout, stderr string) error {
	raw := strings.TrimSpace(stderr)
	if raw == "" {
		raw = strings.TrimSpace(stdout)
	}
	return fmt.Errorf("screencap-cli に失敗しました (exit=%d): %s", exitCode, raw)
}
