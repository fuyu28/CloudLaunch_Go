// Package main provides the Wails entry point for CloudLaunch.
package main

import (
	"context"
	"embed"
	"fmt"
	"os"
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

	err = wails.Run(&options.App{
		Title:     "CloudLaunch",
		Width:     1200,
		Height:    800,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
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
