// @fileoverview S3オブジェクトの一覧・削除・ダウンロードを提供する。
package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ObjectInfo はS3オブジェクト情報を表す。
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified int64
}

// ListObjects は指定プレフィックス配下のオブジェクトを取得する。
func ListObjects(ctx context.Context, client *s3.Client, bucket string, prefix string) ([]ObjectInfo, error) {
	objects := make([]ObjectInfo, 0)
	input := &s3.ListObjectsV2Input{
		Bucket: &bucket,
	}
	if prefix != "" {
		input.Prefix = stringPtr(prefix)
	}
	paginator := s3.NewListObjectsV2Paginator(client, input)

	for paginator.HasMorePages() {
		page, error := paginator.NextPage(ctx)
		if error != nil {
			return nil, error
		}
		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}
			lastModified := int64(0)
			if obj.LastModified != nil {
				lastModified = obj.LastModified.UnixMilli()
			}
			size := int64(0)
			if obj.Size != nil {
				size = *obj.Size
			}
			objects = append(objects, ObjectInfo{
				Key:          *obj.Key,
				Size:         size,
				LastModified: lastModified,
			})
		}
	}

	return objects, nil
}

// DeleteObjectsByPrefix は指定プレフィックス配下のオブジェクトを削除する。
func DeleteObjectsByPrefix(ctx context.Context, client *s3.Client, bucket string, prefix string) error {
	objects, error := ListObjects(ctx, client, bucket, prefix)
	if error != nil {
		return error
	}
	if len(objects) == 0 {
		return nil
	}

	const maxBatch = 1000
	for start := 0; start < len(objects); start += maxBatch {
		end := start + maxBatch
		if end > len(objects) {
			end = len(objects)
		}
		batch := make([]s3types.ObjectIdentifier, 0, end-start)
		for _, obj := range objects[start:end] {
			batch = append(batch, s3types.ObjectIdentifier{Key: &obj.Key})
		}
		_, error := client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: &bucket,
			Delete: &s3types.Delete{Objects: batch},
		})
		if error != nil {
			return error
		}
	}

	return nil
}

// DeleteObject は単一オブジェクトを削除する。
func DeleteObject(ctx context.Context, client *s3.Client, bucket string, key string) error {
	_, error := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	return error
}

// DownloadPrefix はプレフィックス配下のオブジェクトをダウンロードする。
func DownloadPrefix(ctx context.Context, client *s3.Client, bucket string, prefix string, destination string) error {
	objects, error := ListObjects(ctx, client, bucket, prefix)
	if error != nil {
		return error
	}
	for _, obj := range objects {
		relative := strings.TrimPrefix(obj.Key, prefix)
		relative = strings.TrimPrefix(relative, "/")
		targetPath := filepath.Join(destination, filepath.FromSlash(relative))
		if error := os.MkdirAll(filepath.Dir(targetPath), 0o700); error != nil {
			return error
		}
		response, error := client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &obj.Key,
		})
		if error != nil {
			return error
		}
		content, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		if error := os.WriteFile(targetPath, content, 0o600); error != nil {
			return error
		}
	}
	return nil
}

// DownloadObject は単一オブジェクトをダウンロードする。
func DownloadObject(ctx context.Context, client *s3.Client, bucket string, key string) (data []byte, err error) {
	response, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	return io.ReadAll(response.Body)
}
