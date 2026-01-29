// @fileoverview サービス層の入力検証ヘルパーを提供する。
package services

import "strings"

func requireNonEmpty(value string, field string) (string, string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", field + "が空です", false
	}
	return trimmed, "", true
}
