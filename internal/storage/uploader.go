// @fileoverview S3 へのファイル/フォルダアップロードを提供する。
package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// UploadSummary はアップロード結果の要約を表す。
type UploadSummary struct {
	FileCount  int
	TotalBytes int64
	Keys       []string
}

const (
	defaultUploadConcurrency = 6
	defaultUploadRetryCount  = 3
)

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
func UploadFolder(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	folderPath string,
	prefix string,
	concurrency int,
	retryCount int,
) (UploadSummary, error) {
	type uploadTask struct {
		path string
		key  string
	}

	tasks := make([]uploadTask, 0)
	walkErr := filepath.WalkDir(folderPath, func(path string, entry os.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}
		if entry.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(folderPath, path)
		if err != nil {
			return err
		}
		key := joinKey(prefix, filepath.ToSlash(relativePath))
		tasks = append(tasks, uploadTask{path: path, key: key})
		return nil
	})
	if walkErr != nil {
		return UploadSummary{}, walkErr
	}

	if len(tasks) == 0 {
		return UploadSummary{Keys: make([]string, 0)}, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if concurrency <= 0 {
		concurrency = defaultUploadConcurrency
	}
	if retryCount < 0 {
		retryCount = defaultUploadRetryCount
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error
	var summaryMu sync.Mutex
	summary := UploadSummary{Keys: make([]string, 0, len(tasks))}

	for _, task := range tasks {
		wg.Add(1)
		task := task
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			size, err := uploadFileWithRetry(ctx, client, bucket, task.key, task.path, retryCount)
			if err != nil {
				errOnce.Do(func() {
					firstErr = err
					cancel()
				})
				return
			}

			summaryMu.Lock()
			summary.FileCount++
			summary.TotalBytes += size
			summary.Keys = append(summary.Keys, task.key)
			summaryMu.Unlock()
		}()
	}

	wg.Wait()
	if firstErr != nil {
		return UploadSummary{}, firstErr
	}
	return summary, nil
}

func uploadFileWithRetry(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	key string,
	filePath string,
	retryCount int,
) (int64, error) {
	var lastErr error
	for attempt := 0; attempt <= retryCount; attempt++ {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		if attempt > 0 {
			backoff := time.Duration(attempt) * 250 * time.Millisecond
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return 0, ctx.Err()
			case <-timer.C:
			}
		}

		size, err := UploadFile(ctx, client, bucket, key, filePath)
		if err != nil {
			lastErr = err
			continue
		}
		return size, nil
	}
	if lastErr == nil {
		return 0, errors.New("upload retry exceeded")
	}
	return 0, lastErr
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
