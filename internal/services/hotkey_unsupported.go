//go:build !windows

package services

import (
	"errors"
	"log/slog"
)

type hotkeyServiceUnsupported struct{}

func newHotkeyService(logger *slog.Logger, config HotkeyConfig, handler HotkeyHandler) HotkeyService {
	return &hotkeyServiceUnsupported{}
}

func (service *hotkeyServiceUnsupported) Start() error {
	return errors.New("hotkey is only supported on Windows")
}

func (service *hotkeyServiceUnsupported) Stop() {}
