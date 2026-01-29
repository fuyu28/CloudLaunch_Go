// @fileoverview S3 へのファイル/フォルダアップロードを提供する。
package storage

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// UploadSummary はアップロード結果の要約を表す。
type UploadSummary struct {
	FileCount  int
	TotalBytes int64
	Keys       []string
}

// UploadFile は単一ファイルをアップロードする。
func UploadFile(ctx context.Context, client *s3.Client, bucket string, key string, filePath string) (size int64, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	info, err := file.Stat()
	if err != nil {
		return 0, err
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   file,
	})
	if err != nil {
		return 0, err
	}

	return info.Size(), nil
}

// UploadFolder はフォルダ配下のファイルをすべてアップロードする。
func UploadFolder(ctx context.Context, client *s3.Client, bucket string, folderPath string, prefix string) (UploadSummary, error) {
	summary := UploadSummary{Keys: make([]string, 0)}

	error := filepath.WalkDir(folderPath, func(path string, entry os.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}
		if entry.IsDir() {
			return nil
		}

		relativePath, error := filepath.Rel(folderPath, path)
		if error != nil {
			return error
		}
		key := joinKey(prefix, filepath.ToSlash(relativePath))
		size, error := UploadFile(ctx, client, bucket, key, path)
		if error != nil {
			return error
		}

		summary.FileCount++
		summary.TotalBytes += size
		summary.Keys = append(summary.Keys, key)
		return nil
	})
	if error != nil {
		return UploadSummary{}, error
	}

	return summary, nil
}

// UploadJSON はJSONバイト列をアップロードする。
func UploadJSON(ctx context.Context, client *s3.Client, bucket string, key string, payload []byte) error {
	reader := bytes.NewReader(payload)
	_, error := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &key,
		Body:        reader,
		ContentType: stringPtr("application/json"),
	})
	return error
}

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

// joinKey はプレフィックスとファイル名をS3キーとして結合する。
func joinKey(prefix string, name string) string {
	trimmed := strings.Trim(prefix, "/")
	if trimmed == "" {
		return name
	}
	return fmt.Sprintf("%s/%s", trimmed, strings.Trim(name, "/"))
}

// stringPtr は文字列のポインタを返す。
func stringPtr(value string) *string {
	return &value
}
