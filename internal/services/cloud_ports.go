package services

import (
	"context"

	"CloudLaunch_Go/internal/credentials"
	"CloudLaunch_Go/internal/storage"
)

type cloudObjectStore interface {
	UploadFolder(
		ctx context.Context,
		cfg storage.S3Config,
		credential credentials.Credential,
		folderPath string,
		prefix string,
		concurrency int,
		retryCount int,
	) (storage.UploadSummary, error)
	SaveMetadata(
		ctx context.Context,
		cfg storage.S3Config,
		credential credentials.Credential,
		key string,
		metadata storage.CloudMetadata,
	) error
	LoadMetadata(
		ctx context.Context,
		cfg storage.S3Config,
		credential credentials.Credential,
		key string,
	) (*storage.CloudMetadata, error)
	ListObjects(
		ctx context.Context,
		cfg storage.S3Config,
		credential credentials.Credential,
		prefix string,
	) ([]storage.ObjectInfo, error)
	UploadBytes(
		ctx context.Context,
		cfg storage.S3Config,
		credential credentials.Credential,
		key string,
		payload []byte,
		contentType string,
	) error
	DownloadObject(
		ctx context.Context,
		cfg storage.S3Config,
		credential credentials.Credential,
		key string,
	) ([]byte, error)
}

type storageCloudObjectStore struct{}

func (storageCloudObjectStore) UploadFolder(
	ctx context.Context,
	cfg storage.S3Config,
	credential credentials.Credential,
	folderPath string,
	prefix string,
	concurrency int,
	retryCount int,
) (storage.UploadSummary, error) {
	client, err := storage.NewClient(ctx, cfg, credential)
	if err != nil {
		return storage.UploadSummary{}, err
	}
	return storage.UploadFolder(ctx, client, cfg.Bucket, folderPath, prefix, concurrency, retryCount)
}

func (storageCloudObjectStore) SaveMetadata(
	ctx context.Context,
	cfg storage.S3Config,
	credential credentials.Credential,
	key string,
	metadata storage.CloudMetadata,
) error {
	client, err := storage.NewClient(ctx, cfg, credential)
	if err != nil {
		return err
	}
	return storage.SaveMetadata(ctx, client, cfg.Bucket, key, metadata)
}

func (storageCloudObjectStore) LoadMetadata(
	ctx context.Context,
	cfg storage.S3Config,
	credential credentials.Credential,
	key string,
) (*storage.CloudMetadata, error) {
	client, err := storage.NewClient(ctx, cfg, credential)
	if err != nil {
		return nil, err
	}
	return storage.LoadMetadata(ctx, client, cfg.Bucket, key)
}

func (storageCloudObjectStore) ListObjects(
	ctx context.Context,
	cfg storage.S3Config,
	credential credentials.Credential,
	prefix string,
) ([]storage.ObjectInfo, error) {
	client, err := storage.NewClient(ctx, cfg, credential)
	if err != nil {
		return nil, err
	}
	return storage.ListObjects(ctx, client, cfg.Bucket, prefix)
}

func (storageCloudObjectStore) UploadBytes(
	ctx context.Context,
	cfg storage.S3Config,
	credential credentials.Credential,
	key string,
	payload []byte,
	contentType string,
) error {
	client, err := storage.NewClient(ctx, cfg, credential)
	if err != nil {
		return err
	}
	return storage.UploadBytes(ctx, client, cfg.Bucket, key, payload, contentType)
}

func (storageCloudObjectStore) DownloadObject(
	ctx context.Context,
	cfg storage.S3Config,
	credential credentials.Credential,
	key string,
) ([]byte, error) {
	client, err := storage.NewClient(ctx, cfg, credential)
	if err != nil {
		return nil, err
	}
	return storage.DownloadObject(ctx, client, cfg.Bucket, key)
}
