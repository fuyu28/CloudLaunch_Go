//go:build !windows

// @fileoverview 非Windows環境向けの簡易ファイルストア。
package credentials

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
)

// FileStore はローカルファイルに認証情報を保存する。
// Note: 本番では Windows Credential Manager を利用する想定。
type FileStore struct {
	Directory string
}

// NewFileStore は FileStore を生成する。
func NewFileStore(directory string) *FileStore {
	return &FileStore{Directory: directory}
}

// Save は認証情報を保存する。
func (store *FileStore) Save(_ context.Context, key string, credential Credential) error {
	if error := os.MkdirAll(store.Directory, 0o700); error != nil {
		return error
	}

	path := store.filePath(key)
	blob, error := json.Marshal(credential)
	if error != nil {
		return error
	}

	return os.WriteFile(path, blob, 0o600)
}

// Load は認証情報を取得する。
func (store *FileStore) Load(_ context.Context, key string) (*Credential, error) {
	path := store.filePath(key)
	blob, error := os.ReadFile(path)
	if error != nil {
		if os.IsNotExist(error) {
			return nil, nil
		}
		return nil, error
	}

	var data Credential
	if error := json.Unmarshal(blob, &data); error != nil {
		return nil, error
	}
	return &data, nil
}

// Delete は認証情報を削除する。
func (store *FileStore) Delete(_ context.Context, key string) error {
	path := store.filePath(key)
	if error := os.Remove(path); error != nil && !os.IsNotExist(error) {
		return error
	}
	return nil
}

func (store *FileStore) filePath(key string) string {
	return filepath.Join(store.Directory, key+".json")
}
