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

	credential, error := service.store.Load(ctx, strings.TrimSpace(credentialKey))
	if error != nil {
		service.logger.Error("認証情報取得に失敗", "error", error)
		return result.ErrorResult[storage.UploadSummary]("認証情報取得に失敗しました", error.Error())
	}
	if credential == nil {
		return result.ErrorResult[storage.UploadSummary]("認証情報が見つかりません", "credentialが空です")
	}

	client, error := storage.NewClient(ctx, storage.S3Config{
		Endpoint:       service.config.S3Endpoint,
		Region:         service.config.S3Region,
		Bucket:         service.config.S3Bucket,
		ForcePathStyle: service.config.S3ForcePathStyle,
		UseTLS:         service.config.S3UseTLS,
	}, *credential)
	if error != nil {
		service.logger.Error("S3クライアント作成に失敗", "error", error)
		return result.ErrorResult[storage.UploadSummary]("S3クライアント作成に失敗しました", error.Error())
	}

	summary, error := storage.UploadFolder(ctx, client, service.config.S3Bucket, folderPath, prefix)
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
	credential, error := service.store.Load(ctx, strings.TrimSpace(credentialKey))
	if error != nil {
		service.logger.Error("認証情報取得に失敗", "error", error)
		return result.ErrorResult[bool]("認証情報取得に失敗しました", error.Error())
	}
	if credential == nil {
		return result.ErrorResult[bool]("認証情報が見つかりません", "credentialが空です")
	}

	client, error := storage.NewClient(ctx, storage.S3Config{
		Endpoint:       service.config.S3Endpoint,
		Region:         service.config.S3Region,
		Bucket:         service.config.S3Bucket,
		ForcePathStyle: service.config.S3ForcePathStyle,
		UseTLS:         service.config.S3UseTLS,
	}, *credential)
	if error != nil {
		service.logger.Error("S3クライアント作成に失敗", "error", error)
		return result.ErrorResult[bool]("S3クライアント作成に失敗しました", error.Error())
	}

	if error := storage.SaveMetadata(ctx, client, service.config.S3Bucket, service.config.CloudMetadataKey, metadata); error != nil {
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
	credential, error := service.store.Load(ctx, strings.TrimSpace(credentialKey))
	if error != nil {
		service.logger.Error("認証情報取得に失敗", "error", error)
		return result.ErrorResult[*storage.CloudMetadata]("認証情報取得に失敗しました", error.Error())
	}
	if credential == nil {
		return result.ErrorResult[*storage.CloudMetadata]("認証情報が見つかりません", "credentialが空です")
	}

	client, error := storage.NewClient(ctx, storage.S3Config{
		Endpoint:       service.config.S3Endpoint,
		Region:         service.config.S3Region,
		Bucket:         service.config.S3Bucket,
		ForcePathStyle: service.config.S3ForcePathStyle,
		UseTLS:         service.config.S3UseTLS,
	}, *credential)
	if error != nil {
		service.logger.Error("S3クライアント作成に失敗", "error", error)
		return result.ErrorResult[*storage.CloudMetadata]("S3クライアント作成に失敗しました", error.Error())
	}

	metadata, error := storage.LoadMetadata(ctx, client, service.config.S3Bucket, service.config.CloudMetadataKey)
	if error != nil {
		service.logger.Error("メタ情報取得に失敗", "error", error)
		return result.ErrorResult[*storage.CloudMetadata]("メタ情報取得に失敗しました", error.Error())
	}
	return result.OkResult(metadata)
}

// validateCloudInput はクラウドアップロード入力の基本チェックを行う。
func validateCloudInput(credentialKey string, folderPath string) error {
	if strings.TrimSpace(credentialKey) == "" {
		return errors.New("credentialKeyが空です")
	}
	if strings.TrimSpace(folderPath) == "" {
		return errors.New("folderPathが空です")
	}
	return nil
}
