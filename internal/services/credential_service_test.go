package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"CloudLaunch_Go/internal/credentials"
)

type fakeCredentialStore struct {
	savedKey        string
	savedCredential credentials.Credential
	loadedKey       string
	deletedKey      string
	loadResult      *credentials.Credential
	saveErr         error
	loadErr         error
	deleteErr       error
}

func (store *fakeCredentialStore) Save(ctx context.Context, key string, credential credentials.Credential) error {
	store.savedKey = key
	store.savedCredential = credential
	return store.saveErr
}

func (store *fakeCredentialStore) Load(ctx context.Context, key string) (*credentials.Credential, error) {
	store.loadedKey = key
	return store.loadResult, store.loadErr
}

func (store *fakeCredentialStore) Delete(ctx context.Context, key string) error {
	store.deletedKey = key
	return store.deleteErr
}

func TestCredentialServiceSaveCredentialUsesStoreBoundary(t *testing.T) {
	t.Parallel()

	store := &fakeCredentialStore{}
	service := NewCredentialService(store, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.SaveCredential(context.Background(), " default ", CredentialInput{
		AccessKeyID:     " access ",
		SecretAccessKey: " secret ",
		BucketName:      " bucket ",
		Region:          " region ",
		Endpoint:        " endpoint ",
	})
	if !result.Success {
		t.Fatalf("expected success, got %#v", result.Error)
	}
	if store.savedKey != "default" {
		t.Fatalf("expected trimmed key, got %q", store.savedKey)
	}
	if store.savedCredential.AccessKeyID != "access" || store.savedCredential.SecretAccessKey != "secret" {
		t.Fatalf("expected trimmed credentials, got %#v", store.savedCredential)
	}
	if store.savedCredential.BucketName != "bucket" || store.savedCredential.Region != "region" || store.savedCredential.Endpoint != "endpoint" {
		t.Fatalf("expected trimmed config, got %#v", store.savedCredential)
	}
}

func TestCredentialServiceDeleteCredentialReturnsStoreError(t *testing.T) {
	t.Parallel()

	store := &fakeCredentialStore{deleteErr: errors.New("boom")}
	service := NewCredentialService(store, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.DeleteCredential(context.Background(), "default")
	if result.Success {
		t.Fatal("expected error result")
	}
	if store.deletedKey != "default" {
		t.Fatalf("expected delete key to be forwarded, got %q", store.deletedKey)
	}
}
