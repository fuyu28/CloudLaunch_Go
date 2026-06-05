// @fileoverview S3オブジェクトの一覧・削除・ダウンロードを提供する。
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	objects, err := ListObjects(ctx, client, bucket, prefix)
	if err != nil {
		return err
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

const defaultDownloadRetryCount = 3

// DownloadPrefix はプレフィックス配下のオブジェクトをダウンロードする。
func DownloadPrefix(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	prefix string,
	destination string,
	concurrency int,
	retryCount int,
) error {
	objects, err := ListObjects(ctx, client, bucket, prefix)
	if err != nil {
		return err
	}
	if len(objects) == 0 {
		return nil
	}

	if concurrency <= 0 {
		concurrency = defaultUploadConcurrency
	}
	if retryCount < 0 {
		retryCount = defaultDownloadRetryCount
	}

	type downloadTask struct {
		key        string
		targetPath string
	}
	tasks := make([]downloadTask, 0, len(objects))
	baseDestination, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(baseDestination, 0o700); err != nil {
		return err
	}
	for _, obj := range objects {
		targetPath, err := resolveSafeDownloadPath(baseDestination, obj.Key, prefix)
		if err != nil {
			return err
		}
		if error := os.MkdirAll(filepath.Dir(targetPath), 0o700); error != nil {
			return error
		}
		tasks = append(tasks, downloadTask{key: obj.Key, targetPath: targetPath})
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	workerCount := concurrency
	if workerCount > len(tasks) {
		workerCount = len(tasks)
	}
	taskCh := make(chan downloadTask, len(tasks))
	for _, task := range tasks {
		taskCh <- task
	}
	close(taskCh)

	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskCh {
				if ctx.Err() != nil {
					return
				}
				if err := downloadObjectWithRetry(ctx, client, bucket, task.key, task.targetPath, retryCount); err != nil {
					errOnce.Do(func() {
						firstErr = err
						cancel()
					})
					return
				}
			}
		}()
	}

	wg.Wait()
	if firstErr != nil {
		return firstErr
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

func downloadObjectWithRetry(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	key string,
	targetPath string,
	retryCount int,
) error {
	var lastErr error
	for attempt := 0; attempt <= retryCount; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if attempt > 0 {
			backoff := time.Duration(attempt) * 250 * time.Millisecond
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}

		if err := downloadObjectToPath(ctx, client, bucket, key, targetPath); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr == nil {
		return errors.New("download retry exceeded")
	}
	return lastErr
}

func downloadObjectToPath(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	key string,
	targetPath string,
) error {
	response, error := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if error != nil {
		return error
	}
	defer func() {
		_ = response.Body.Close()
	}()
	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, response.Body); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if error := os.Chmod(targetPath, 0o600); error != nil {
		return error
	}
	return nil
}

func resolveSafeDownloadPath(destination string, key string, prefix string) (string, error) {
	relative := strings.TrimPrefix(key, prefix)
	relative = strings.TrimPrefix(relative, "/")
	relative = strings.TrimSpace(relative)
	if relative == "" {
		return "", fmt.Errorf("object key is invalid: %s", key)
	}

	cleaned := path.Clean(relative)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("object key escapes destination: %s", key)
	}

	targetPath := filepath.Join(destination, filepath.FromSlash(cleaned))
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return "", err
	}

	relToBase, err := filepath.Rel(destination, absTarget)
	if err != nil {
		return "", err
	}
	if relToBase == ".." || strings.HasPrefix(relToBase, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("object key escapes destination: %s", key)
	}
	return absTarget, nil
}
