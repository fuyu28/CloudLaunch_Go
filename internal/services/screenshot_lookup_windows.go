//go:build windows

package services

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"CloudLaunch_Go/internal/models"
)

func (service *ScreenshotService) findGameByPID(ctx context.Context, pid int) (*models.Game, *ProcessInfo, error) {
	if service.processMonitor == nil {
		return nil, nil, errors.New("process monitor is nil")
	}
	if pid <= 0 {
		return nil, nil, errors.New("pid is invalid")
	}
	proc, err := service.processMonitor.FindProcessByPID(pid)
	if err != nil {
		return nil, nil, err
	}
	if proc == nil {
		return nil, nil, errors.New("process not found")
	}
	exePath := strings.TrimSpace(proc.Cmd)
	if exePath == "" {
		exePath = proc.Name
	}
	game, err := service.repository.GetGameByExePath(ctx, exePath)
	if err != nil {
		return nil, proc, err
	}
	if game != nil {
		return game, proc, nil
	}

	games, err := service.repository.ListGames(ctx, "", "", "", "")
	if err != nil {
		return nil, proc, err
	}
	exeName := strings.ToLower(filepath.Base(exePath))
	for _, g := range games {
		if strings.ToLower(filepath.Base(g.ExePath)) == exeName {
			game := g
			return &game, proc, nil
		}
	}

	return nil, proc, errors.New("game not found for process")
}
