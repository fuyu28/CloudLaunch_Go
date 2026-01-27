// @fileoverview S3互換ストレージクライアントの生成を提供する。
package storage

import (
	"context"
	"fmt"
	"strings"

	"CloudLaunch_Go/internal/credentials"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscreds "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Config はS3接続に必要な設定を表す。
type S3Config struct {
	Endpoint       string
	Region         string
	Bucket         string
	ForcePathStyle bool
	UseTLS         bool
}

// NewClient は S3Config と認証情報からクライアントを生成する。
func NewClient(ctx context.Context, cfg S3Config, credential credentials.Credential) (*s3.Client, error) {
	awsCfg, error := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(awscreds.NewStaticCredentialsProvider(
			credential.AccessKeyID,
			credential.SecretAccessKey,
			"",
		)),
	)
	if error != nil {
		return nil, error
	}

	options := []func(*s3.Options){
		func(o *s3.Options) {
			o.UsePathStyle = cfg.ForcePathStyle
		},
	}

	endpoint := normalizeEndpoint(cfg.Endpoint, cfg.UseTLS)
	if endpoint != "" {
		options = append(options, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}

	return s3.NewFromConfig(awsCfg, options...), nil
}

// normalizeEndpoint はエンドポイントのスキームを補完する。
func normalizeEndpoint(endpoint string, useTLS bool) string {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed
	}
	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, trimmed)
}
