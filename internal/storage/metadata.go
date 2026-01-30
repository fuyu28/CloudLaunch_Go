// @fileoverview クラウドのメタ情報(JSON)の読み書きを提供する。
package storage

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// CloudGameMetadata はクラウドに保存するゲーム情報を表す。
type CloudGameMetadata struct {
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	Publisher      string     `json:"publisher"`
	ImageKey       *string    `json:"imageKey,omitempty"`
	PlayStatus     string     `json:"playStatus"`
	TotalPlayTime  int64      `json:"totalPlayTime"`
	LastPlayed     *time.Time `json:"lastPlayed,omitempty"`
	ClearedAt      *time.Time `json:"clearedAt,omitempty"`
	CurrentChapter *string    `json:"currentChapter,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// CloudMetadata は全ゲーム情報のまとめを表す。
type CloudMetadata struct {
	Version   int                 `json:"version"`
	UpdatedAt time.Time           `json:"updatedAt"`
	Games     []CloudGameMetadata `json:"games"`
}

// LoadMetadata はクラウド上のメタ情報を取得する。
func LoadMetadata(ctx context.Context, client *s3.Client, bucket string, key string) (metadata *CloudMetadata, err error) {
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

	var parsed CloudMetadata
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

// SaveMetadata はメタ情報をクラウドに保存する。
func SaveMetadata(ctx context.Context, client *s3.Client, bucket string, key string, metadata CloudMetadata) error {
	payload, error := json.Marshal(metadata)
	if error != nil {
		return error
	}
	return UploadJSON(ctx, client, bucket, key, payload)
}
