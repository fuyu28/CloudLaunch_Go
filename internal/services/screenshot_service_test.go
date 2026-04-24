package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/models"
)

type fakeScreenshotRepository struct {
	getGameByIDFn func(ctx context.Context, gameID string) (*models.Game, error)
}

func (repository fakeScreenshotRepository) GetGameByID(ctx context.Context, gameID string) (*models.Game, error) {
	return repository.getGameByIDFn(ctx, gameID)
}

func TestScreenshotServiceCaptureGameScreenshotRejectsEmptyGameID(t *testing.T) {
	t.Parallel()

	service := NewScreenshotService(config.Config{}, fakeScreenshotRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) {
			return nil, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.CaptureGameScreenshot(context.Background(), "   ")
	if err == nil {
		t.Fatalf("expected empty game id error")
	}
}

func TestScreenshotServiceCaptureGameScreenshotReturnsRepositoryError(t *testing.T) {
	t.Parallel()

	service := NewScreenshotService(config.Config{}, fakeScreenshotRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) {
			return nil, errors.New("db down")
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.CaptureGameScreenshot(context.Background(), "game-1")
	if err == nil {
		t.Fatalf("expected repository error")
	}
}

func TestScreenshotServiceBuildScreenshotPathsUsesConfiguredExtension(t *testing.T) {
	t.Parallel()

	service := NewScreenshotService(config.Config{ScreenshotLocalJpeg: true}, fakeScreenshotRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) {
			return nil, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	fullPath, tmpPath, err := service.buildScreenshotPaths("game-1", t.TempDir())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.HasSuffix(fullPath, ".jpg") {
		t.Fatalf("expected jpg path, got %s", fullPath)
	}
	if !strings.HasSuffix(tmpPath, ".tmp.png") {
		t.Fatalf("expected tmp png path, got %s", tmpPath)
	}
	if filepath.Dir(fullPath) != filepath.Dir(tmpPath) {
		t.Fatalf("expected same output directory")
	}
}
