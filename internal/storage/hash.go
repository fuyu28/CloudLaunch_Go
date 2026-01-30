// @fileoverview ディレクトリのハッシュ計算を提供する。
package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// HashDirectory はディレクトリ配下の全ファイルからハッシュを生成する。
func HashDirectory(root string) (string, error) {
	root = strings.TrimSpace(root)
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", os.ErrInvalid
	}

	paths := make([]string, 0)
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, rel)
		return nil
	}); err != nil {
		return "", err
	}

	sort.Strings(paths)
	hasher := sha256.New()
	for _, rel := range paths {
		normalized := filepath.ToSlash(rel)
		_, _ = hasher.Write([]byte(normalized))
		_, _ = hasher.Write([]byte{0})
		fileHash, err := hashFile(filepath.Join(root, rel))
		if err != nil {
			return "", err
		}
		_, _ = hasher.Write(fileHash)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func hashFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, err
	}
	return hasher.Sum(nil), nil
}
