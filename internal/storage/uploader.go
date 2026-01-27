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
func UploadFile(ctx context.Context, client *s3.Client, bucket string, key string, filePath string) (int64, error) {
	file, error := os.Open(filePath)
	if error != nil {
		return 0, error
	}
	defer file.Close()

	info, error := file.Stat()
	if error != nil {
		return 0, error
	}

	_, error = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   file,
	})
	if error != nil {
		return 0, error
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
