//go:build windows

// Windows Credential Manager を使う認証情報ストア。
package credentials

import (
	"context"
	"encoding/json"

	"github.com/danieljoos/wincred"
)

// WindowsStore は Credential Manager を利用する実装。
type WindowsStore struct {
	Namespace string
}

func NewWindowsStore(namespace string) *WindowsStore {
	return &WindowsStore{Namespace: namespace}
}

func (store *WindowsStore) Save(_ context.Context, key string, credential Credential) error {
	blob, error := json.Marshal(credential)
	if error != nil {
		return error
	}

	cred := wincred.NewGenericCredential(store.qualifiedKey(key))
	cred.CredentialBlob = blob
	cred.Persist = wincred.PersistLocalMachine
	return cred.Write()
}

// Load は未登録キーをエラーにせず (nil, nil) を返す（呼び出し側で「未設定」と区別できるようにする）。
func (store *WindowsStore) Load(_ context.Context, key string) (*Credential, error) {
	cred, error := wincred.GetGenericCredential(store.qualifiedKey(key))
	if error != nil {
		if error == wincred.ErrElementNotFound {
			return nil, nil
		}
		return nil, error
	}

	var data Credential
	if error := json.Unmarshal(cred.CredentialBlob, &data); error != nil {
		return nil, error
	}
	return &data, nil
}

// Delete は未登録キーもエラーにしない（べき等な削除として扱う）。
func (store *WindowsStore) Delete(_ context.Context, key string) error {
	cred, error := wincred.GetGenericCredential(store.qualifiedKey(key))
	if error != nil {
		if error == wincred.ErrElementNotFound {
			return nil
		}
		return error
	}
	return cred.Delete()
}

func (store *WindowsStore) qualifiedKey(key string) string {
	return store.Namespace + ":" + key
}
