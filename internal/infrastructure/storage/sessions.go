// @fileoverview クラウドのセッション情報(JSON)の読み書きを提供する。
package storage

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// CloudSessionRecord はクラウドに保存するセッション情報を表す。
type CloudSessionRecord struct {
	ID          string    `json:"id"`
	PlayedAt    time.Time `json:"playedAt"`
	Duration    int64     `json:"duration"`
	SessionName *string   `json:"sessionName,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// LoadSessions はクラウド上のセッション情報を取得する。
func LoadSessions(ctx context.Context, client *s3.Client, bucket string, key string) (sessions []CloudSessionRecord, err error) {
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

	var parsed []CloudSessionRecord
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

// SaveSessions はセッション情報をクラウドに保存する。
func SaveSessions(ctx context.Context, client *s3.Client, bucket string, key string, sessions []CloudSessionRecord) error {
	payload, err := json.Marshal(sessions)
	if err != nil {
		return err
	}
	return UploadJSON(ctx, client, bucket, key, payload)
}
