// Package main provides the CloudLaunch Go backend entry point.
package main

import (
	"context"
	"log"

	"CloudLaunch_Go/internal/app"
)

func main() {
	ctx := context.Background()
	application, error := app.NewApp(ctx)
	if error != nil {
		log.Fatalf("failed to initialize app: %v", error)
	}
	defer func() {
		if shutdownError := application.Shutdown(ctx); shutdownError != nil {
			log.Printf("failed to shutdown app: %v", shutdownError)
		}
	}()

	// TODO: Wails の起動処理を追加する。
}
