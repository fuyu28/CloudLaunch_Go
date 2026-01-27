//go:build !windows

// @fileoverview 非Windows向けの認証情報ストア初期化を提供する。
package app

import (
	"path/filepath"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/credentials"
)

func newCredentialStore(cfg config.Config) credentials.Store {
	return credentials.NewFileStore(filepath.Join(cfg.AppDataDir, "credentials"))
}
