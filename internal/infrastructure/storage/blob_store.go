// コンテンツアドレッシングブロブの読み書きを提供する。
package storage

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"CloudLaunch_Go/internal/util"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// BlobKind はS3上のオブジェクト種別を表す。
type BlobKind = string

const (
	BlobKindCommit BlobKind = "commits" // MetaSnapshot（git の commit 相当）
	BlobKindTree   BlobKind = "trees"   // SaveSnapshot（git の tree 相当）
	BlobKindMeta   BlobKind = "meta"    // ゲーム情報・セッション情報 JSON
	BlobKindObject BlobKind = "objects" // セーブファイル実データ・画像
)

func blobKey(gameID, kind, hash string) string {
	return fmt.Sprintf("games/%s/%s/%s", gameID, kind, hash)
}

func contentTypeForKind(kind string) string {
	switch kind {
	case BlobKindCommit, BlobKindTree, BlobKindMeta:
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

func blobHashBytes(data []byte) string {
	return util.Sha256Hex(data)
}

// ResolveSafeRelativePath は baseDir 配下のスラッシュ区切り相対パスを解決する。
// ファイルシステム書き込み前に、空・絶対パス・親ディレクトリ参照を拒否する。
func ResolveSafeRelativePath(baseDir, relPath string) (string, error) {
	trimmed := strings.TrimSpace(relPath)
	if trimmed == "" {
		return "", fmt.Errorf("relative path is empty")
	}

	cleaned := path.Clean(trimmed)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || path.IsAbs(cleaned) {
		return "", fmt.Errorf("relative path escapes base directory: %s", relPath)
	}

	localRel := filepath.FromSlash(cleaned)
	if filepath.IsAbs(localRel) || filepath.VolumeName(localRel) != "" {
		return "", fmt.Errorf("relative path escapes base directory: %s", relPath)
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	targetPath := filepath.Join(absBase, localRel)
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return "", err
	}

	relToBase, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return "", err
	}
	if relToBase == ".." || strings.HasPrefix(relToBase, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("relative path escapes base directory: %s", relPath)
	}
	return absTarget, nil
}

func blobExists(ctx context.Context, client *s3.Client, bucket, gameID, kind, hash string) (bool, error) {
	key := blobKey(gameID, kind, hash)
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
func PutBlob(ctx context.Context, client *s3.Client, bucket, gameID, kind, hash string, data []byte) error {
	if blobHashBytes(data) != hash {
		return fmt.Errorf("blob hash mismatch: %s/%s", kind, hash)
	}
	exists, err := blobExists(ctx, client, bucket, gameID, kind, hash)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return UploadBytes(ctx, client, bucket, blobKey(gameID, kind, hash), data, contentTypeForKind(kind))
}

// GetBlob はS3からブロブを取得する。
func GetBlob(ctx context.Context, client *s3.Client, bucket, gameID, kind, hash string) ([]byte, error) {
	data, err := DownloadObject(ctx, client, bucket, blobKey(gameID, kind, hash))
	if err != nil {
		return nil, err
	}
	if blobHashBytes(data) != hash {
		return nil, fmt.Errorf("blob hash mismatch: %s/%s", kind, hash)
	}
	return data, nil
}

// ListBlobHashes はゲームの既存セーブファイルブロブのハッシュを一括取得する。
// objects/ プレフィックスのみを対象とする。
func ListBlobHashes(ctx context.Context, client *s3.Client, bucket, gameID string) (map[string]struct{}, error) {
	prefix := fmt.Sprintf("games/%s/%s/", gameID, BlobKindObject)
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

// PutBlobs はセーブファイルブロブを一括アップロードする（objects/ 固定）。
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
		if blobHashBytes(data) != hash {
			return fmt.Errorf("blob hash mismatch: %s/%s", BlobKindObject, hash)
		}
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
				putErr := UploadBytes(ctx, client, bucket, blobKey(gameID, BlobKindObject, t.hash), t.data, contentTypeForKind(BlobKindObject))
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

// DownloadBlobs はセーブファイルブロブを並列ダウンロードしてローカルに保存する（objects/ 固定）。
// blobs は relPath → hash のマップ。saveDir 配下の relPath に書き込む。
// onProgress は (ダウンロード済み件数, 総件数) を受け取るコールバック。nil 可。
func DownloadBlobs(
	ctx context.Context,
	client *s3.Client,
	bucket, gameID, saveDir string,
	blobs map[string]string,
	concurrency int,
	onProgress func(downloaded, total int),
) error {
	if len(blobs) == 0 {
		return nil
	}

	if concurrency <= 0 {
		concurrency = defaultUploadConcurrency
	}

	type task struct {
		relPath string
		hash    string
	}
	tasks := make([]task, 0, len(blobs))
	for relPath, hash := range blobs {
		tasks = append(tasks, task{relPath: relPath, hash: hash})
	}

	total := len(tasks)
	workerCount := concurrency
	if workerCount > total {
		workerCount = total
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	taskCh := make(chan task, total)
	for _, t := range tasks {
		taskCh <- t
	}
	close(taskCh)

	var wg sync.WaitGroup
	var errOnce sync.Once
	var firstErr error
	var mu sync.Mutex
	downloaded := 0

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range taskCh {
				if ctx.Err() != nil {
					return
				}
				data, err := GetBlob(ctx, client, bucket, gameID, BlobKindObject, t.hash)
				if err != nil {
					errOnce.Do(func() { firstErr = err; cancel() })
					return
				}
				targetPath, err := ResolveSafeRelativePath(saveDir, t.relPath)
				if err != nil {
					errOnce.Do(func() { firstErr = err; cancel() })
					return
				}
				if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
					errOnce.Do(func() { firstErr = err; cancel() })
					return
				}
				if err := os.WriteFile(targetPath, data, 0o600); err != nil {
					errOnce.Do(func() { firstErr = err; cancel() })
					return
				}
				if onProgress != nil {
					mu.Lock()
					downloaded++
					onProgress(downloaded, total)
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
	return firstErr
}
