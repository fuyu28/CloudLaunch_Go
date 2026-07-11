// クラウド上のメモパスを生成・解析する。
package memo

import (
	"regexp"
	"strings"
)

// memoID は DB 側で hex(randomblob) 生成のため下線を含まない。タイトルの下線と
// 衝突しても最後の `_` 区切りを ID とみなせるので、この前提を崩す ID 生成に変えない。
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
func BuildMemoPath(gameID string, memoTitle string, memoID string) string {
	return "games/" + strings.TrimSpace(gameID) + "/memo/" + SanitizeForCloudPath(memoTitle) + "_" + memoID + ".md"
}

// BuildMemoPrefix はメモのプレフィックスを生成する。
func BuildMemoPrefix(gameID string) string {
	if strings.TrimSpace(gameID) == "" {
		return "games/"
	}
	return "games/" + strings.TrimSpace(gameID) + "/memo/"
}

// IsMemoPath はメモパスかどうか判定する。
func IsMemoPath(path string) bool {
	return strings.Contains(path, "/memo/") && strings.HasSuffix(path, ".md")
}

// ExtractMemoInfo はメモパスから情報を抽出する。
func ExtractMemoInfo(path string) (gameID string, memoTitle string, memoID string, ok bool) {
	matches := memoPathRegex.FindStringSubmatch(path)
	if len(matches) != 4 {
		return "", "", "", false
	}
	return matches[1], matches[2], matches[3], true
}
