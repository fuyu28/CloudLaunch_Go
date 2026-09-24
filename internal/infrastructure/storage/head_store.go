// ゲームごとのリモートHEADの読み書きを提供する。
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

func headKeyV2(gameID string) string {
	return fmt.Sprintf("games/%s/HEAD.v2", gameID)
}

// WriteHEAD はレガシー v1 HEAD を書き込む（新クライアントは使わない。テスト／移行用）。
func WriteHEAD(ctx context.Context, client *s3.Client, bucket, gameID, hash string) error {
	return UploadBytes(ctx, client, bucket, headKey(gameID), []byte(hash), "text/plain")
}

// WriteHEADv2 はプロトコル v2 のリモート HEAD を書き込む。レガシー HEAD は上書きしない。
func WriteHEADv2(ctx context.Context, client *s3.Client, bucket, gameID, hash string) error {
	return UploadBytes(ctx, client, bucket, headKeyV2(gameID), []byte(hash), "text/plain")
}

// ReadHEAD はレガシー v1 HEAD を取得する。存在しない場合は "" を返す。
func ReadHEAD(ctx context.Context, client *s3.Client, bucket, gameID string) (string, error) {
	return readHeadObject(ctx, client, bucket, headKey(gameID))
}

// ReadHEADv2 はプロトコル v2 HEAD を取得する。存在しない場合は "" を返す。
func ReadHEADv2(ctx context.Context, client *s3.Client, bucket, gameID string) (string, error) {
	return readHeadObject(ctx, client, bucket, headKeyV2(gameID))
}

// ReadPreferredHEAD は HEAD.v2 を優先し、無ければレガシー HEAD を返す。
// usedV2 は実際に v2 キーから読んだときに true。
func ReadPreferredHEAD(ctx context.Context, client *s3.Client, bucket, gameID string) (hash string, usedV2 bool, err error) {
	v2, err := ReadHEADv2(ctx, client, bucket, gameID)
	if err != nil {
		return "", false, err
	}
	if v2 != "" {
		return v2, true, nil
	}
	v1, err := ReadHEAD(ctx, client, bucket, gameID)
	if err != nil {
		return "", false, err
	}
	return v1, false, nil
}

func readHeadObject(ctx context.Context, client *s3.Client, bucket, key string) (string, error) {
	data, err := DownloadObject(ctx, client, bucket, key)
	if err != nil {
		if IsNotFoundError(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
