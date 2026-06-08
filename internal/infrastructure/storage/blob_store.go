// @fileoverview コンテンツアドレッシングブロブの読み書きを提供する。
package storage

import (
	"bytes"
	"context"
	"fmt"

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
