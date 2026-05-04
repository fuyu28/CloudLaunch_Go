// @fileoverview クラウドのプレイルート情報(JSON)の読み書きを提供する。
package storage

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// CloudPlayRouteRecord はクラウドに保存するプレイルート情報を表す。
type CloudPlayRouteRecord struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sortOrder"`
	CreatedAt time.Time `json:"createdAt"`
}

// LoadPlayRoutes はクラウド上のプレイルート情報を取得する。
func LoadPlayRoutes(ctx context.Context, client *s3.Client, bucket string, key string) (routes []CloudPlayRouteRecord, err error) {
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

	var parsed []CloudPlayRouteRecord
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

// SavePlayRoutes はプレイルート情報をクラウドに保存する。
func SavePlayRoutes(ctx context.Context, client *s3.Client, bucket string, key string, routes []CloudPlayRouteRecord) error {
	payload, err := json.Marshal(routes)
	if err != nil {
		return err
	}
	return UploadJSON(ctx, client, bucket, key, payload)
}
