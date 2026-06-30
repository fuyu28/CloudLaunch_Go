// S3オブジェクトの一覧・削除・ダウンロードを提供する。
package storage

import (
	"context"
	"fmt"
	"io"
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
		output, error := client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: &bucket,
			Delete: &s3types.Delete{Objects: batch},
		})
		if error != nil {
			return error
		}
		// DeleteObjects は HTTP 200 でも個別キーの失敗（AccessDenied / MFA delete /
		// バージョニング有効バケットでの NoSuchKey 等）が output.Errors に入る。
		// これを無視するとクラウド削除を「成功」と扱って LocalSyncHead をクリアしつつ
		// 孤児 blob がバケットに残ってしまうので、集約してエラー化する。
		if perKeyErr := deleteErrorsToError(output.Errors); perKeyErr != nil {
			return perKeyErr
		}
	}

	return nil
}

// deleteErrorsToError は DeleteObjects の per-key 失敗一覧を集約エラーへ変換する。
func deleteErrorsToError(errs []s3types.Error) error {
	if len(errs) == 0 {
		return nil
	}
	const sampleLimit = 5
	parts := make([]string, 0, sampleLimit+1)
	for i, e := range errs {
		if i >= sampleLimit {
			parts = append(parts, fmt.Sprintf("... (+%d more)", len(errs)-sampleLimit))
			break
		}
		key, code, msg := "?", "?", "?"
		if e.Key != nil {
			key = *e.Key
		}
		if e.Code != nil {
			code = *e.Code
		}
		if e.Message != nil {
			msg = *e.Message
		}
		parts = append(parts, fmt.Sprintf("%s: %s (%s)", key, code, msg))
	}
	return fmt.Errorf("DeleteObjects に %d 件の個別失敗: %s", len(errs), strings.Join(parts, "; "))
}

// DeleteObject は単一オブジェクトを削除する。
func DeleteObject(ctx context.Context, client *s3.Client, bucket string, key string) error {
	_, error := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	return error
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
