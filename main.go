// Package main provides the Wails entry point for CloudLaunch.
package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"

	"CloudLaunch_Go/internal/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	ctx := context.Background()
	backend, err := app.NewApp(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to initialize app: %v\n", err)
		panic(err)
	}

	// 想定外の panic はログ（error.log）に残してから再送出する。
	// バックグラウンド goroutine の panic は各 goroutine 側で回収するため、
	// ここで拾うのは主に起動・実行系の致命的な panic。
	defer func() {
		if r := recover(); r != nil {
			if backend.Logger != nil {
				backend.Logger.Error("致命的な panic でアプリが停止しました",
					"panic", fmt.Sprintf("%v", r), "stack", string(debug.Stack()))
			}
			panic(r)
		}
	}()

	// フレームレスは Windows のみ。Windows では独自のタイトルバー（最小化/最大化/閉じる）を
	// 描画する。macOS / Linux ではネイティブのウィンドウ装飾を使い、独自ボタンは表示しない
	// （フロント側でプラットフォームを判定して出し分ける）。
	frameless := runtime.GOOS == "windows"

	err = wails.Run(&options.App{
		Title:     "CloudLaunch",
		Width:     1200,
		Height:    800,
		Frameless: frameless,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// cloudlaunch テーマの base-200 に合わせた淡い色。初期描画の暗いちらつきや
		// フレームレス時の角の隙間が目立たないようにする。
		BackgroundColour: &options.RGBA{R: 243, G: 243, B: 247, A: 1},
		OnStartup:        backend.Startup,
		OnShutdown: func(ctx context.Context) {
			_ = backend.Shutdown(ctx)
		},
		Bind: []interface{}{
			backend,
		},
	})
	if err != nil {
		backend.Logger.Error("アプリの実行に失敗しました", "error", err)
		panic(err)
	}
}
