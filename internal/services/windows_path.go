// Windows パス区切りの正規化ヘルパを提供する。
package services

import (
	"path"
	"strings"
)

func normalizeWindowsPathSeparators(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return strings.ReplaceAll(trimmed, "\\", "/")
}

func windowsPathBase(value string) string {
	normalized := normalizeWindowsPathSeparators(value)
	if normalized == "" {
		return ""
	}
	return path.Base(normalized)
}

func windowsPathDir(value string) string {
	normalized := normalizeWindowsPathSeparators(value)
	if normalized == "" {
		return ""
	}
	return path.Dir(normalized)
}
