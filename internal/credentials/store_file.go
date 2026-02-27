//go:build !windows

// @fileoverview 非Windows環境向けの簡易ファイルストア。
package credentials

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var credentialKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

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
	validatedKey, err := validateCredentialKey(key)
	if err != nil {
		return err
	}
	if error := os.MkdirAll(store.Directory, 0o700); error != nil {
		return error
	}

	path := store.filePath(validatedKey)
	blob, error := json.Marshal(credential)
	if error != nil {
		return error
	}

	return os.WriteFile(path, blob, 0o600)
}

// Load は認証情報を取得する。
func (store *FileStore) Load(_ context.Context, key string) (*Credential, error) {
	validatedKey, err := validateCredentialKey(key)
	if err != nil {
		return nil, err
	}
	path := store.filePath(validatedKey)
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
	validatedKey, err := validateCredentialKey(key)
	if err != nil {
		return err
	}
	path := store.filePath(validatedKey)
	if error := os.Remove(path); error != nil && !os.IsNotExist(error) {
		return error
	}
	return nil
}

func (store *FileStore) filePath(key string) string {
	return filepath.Join(store.Directory, key+".json")
}

func validateCredentialKey(key string) (string, error) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return "", errors.New("key is empty")
	}
	if !credentialKeyPattern.MatchString(trimmed) {
		return "", errors.New("key contains invalid characters")
	}
	return trimmed, nil
}
