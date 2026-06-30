package services

import (
	"context"
	"strings"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/infrastructure/credentials"
	"CloudLaunch_Go/internal/infrastructure/storage"
	"CloudLaunch_Go/internal/util"
)

// resolveS3Config はアプリ設定と認証情報から S3Config を構築する。
func resolveS3Config(base config.Config, credential *credentials.Credential) storage.S3Config {
	return storage.S3Config{
		Endpoint: util.FirstNonEmpty(credential.Endpoint, base.S3Endpoint),
		Region:   util.FirstNonEmpty(credential.Region, base.S3Region),
		Bucket:   util.FirstNonEmpty(credential.BucketName, base.S3Bucket),
		UseTLS:   base.S3UseTLS,
	}
}

// cloudObjectStore は MemoCloudService が依存するストレージ操作を抽象化する。
type cloudObjectStore interface {
	ListObjects(ctx context.Context, cfg storage.S3Config, credential credentials.Credential, prefix string) ([]storage.ObjectInfo, error)
	UploadBytes(ctx context.Context, cfg storage.S3Config, credential credentials.Credential, key string, payload []byte, contentType string) error
	DownloadObject(ctx context.Context, cfg storage.S3Config, credential credentials.Credential, key string) ([]byte, error)
}

type storageCloudObjectStore struct{}

func (storageCloudObjectStore) ListObjects(ctx context.Context, cfg storage.S3Config, credential credentials.Credential, prefix string) ([]storage.ObjectInfo, error) {
	client, err := storage.NewClient(ctx, cfg, credential)
	if err != nil {
		return nil, err
	}
	return storage.ListObjects(ctx, client, cfg.Bucket, prefix)
}

func (storageCloudObjectStore) UploadBytes(ctx context.Context, cfg storage.S3Config, credential credentials.Credential, key string, payload []byte, contentType string) error {
	client, err := storage.NewClient(ctx, cfg, credential)
	if err != nil {
		return err
	}
	return storage.UploadBytes(ctx, client, cfg.Bucket, key, payload, contentType)
}

func (storageCloudObjectStore) DownloadObject(ctx context.Context, cfg storage.S3Config, credential credentials.Credential, key string) ([]byte, error) {
	client, err := storage.NewClient(ctx, cfg, credential)
	if err != nil {
		return nil, err
	}
	return storage.DownloadObject(ctx, client, cfg.Bucket, key)
}

// normalizeImageExt はコンテンツタイプから画像拡張子を決定する。
func normalizeImageExt(ext string, contentType string) string {
	trimmed := strings.ToLower(strings.TrimSpace(ext))
	if trimmed != "" {
		if strings.HasPrefix(trimmed, ".") {
			return trimmed
		}
		return "." + trimmed
	}
	if strings.Contains(contentType, "png") {
		return ".png"
	}
	if strings.Contains(contentType, "gif") {
		return ".gif"
	}
	if strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg") {
		return ".jpg"
	}
	return ".png"
}
