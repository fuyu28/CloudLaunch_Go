// @fileoverview API結果型とエラー型を提供する。
package result

import "time"

// ApiError は API で返すエラー情報を表す。
type ApiError struct {
	Message string    `json:"message"`
	Detail  string    `json:"detail"`
	At      time.Time `json:"at"`
}

// ApiResult は API レスポンスの共通形を表す。
type ApiResult[T any] struct {
	Success bool      `json:"success"`
	Data    T         `json:"data,omitempty"`
	Error   *ApiError `json:"error,omitempty"`
}

// OkResult は成功時の ApiResult を生成する。
func OkResult[T any](data T) ApiResult[T] {
	return ApiResult[T]{
		Success: true,
		Data:    data,
	}
}

// ErrorResult は失敗時の ApiResult を生成する。
func ErrorResult[T any](message string, detail string) ApiResult[T] {
	return ApiResult[T]{
		Success: false,
		Error: &ApiError{
			Message: message,
			Detail:  detail,
			At:      time.Now(),
		},
	}
}
