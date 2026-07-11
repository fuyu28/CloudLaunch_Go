// クラウド接続に共通する S3 設定解決とヘルパを提供する。
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
// ForcePathStyle は ValidateCredential（app 層）と揃えないと、接続テストは通るが
// Push/Pull だけ MinIO 等で失敗する経路分裂になる。
func resolveS3Config(base config.Config, credential *credentials.Credential) storage.S3Config {
	return storage.S3Config{
		Endpoint:       util.FirstNonEmpty(credential.Endpoint, base.S3Endpoint),
		Region:         util.FirstNonEmpty(credential.Region, base.S3Region),
		Bucket:         util.FirstNonEmpty(credential.BucketName, base.S3Bucket),
		ForcePathStyle: base.S3ForcePathStyle,
		UseTLS:         base.S3UseTLS,
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
// 既知の画像フォーマット（jpeg/png/gif/webp/bmp/avif）はそれぞれの拡張子に正規化。
// 不明な content-type は ".png" を返すが、これは「拡張子が嘘」になる失敗モード
// （webp バイトを .png 名で保存するなど）の原因なので、新しいフォーマットは
// 必ずこのテーブルに追加する。
func normalizeImageExt(ext string, contentType string) string {
	trimmed := strings.ToLower(strings.TrimSpace(ext))
	if trimmed != "" {
		if strings.HasPrefix(trimmed, ".") {
			return trimmed
		}
		return "." + trimmed
	}
	contentType = strings.ToLower(contentType)
	switch {
	case strings.Contains(contentType, "png"):
		return ".png"
	case strings.Contains(contentType, "gif"):
		return ".gif"
	case strings.Contains(contentType, "jpeg"), strings.Contains(contentType, "jpg"):
		return ".jpg"
	case strings.Contains(contentType, "webp"):
		return ".webp"
	case strings.Contains(contentType, "bmp"):
		return ".bmp"
	case strings.Contains(contentType, "avif"):
		return ".avif"
	default:
		return ".png"
	}
}
