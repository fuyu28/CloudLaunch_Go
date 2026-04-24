package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"CloudLaunch_Go/internal/models"
)

type fakeUploadRepository struct {
	createUploadFn      func(ctx context.Context, upload models.Upload) (*models.Upload, error)
	listUploadsByGameFn func(ctx context.Context, gameID string) ([]models.Upload, error)
}

func (repository fakeUploadRepository) CreateUpload(ctx context.Context, upload models.Upload) (*models.Upload, error) {
	return repository.createUploadFn(ctx, upload)
}

func (repository fakeUploadRepository) ListUploadsByGame(ctx context.Context, gameID string) ([]models.Upload, error) {
	return repository.listUploadsByGameFn(ctx, gameID)
}

func TestUploadServiceCreateUploadUsesRepositoryBoundary(t *testing.T) {
	t.Parallel()

	service := NewUploadService(fakeUploadRepository{
		createUploadFn: func(ctx context.Context, upload models.Upload) (*models.Upload, error) {
			upload.ID = "upload-1"
			return &upload, nil
		},
		listUploadsByGameFn: func(ctx context.Context, gameID string) ([]models.Upload, error) { return nil, nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.CreateUpload(context.Background(), UploadInput{
		Comment: "comment",
		GameID:  "game-1",
	})

	if !result.Success {
		t.Fatalf("expected success, got %#v", result.Error)
	}
	if result.Data == nil || result.Data.ID != "upload-1" {
		t.Fatalf("expected upload to be returned")
	}
}

func TestUploadServiceListUploadsByGameHandlesRepositoryError(t *testing.T) {
	t.Parallel()

	service := NewUploadService(fakeUploadRepository{
		createUploadFn: func(ctx context.Context, upload models.Upload) (*models.Upload, error) {
			return &upload, nil
		},
		listUploadsByGameFn: func(ctx context.Context, gameID string) ([]models.Upload, error) {
			return nil, errors.New("db down")
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.ListUploadsByGame(context.Background(), "game-1")

	if result.Success {
		t.Fatalf("expected failure")
	}
}

func TestUploadServiceCreateUploadRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	service := NewUploadService(fakeUploadRepository{
		createUploadFn: func(ctx context.Context, upload models.Upload) (*models.Upload, error) {
			return &upload, nil
		},
		listUploadsByGameFn: func(ctx context.Context, gameID string) ([]models.Upload, error) { return nil, nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.CreateUpload(context.Background(), UploadInput{
		Comment: "",
		GameID:  "game-1",
	})

	if result.Success {
		t.Fatalf("expected invalid input to fail")
	}
}
