// Package main provides the Wails entry point for CloudLaunch.
package main

import (
	"context"
	"embed"

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
		panic(err)
	}

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
		panic(err)
	}
}
