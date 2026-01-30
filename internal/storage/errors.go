// @fileoverview クラウドストレージエラー判定を提供する。
package storage

import (
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// IsNotFoundError はS3のNotFound系エラーかどうかを判定する。
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		if code == "NoSuchKey" || code == "NotFound" {
			return true
		}
	}
	var noSuchKey *types.NoSuchKey
	return errors.As(err, &noSuchKey)
}
