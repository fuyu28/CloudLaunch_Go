// @fileoverview メモ内容のハッシュ計算を提供する。
package memo

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// CalculateContentHash はメモ内容のSHA256ハッシュを返す。
func CalculateContentHash(content string) string {
	trimmed := strings.TrimSpace(content)
	hash := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(hash[:])
}
