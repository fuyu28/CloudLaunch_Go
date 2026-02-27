//go:build !windows

package app

func (app *App) startHotkey() error { return nil }

func (app *App) stopHotkey() {}
