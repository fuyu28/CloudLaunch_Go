// @fileoverview API用の共通ヘルパーを提供する。
package app

import "CloudLaunch_Go/internal/result"

func errorResult[T any](message string, err error) result.ApiResult[T] {
	return result.ErrorResult[T](message, err.Error())
}
