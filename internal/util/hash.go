package util

import (
	"crypto/sha256"
	"encoding/hex"
)

// Sha256Hex は入力のSHA256をhex文字列で返す。
// services 層の content fingerprinting と infrastructure 層のブロブ検証で共有する。
func Sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
