// @fileoverview ゲームごとのリモートHEADの読み書きを提供する。
package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func headKey(gameID string) string {
	return fmt.Sprintf("games/%s/HEAD", gameID)
}

// WriteHEAD はリモートHEADをS3に書き込む。
func WriteHEAD(ctx context.Context, client *s3.Client, bucket, gameID, hash string) error {
	return UploadBytes(ctx, client, bucket, headKey(gameID), []byte(hash), "text/plain")
}

// ReadHEAD はS3からリモートHEADを取得する。存在しない場合は "" を返す。
func ReadHEAD(ctx context.Context, client *s3.Client, bucket, gameID string) (string, error) {
	data, err := DownloadObject(ctx, client, bucket, headKey(gameID))
	if err != nil {
		if IsNotFoundError(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
