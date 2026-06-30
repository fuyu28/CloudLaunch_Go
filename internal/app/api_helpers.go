// @fileoverview API用の共通ヘルパーを提供する。
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
