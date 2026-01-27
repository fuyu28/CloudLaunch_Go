//go:build windows

// @fileoverview Windows向けの認証情報ストア初期化を提供する。
package app

import (
	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/credentials"
)

func newCredentialStore(cfg config.Config) credentials.Store {
	return credentials.NewWindowsStore(cfg.CredentialNamespace)
}
