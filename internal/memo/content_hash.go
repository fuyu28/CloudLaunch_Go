// メモ内容のハッシュ計算を提供する。
package memo

import (
	"strings"

	"CloudLaunch_Go/internal/util"
)

// CalculateContentHash はメモ内容のSHA256ハッシュを返す。
// 前後の空白は同期判定で同一視するため、ハッシュ前に TrimSpace する。
func CalculateContentHash(content string) string {
	return util.Sha256Hex([]byte(strings.TrimSpace(content)))
}
