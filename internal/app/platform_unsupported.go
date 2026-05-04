//go:build !windows

package app

import (
	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/credentials"
)

func (app *App) startHotkey() error {
	return nil
}

func (app *App) stopHotkey() {}

func newCredentialStore(cfg config.Config) credentials.Store {
	return credentials.NewUnsupportedStore(cfg.CredentialNamespace)
}
