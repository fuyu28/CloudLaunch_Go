// @fileoverview 認証情報ストアの共通インターフェースを定義する。
package credentials

import "context"

// Credential は S3 互換ストレージ用の認証情報を表す。
type Credential struct {
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Region          string
	Endpoint        string
}

// Store は認証情報の保存・取得・削除を提供する。
type Store interface {
	Save(ctx context.Context, key string, credential Credential) error
	Load(ctx context.Context, key string) (*Credential, error)
	Delete(ctx context.Context, key string) error
}
