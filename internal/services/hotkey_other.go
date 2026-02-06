//go:build !windows

package services

import (
	"errors"
	"log/slog"
)

type hotkeyServiceUnsupported struct {
	logger *slog.Logger
}

func newHotkeyService(logger *slog.Logger, _ HotkeyConfig, _ HotkeyHandler) HotkeyService {
	return &hotkeyServiceUnsupported{logger: logger}
}

func (service *hotkeyServiceUnsupported) Start() error {
	if service.logger != nil {
		service.logger.Warn("ホットキーはこのOSでは利用できません")
	}
	return errors.New("hotkey is not supported")
}

func (service *hotkeyServiceUnsupported) Stop() {}
