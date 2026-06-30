// API用の共通ヘルパーを提供する。
package app

import (
	"errors"
	"strings"

	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/services"
)

func errorResultWithLog[T any](app *App, message string, err error, attrs ...any) result.ApiResult[T] {
	if app != nil && app.Logger != nil {
		logAttrs := make([]any, 0, len(attrs)+2)
		logAttrs = append(logAttrs, "error", err)
		logAttrs = append(logAttrs, attrs...)
		app.Logger.Error(message, logAttrs...)
	}
	return result.ErrorResult[T](message, err.Error())
}

func serviceErrorResult[T any](err error, fallbackMessage string) result.ApiResult[T] {
	if err == nil {
		return result.ErrorResult[T](fallbackMessage, "不明なエラーです")
	}
	serviceErr := &services.ServiceError{}
	if errors.As(err, &serviceErr) {
		message := serviceErr.Message
		if strings.TrimSpace(message) == "" {
			message = fallbackMessage
		}
		return result.ErrorResult[T](message, serviceErr.Detail)
	}
	return result.ErrorResult[T](fallbackMessage, err.Error())
}

func serviceResult[T any](data T, err error, fallbackMessage string) result.ApiResult[T] {
	if err != nil {
		return serviceErrorResult[T](err, fallbackMessage)
	}
	return result.OkResult(data)
}

// boolResult はエラーがあれば ServiceError を解いて ErrorResult を、無ければ OkResult(true) を返す。
// 「サービスを呼んでエラーなら ErrorResult、成功なら true」だけの bool 系 API メソッドの定型を集約する。
func boolResult(err error, fallbackMessage string) result.ApiResult[bool] {
	if err != nil {
		return serviceErrorResult[bool](err, fallbackMessage)
	}
	return result.OkResult(true)
}

// requireGameID は API 入力の gameID をトリムし、空なら標準形式の ErrorResult を返す。
// ok=true のとき trimmed が有効値、ok=false のとき errResult を return すればよい。
func requireGameID[T any](gameID string) (trimmed string, errResult result.ApiResult[T], ok bool) {
	trimmed = strings.TrimSpace(gameID)
	if trimmed == "" {
		return "", result.ErrorResult[T]("ゲームIDが不正です", "gameID is empty"), false
	}
	return trimmed, result.ApiResult[T]{}, true
}
