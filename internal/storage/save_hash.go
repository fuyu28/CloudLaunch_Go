// @fileoverview セーブデータハッシュ(JSON)の読み書きを提供する。
package storage

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// SaveHashMetadata はセーブデータハッシュのメタ情報を表す。
type SaveHashMetadata struct {
	Hash      string    `json:"hash"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// LoadSaveHash はクラウド上のセーブデータハッシュを取得する。
func LoadSaveHash(ctx context.Context, client *s3.Client, bucket string, key string) (metadata *SaveHashMetadata, err error) {
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

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var parsed SaveHashMetadata
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

// SaveSaveHash はセーブデータハッシュをクラウドに保存する。
func SaveSaveHash(ctx context.Context, client *s3.Client, bucket string, key string, metadata SaveHashMetadata) error {
	payload, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	return UploadJSON(ctx, client, bucket, key, payload)
}
