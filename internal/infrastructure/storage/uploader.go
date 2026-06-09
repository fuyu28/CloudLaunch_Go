// @fileoverview S3 へのファイル/フォルダアップロードを提供する。
package storage

import (
	"bytes"
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	defaultUploadConcurrency = 6
)

// UploadBytes は任意のバイト列をアップロードする。
func UploadBytes(ctx context.Context, client *s3.Client, bucket string, key string, payload []byte, contentType string) error {
	reader := bytes.NewReader(payload)
	input := &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   reader,
	}
	if strings.TrimSpace(contentType) != "" {
		input.ContentType = stringPtr(contentType)
	}
	_, error := client.PutObject(ctx, input)
	return error
}

// stringPtr は文字列のポインタを返す。
func stringPtr(value string) *string {
	return &value
}
