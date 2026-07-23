//go:build !windows

package services

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"CloudLaunch_Go/internal/domain"
)

func TestDefaultProcessProviderUnsupported(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	provider := defaultProcessProvider(logger)

	processes, source := provider()
	if source != "unsupported" {
		t.Fatalf("source = %q, want unsupported", source)
	}
	if len(processes) != 0 {
		t.Fatalf("processes = %#v, want empty", processes)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no logs for unsupported enumeration, got %q", buf.String())
	}

	// 監視ループ相当の再呼び出しでも警告を出さない。
	_, _ = provider()
	_, _ = provider()
	if buf.Len() != 0 {
		t.Fatalf("expected still no logs after repeated calls, got %q", buf.String())
	}
}

func TestNewProcessMonitorServiceUsesUnsupportedProvider(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	service := NewProcessMonitorService(fakeProcessMonitorRepository{
		createPlaySessionAndRefreshGameFn: func(ctx context.Context, session domain.PlaySession) (*domain.PlaySession, error) {
			return &session, nil
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) { return nil, nil },
		updateGameFn:  func(ctx context.Context, game domain.Game) (*domain.Game, error) { return &game, nil },
		listGamesFn: func(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error) {
			return nil, nil
		},
	}, logger, nil)

	snapshot := service.GetProcessSnapshot()
	if snapshot.Source != "unsupported" {
		t.Fatalf("snapshot.Source = %q, want unsupported", snapshot.Source)
	}
	if len(snapshot.Items) != 0 {
		t.Fatalf("snapshot.Items = %#v, want empty", snapshot.Items)
	}
	if strings.Contains(buf.String(), "フォールバック") {
		t.Fatalf("unexpected fallback warning on unsupported platform: %q", buf.String())
	}
}
