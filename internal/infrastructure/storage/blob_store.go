// @fileoverview コンテンツアドレッシングブロブの読み書きを提供する。
package storage

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func blobKey(gameID, hash string) string {
	return fmt.Sprintf("games/%s/objects/%s", gameID, hash)
}

// BlobExists はブロブがS3に存在するかを確認する。
func BlobExists(ctx context.Context, client *s3.Client, bucket, gameID, hash string) (bool, error) {
	key := blobKey(gameID, hash)
	_, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		if IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// PutBlob はブロブをS3にアップロードする。既に存在する場合はスキップする。
func PutBlob(ctx context.Context, client *s3.Client, bucket, gameID, hash string, data []byte) error {
	exists, err := BlobExists(ctx, client, bucket, gameID, hash)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	key := blobKey(gameID, hash)
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   bytes.NewReader(data),
	})
	return err
}

// GetBlob はS3からブロブを取得する。
func GetBlob(ctx context.Context, client *s3.Client, bucket, gameID, hash string) ([]byte, error) {
	return DownloadObject(ctx, client, bucket, blobKey(gameID, hash))
}

// ListBlobHashes はゲームの既存ブロブハッシュを一括取得する。
func ListBlobHashes(ctx context.Context, client *s3.Client, bucket, gameID string) (map[string]struct{}, error) {
	prefix := fmt.Sprintf("games/%s/objects/", gameID)
	existing := make(map[string]struct{})
	input := &s3.ListObjectsV2Input{
		Bucket: &bucket,
		Prefix: &prefix,
	}
	paginator := s3.NewListObjectsV2Paginator(client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}
			existing[path.Base(*obj.Key)] = struct{}{}
		}
	}
	return existing, nil
}

// PutBlobs はブロブを一括アップロードする。
// ListObjectsV2 でリモートの既存ハッシュを一括取得し、不足分のみ並列アップロードする。
// onProgress は (アップロード済み件数, 総件数) を受け取るコールバック。nil 可。
func PutBlobs(
	ctx context.Context,
	client *s3.Client,
	bucket, gameID string,
	blobs map[string][]byte,
	concurrency int,
	onProgress func(uploaded, total int),
) error {
	total := len(blobs)
	if total == 0 {
		return nil
	}

	existing, err := ListBlobHashes(ctx, client, bucket, gameID)
	if err != nil {
		return err
	}

	type task struct {
		hash string
		data []byte
	}
	tasks := make([]task, 0, total)
	for hash, data := range blobs {
		if _, ok := existing[hash]; !ok {
			tasks = append(tasks, task{hash: hash, data: data})
		}
	}

	alreadyDone := total - len(tasks)
	if onProgress != nil {
		onProgress(alreadyDone, total)
	}
	if len(tasks) == 0 {
		return nil
	}

	if concurrency <= 0 {
		concurrency = defaultUploadConcurrency
	}
	workerCount := concurrency
	if workerCount > len(tasks) {
		workerCount = len(tasks)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	taskCh := make(chan task, len(tasks))
	for _, t := range tasks {
		taskCh <- t
	}
	close(taskCh)

	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error
	var mu sync.Mutex
	uploaded := alreadyDone

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range taskCh {
				if ctx.Err() != nil {
					return
				}
				key := blobKey(gameID, t.hash)
				_, putErr := client.PutObject(ctx, &s3.PutObjectInput{
					Bucket: &bucket,
					Key:    &key,
					Body:   bytes.NewReader(t.data),
				})
				if putErr != nil {
					errOnce.Do(func() {
						firstErr = putErr
						cancel()
					})
					return
				}
				if onProgress != nil {
					mu.Lock()
					uploaded++
					onProgress(uploaded, total)
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
	return firstErr
}
