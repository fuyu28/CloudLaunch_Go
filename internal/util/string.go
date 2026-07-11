// 文字列の共通ヘルパ（FirstNonEmpty など）を提供する。
package util

import "strings"

// FirstNonEmpty はスペースをトリムした後、最初の空でない文字列を返す。
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
