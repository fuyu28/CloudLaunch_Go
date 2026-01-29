// @fileoverview 認証情報の保存・取得・削除を提供する。
package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"CloudLaunch_Go/internal/credentials"
	"CloudLaunch_Go/internal/result"
)

// CredentialService は認証情報管理を提供する。
type CredentialService struct {
	store  credentials.Store
	logger *slog.Logger
}

// NewCredentialService は CredentialService を生成する。
func NewCredentialService(store credentials.Store, logger *slog.Logger) *CredentialService {
	return &CredentialService{store: store, logger: logger}
}

// SaveCredential は認証情報を保存する。
func (service *CredentialService) SaveCredential(ctx context.Context, key string, input CredentialInput) result.ApiResult[bool] {
	if error := validateCredentialInput(input); error != nil {
		return result.ErrorResult[bool]("認証情報が不正です", error.Error())
	}

	credential := credentials.Credential{
		AccessKeyID:     strings.TrimSpace(input.AccessKeyID),
		SecretAccessKey: strings.TrimSpace(input.SecretAccessKey),
		BucketName:      strings.TrimSpace(input.BucketName),
		Region:          strings.TrimSpace(input.Region),
		Endpoint:        strings.TrimSpace(input.Endpoint),
	}

	if error := service.store.Save(ctx, strings.TrimSpace(key), credential); error != nil {
		service.logger.Error("認証情報保存に失敗", "error", error)
		return result.ErrorResult[bool]("認証情報保存に失敗しました", error.Error())
	}
	return result.OkResult(true)
}

// LoadCredential は認証情報を取得する。
func (service *CredentialService) LoadCredential(ctx context.Context, key string) result.ApiResult[*credentials.Credential] {
	credential, error := service.store.Load(ctx, strings.TrimSpace(key))
	if error != nil {
		service.logger.Error("認証情報取得に失敗", "error", error)
		return result.ErrorResult[*credentials.Credential]("認証情報取得に失敗しました", error.Error())
	}
	return result.OkResult(credential)
}

// DeleteCredential は認証情報を削除する。
func (service *CredentialService) DeleteCredential(ctx context.Context, key string) result.ApiResult[bool] {
	trimmedKey, detail, ok := requireNonEmpty(key, "key")
	if !ok {
		return result.ErrorResult[bool]("キーが不正です", detail)
	}
	if error := service.store.Delete(ctx, trimmedKey); error != nil {
		service.logger.Error("認証情報削除に失敗", "error", error)
		return result.ErrorResult[bool]("認証情報削除に失敗しました", error.Error())
	}
	return result.OkResult(true)
}

// CredentialInput は認証情報入力を表す。
type CredentialInput struct {
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Region          string
	Endpoint        string
}

// CredentialOutput はUIに返す認証情報の最小情報を表す。
type CredentialOutput struct {
	AccessKeyID string
	BucketName  string
	Region      string
	Endpoint    string
}

// validateCredentialInput は認証情報入力の基本チェックを行う。
func validateCredentialInput(input CredentialInput) error {
	if _, detail, ok := requireNonEmpty(input.BucketName, "bucketName"); !ok {
		return errors.New(detail)
	}
	if _, detail, ok := requireNonEmpty(input.Region, "region"); !ok {
		return errors.New(detail)
	}
	if _, detail, ok := requireNonEmpty(input.Endpoint, "endpoint"); !ok {
		return errors.New(detail)
	}
	if _, detail, ok := requireNonEmpty(input.AccessKeyID, "accessKeyID"); !ok {
		return errors.New(detail)
	}
	if _, detail, ok := requireNonEmpty(input.SecretAccessKey, "secretAccessKey"); !ok {
		return errors.New(detail)
	}
	return nil
}
