package services

import (
	"context"
	"errors"
	"os"

	"CloudLaunch_Go/internal/infrastructure/storage"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type cloudSyncStorage interface {
	LoadMetadata(ctx context.Context, client *s3.Client, bucket string, key string) (*storage.CloudMetadata, error)
	SaveMetadata(ctx context.Context, client *s3.Client, bucket string, key string, metadata storage.CloudMetadata) error
	DeleteObjectsByPrefix(ctx context.Context, client *s3.Client, bucket string, prefix string) error
	SaveSessions(ctx context.Context, client *s3.Client, bucket string, key string, sessions []storage.CloudSessionRecord) error
	LoadSessions(ctx context.Context, client *s3.Client, bucket string, key string) ([]storage.CloudSessionRecord, error)
	UploadBytes(ctx context.Context, client *s3.Client, bucket string, key string, payload []byte, contentType string) error
	DownloadObject(ctx context.Context, client *s3.Client, bucket string, key string) ([]byte, error)
}

type storageCloudSyncStorage struct{}

type cloudImageFileStore interface {
	EnsureDir(path string) error
	Exists(path string) (bool, error)
	WriteFile(path string, payload []byte, perm os.FileMode) error
}

type osCloudImageFileStore struct{}

type cloudImageLoader interface {
	Load(path string) ([]byte, string, string, error)
}

type defaultCloudImageLoader struct{}

func (defaultCloudImageLoader) Load(path string) ([]byte, string, string, error) {
	return loadImagePayload(path)
}

func (osCloudImageFileStore) EnsureDir(path string) error {
	return os.MkdirAll(path, 0o700)
}

func (osCloudImageFileStore) Exists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (osCloudImageFileStore) WriteFile(path string, payload []byte, perm os.FileMode) error {
	return os.WriteFile(path, payload, perm)
}

func (storageCloudSyncStorage) LoadMetadata(ctx context.Context, client *s3.Client, bucket string, key string) (*storage.CloudMetadata, error) {
	return storage.LoadMetadata(ctx, client, bucket, key)
}

func (storageCloudSyncStorage) SaveMetadata(ctx context.Context, client *s3.Client, bucket string, key string, metadata storage.CloudMetadata) error {
	return storage.SaveMetadata(ctx, client, bucket, key, metadata)
}

func (storageCloudSyncStorage) DeleteObjectsByPrefix(ctx context.Context, client *s3.Client, bucket string, prefix string) error {
	return storage.DeleteObjectsByPrefix(ctx, client, bucket, prefix)
}

func (storageCloudSyncStorage) SaveSessions(ctx context.Context, client *s3.Client, bucket string, key string, sessions []storage.CloudSessionRecord) error {
	return storage.SaveSessions(ctx, client, bucket, key, sessions)
}

func (storageCloudSyncStorage) LoadSessions(ctx context.Context, client *s3.Client, bucket string, key string) ([]storage.CloudSessionRecord, error) {
	return storage.LoadSessions(ctx, client, bucket, key)
}

func (storageCloudSyncStorage) UploadBytes(ctx context.Context, client *s3.Client, bucket string, key string, payload []byte, contentType string) error {
	return storage.UploadBytes(ctx, client, bucket, key, payload, contentType)
}

func (storageCloudSyncStorage) DownloadObject(ctx context.Context, client *s3.Client, bucket string, key string) ([]byte, error) {
	return storage.DownloadObject(ctx, client, bucket, key)
}
