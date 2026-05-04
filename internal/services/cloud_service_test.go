package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/credentials"
	"CloudLaunch_Go/internal/storage"
)

type fakeCloudObjectStore struct {
	uploadSummary   storage.UploadSummary
	loadMetadata    *storage.CloudMetadata
	uploadErr       error
	saveMetadataErr error
	loadMetadataErr error
	listObjects     []storage.ObjectInfo
	listObjectsErr  error
	uploadedKey     string
}

func (store *fakeCloudObjectStore) UploadFolder(ctx context.Context, cfg storage.S3Config, credential credentials.Credential, folderPath string, prefix string, concurrency int, retryCount int) (storage.UploadSummary, error) {
	return store.uploadSummary, store.uploadErr
}

func (store *fakeCloudObjectStore) SaveMetadata(ctx context.Context, cfg storage.S3Config, credential credentials.Credential, key string, metadata storage.CloudMetadata) error {
	return store.saveMetadataErr
}

func (store *fakeCloudObjectStore) LoadMetadata(ctx context.Context, cfg storage.S3Config, credential credentials.Credential, key string) (*storage.CloudMetadata, error) {
	return store.loadMetadata, store.loadMetadataErr
}

func (store *fakeCloudObjectStore) ListObjects(ctx context.Context, cfg storage.S3Config, credential credentials.Credential, prefix string) ([]storage.ObjectInfo, error) {
	return store.listObjects, store.listObjectsErr
}

func (store *fakeCloudObjectStore) UploadBytes(ctx context.Context, cfg storage.S3Config, credential credentials.Credential, key string, payload []byte, contentType string) error {
	store.uploadedKey = key
	return nil
}

func (store *fakeCloudObjectStore) DownloadObject(ctx context.Context, cfg storage.S3Config, credential credentials.Credential, key string) ([]byte, error) {
	return nil, nil
}

func TestCloudServiceUploadFolderUsesObjectStorePort(t *testing.T) {
	t.Parallel()

	service := NewCloudService(config.Config{}, &fakeCredentialStore{
		loadResult: &credentials.Credential{
			AccessKeyID:     "access",
			SecretAccessKey: "secret",
			BucketName:      "bucket",
			Region:          "region",
			Endpoint:        "endpoint",
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.objectStore = &fakeCloudObjectStore{
		uploadSummary: storage.UploadSummary{FileCount: 2, TotalBytes: 123},
	}

	result := service.UploadFolder(context.Background(), "default", "/tmp/save", "games/game-1")
	if !result.Success {
		t.Fatalf("expected success, got %#v", result.Error)
	}
	if result.Data.FileCount != 2 || result.Data.TotalBytes != 123 {
		t.Fatalf("unexpected upload summary: %#v", result.Data)
	}
}

func TestCloudServiceLoadCloudMetadataReturnsStoreError(t *testing.T) {
	t.Parallel()

	service := NewCloudService(config.Config{}, &fakeCredentialStore{
		loadResult: &credentials.Credential{
			AccessKeyID:     "access",
			SecretAccessKey: "secret",
			BucketName:      "bucket",
			Region:          "region",
			Endpoint:        "endpoint",
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.objectStore = &fakeCloudObjectStore{loadMetadataErr: errors.New("boom")}

	result := service.LoadCloudMetadata(context.Background(), "default")
	if result.Success {
		t.Fatal("expected error result")
	}
}

func TestCloudServiceSaveCloudMetadataUsesResolvedCredentialConfig(t *testing.T) {
	t.Parallel()

	service := NewCloudService(config.Config{CloudMetadataKey: "metadata.json"}, &fakeCredentialStore{
		loadResult: &credentials.Credential{
			AccessKeyID:     "access",
			SecretAccessKey: "secret",
			BucketName:      "bucket",
			Region:          "region",
			Endpoint:        "endpoint",
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.objectStore = &fakeCloudObjectStore{}

	result := service.SaveCloudMetadata(context.Background(), "default", storage.CloudMetadata{
		Version:   1,
		UpdatedAt: time.Now(),
	})
	if !result.Success {
		t.Fatalf("expected success, got %#v", result.Error)
	}
}
