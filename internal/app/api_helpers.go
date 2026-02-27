// @fileoverview API用の共通ヘルパーを提供する。
package app

import "CloudLaunch_Go/internal/result"

func errorResultWithLog[T any](app *App, message string, err error, attrs ...any) result.ApiResult[T] {
	if app != nil && app.Logger != nil {
		logAttrs := make([]any, 0, len(attrs)+2)
		logAttrs = append(logAttrs, "error", err)
		logAttrs = append(logAttrs, attrs...)
		app.Logger.Error(message, logAttrs...)
	}
	return result.ErrorResult[T](message, err.Error())
}
