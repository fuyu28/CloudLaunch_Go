//go:build !windows

package credentials

import (
	"context"
	"errors"
)

type unsupportedStore struct {
	namespace string
}

func NewUnsupportedStore(namespace string) Store {
	return &unsupportedStore{namespace: namespace}
}

func (store *unsupportedStore) Save(ctx context.Context, key string, credential Credential) error {
	return errors.New("credential store is only supported on Windows")
}

func (store *unsupportedStore) Load(ctx context.Context, key string) (*Credential, error) {
	return nil, errors.New("credential store is only supported on Windows")
}

func (store *unsupportedStore) Delete(ctx context.Context, key string) error {
	return errors.New("credential store is only supported on Windows")
}
