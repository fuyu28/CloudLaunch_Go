// @fileoverview クラウド上のメモパスを生成・解析する。
package memo

import (
	"regexp"
	"strings"
)

var memoPathRegex = regexp.MustCompile(`^games/([^/]+)/memo/(.+)_([^_]+)\.md$`)

// SanitizeForCloudPath はクラウド用に文字列を整形する。
func SanitizeForCloudPath(name string) string {
	sanitized := strings.ReplaceAll(name, " ", "_")
	sanitized = strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
			return '_'
		default:
			return r
		}
	}, sanitized)
	for strings.Contains(sanitized, "__") {
		sanitized = strings.ReplaceAll(sanitized, "__", "_")
	}
	sanitized = strings.Trim(sanitized, "_")
	if len(sanitized) > 100 {
		sanitized = sanitized[:100]
	}
	return sanitized
}

// BuildMemoPath はメモのクラウドパスを生成する。
func BuildMemoPath(gameTitle string, memoTitle string, memoID string) string {
	return "games/" + SanitizeForCloudPath(gameTitle) + "/memo/" + SanitizeForCloudPath(memoTitle) + "_" + memoID + ".md"
}

// BuildMemoPrefix はメモのプレフィックスを生成する。
func BuildMemoPrefix(gameTitle string) string {
	if strings.TrimSpace(gameTitle) == "" {
		return "games/"
	}
	return "games/" + SanitizeForCloudPath(gameTitle) + "/memo/"
}

// IsMemoPath はメモパスかどうか判定する。
func IsMemoPath(path string) bool {
	return strings.Contains(path, "/memo/") && strings.HasSuffix(path, ".md")
}

// ExtractMemoInfo はメモパスから情報を抽出する。
func ExtractMemoInfo(path string) (gameTitle string, memoTitle string, memoID string, ok bool) {
	matches := memoPathRegex.FindStringSubmatch(path)
	if len(matches) != 4 {
		return "", "", "", false
	}
	return matches[1], matches[2], matches[3], true
}
