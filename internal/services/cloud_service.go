// @fileoverview クラウドストレージ連携（S3互換）を提供する。
package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/credentials"
	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/storage"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// CloudService はクラウドストレージ操作を提供する。
type CloudService struct {
	config config.Config
	store  credentials.Store
	logger *slog.Logger
}

// NewCloudService は CloudService を生成する。
func NewCloudService(cfg config.Config, store credentials.Store, logger *slog.Logger) *CloudService {
	return &CloudService{config: cfg, store: store, logger: logger}
}

// UploadFolder はフォルダをクラウドへアップロードする。
func (service *CloudService) UploadFolder(
	ctx context.Context,
	credentialKey string,
	folderPath string,
	prefix string,
) result.ApiResult[storage.UploadSummary] {
	if error := validateCloudInput(credentialKey, folderPath); error != nil {
		return result.ErrorResult[storage.UploadSummary]("アップロード入力が不正です", error.Error())
	}

	client, cfg, message, detail, ok := service.newClient(ctx, credentialKey)
	if !ok {
		return result.ErrorResult[storage.UploadSummary](message, detail)
	}

	summary, error := storage.UploadFolder(ctx, client, cfg.Bucket, folderPath, prefix, service.config.S3UploadConcurrency)
	if error != nil {
		service.logger.Error("フォルダアップロードに失敗", "error", error)
		return result.ErrorResult[storage.UploadSummary]("フォルダアップロードに失敗しました", error.Error())
	}
	return result.OkResult(summary)
}

// SaveCloudMetadata はメタ情報をクラウドに保存する。
func (service *CloudService) SaveCloudMetadata(
	ctx context.Context,
	credentialKey string,
	metadata storage.CloudMetadata,
) result.ApiResult[bool] {
	client, cfg, message, detail, ok := service.newClient(ctx, credentialKey)
	if !ok {
		return result.ErrorResult[bool](message, detail)
	}

	if error := storage.SaveMetadata(ctx, client, cfg.Bucket, service.config.CloudMetadataKey, metadata); error != nil {
		service.logger.Error("メタ情報保存に失敗", "error", error)
		return result.ErrorResult[bool]("メタ情報保存に失敗しました", error.Error())
	}
	return result.OkResult(true)
}

// LoadCloudMetadata はクラウドからメタ情報を取得する。
func (service *CloudService) LoadCloudMetadata(
	ctx context.Context,
	credentialKey string,
) result.ApiResult[*storage.CloudMetadata] {
	client, cfg, message, detail, ok := service.newClient(ctx, credentialKey)
	if !ok {
		return result.ErrorResult[*storage.CloudMetadata](message, detail)
	}

	metadata, error := storage.LoadMetadata(ctx, client, cfg.Bucket, service.config.CloudMetadataKey)
	if error != nil {
		service.logger.Error("メタ情報取得に失敗", "error", error)
		return result.ErrorResult[*storage.CloudMetadata]("メタ情報取得に失敗しました", error.Error())
	}
	return result.OkResult(metadata)
}

// validateCloudInput はクラウドアップロード入力の基本チェックを行う。
func validateCloudInput(credentialKey string, folderPath string) error {
	if _, detail, ok := requireNonEmpty(credentialKey, "credentialKey"); !ok {
		return errors.New(detail)
	}
	if _, detail, ok := requireNonEmpty(folderPath, "folderPath"); !ok {
		return errors.New(detail)
	}
	return nil
}

func resolveS3Config(base config.Config, credential *credentials.Credential) storage.S3Config {
	return storage.S3Config{
		Endpoint:       firstNonEmpty(credential.Endpoint, base.S3Endpoint),
		Region:         firstNonEmpty(credential.Region, base.S3Region),
		Bucket:         firstNonEmpty(credential.BucketName, base.S3Bucket),
		ForcePathStyle: base.S3ForcePathStyle,
		UseTLS:         base.S3UseTLS,
	}
}

func (service *CloudService) newClient(
	ctx context.Context,
	credentialKey string,
) (*s3.Client, storage.S3Config, string, string, bool) {
	credential, error := service.store.Load(ctx, strings.TrimSpace(credentialKey))
	if error != nil {
		service.logger.Error("認証情報取得に失敗", "error", error)
		return nil, storage.S3Config{}, "認証情報取得に失敗しました", error.Error(), false
	}
	if credential == nil {
		return nil, storage.S3Config{}, "認証情報が見つかりません", "credentialが空です", false
	}

	cfg := resolveS3Config(service.config, credential)
	client, error := storage.NewClient(ctx, cfg, *credential)
	if error != nil {
		service.logger.Error("S3クライアント作成に失敗", "error", error)
		return nil, cfg, "S3クライアント作成に失敗しました", error.Error(), false
	}
	return client, cfg, "", "", true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
