package app

import (
	"log/slog"
	"testing"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/services"
)

type stubHotkeyService struct {
	starts     int
	stops      int
	notifySet  []bool
	startError error
}

func (s *stubHotkeyService) Start() error {
	s.starts++
	return s.startError
}

func (s *stubHotkeyService) Stop() {
	s.stops++
}

func (s *stubHotkeyService) SetNotify(enabled bool) {
	s.notifySet = append(s.notifySet, enabled)
}

func TestUpdateScreenshotHotkeyNoopWhenUnchanged(t *testing.T) {
	stub := &stubHotkeyService{}
	app := &App{
		Config:        config.Config{ScreenshotHotkey: "Ctrl+Alt+S"},
		Logger:        slog.Default(),
		HotkeyService: stub,
	}

	result := app.UpdateScreenshotHotkey("Ctrl+Alt+S")
	if !result.Success {
		t.Fatalf("expected success, got %#v", result)
	}
	if stub.starts != 0 || stub.stops != 0 {
		t.Fatalf("expected no restart, starts=%d stops=%d", stub.starts, stub.stops)
	}
}

func TestUpdateScreenshotHotkeyNotifyUpdatesFlagWithoutRestart(t *testing.T) {
	stub := &stubHotkeyService{}
	app := &App{
		Config:        config.Config{ScreenshotHotkeyNotify: true},
		Logger:        slog.Default(),
		HotkeyService: stub,
	}

	result := app.UpdateScreenshotHotkeyNotify(false)
	if !result.Success {
		t.Fatalf("expected success, got %#v", result)
	}
	if app.Config.ScreenshotHotkeyNotify {
		t.Fatal("expected notify config to be false")
	}
	if stub.starts != 0 || stub.stops != 0 {
		t.Fatalf("expected no OS re-register, starts=%d stops=%d", stub.starts, stub.stops)
	}
	if len(stub.notifySet) != 1 || stub.notifySet[0] {
		t.Fatalf("expected SetNotify(false), got %#v", stub.notifySet)
	}

	// 同値なら SetNotify も呼ばない
	result = app.UpdateScreenshotHotkeyNotify(false)
	if !result.Success {
		t.Fatalf("expected success on noop, got %#v", result)
	}
	if len(stub.notifySet) != 1 {
		t.Fatalf("expected no additional SetNotify, got %#v", stub.notifySet)
	}
}

func TestUpdateScreenshotHotkeyRejectsUnknownKey(t *testing.T) {
	stub := &stubHotkeyService{}
	app := &App{
		Config:        config.Config{ScreenshotHotkey: "Ctrl+Alt+S"},
		Logger:        slog.Default(),
		HotkeyService: stub,
	}

	result := app.UpdateScreenshotHotkey("Ctrl+Foo")
	if result.Success {
		t.Fatal("expected failure for unknown key")
	}
	if app.Config.ScreenshotHotkey != "Ctrl+Alt+S" {
		t.Fatalf("config should stay unchanged, got %q", app.Config.ScreenshotHotkey)
	}
}

func TestUpdateScreenshotHotkeyRejectsPrintScreenAndF12(t *testing.T) {
	stub := &stubHotkeyService{}
	app := &App{
		Config:        config.Config{ScreenshotHotkey: "Ctrl+Alt+S"},
		Logger:        slog.Default(),
		HotkeyService: stub,
	}

	for _, combo := range []string{"PrintScreen", "F12", "Ctrl+F12"} {
		result := app.UpdateScreenshotHotkey(combo)
		if result.Success {
			t.Fatalf("expected failure for %q", combo)
		}
	}
	if app.Config.ScreenshotHotkey != "Ctrl+Alt+S" {
		t.Fatalf("config should stay unchanged, got %q", app.Config.ScreenshotHotkey)
	}
}

func TestUpdateScreenshotHotkeyAcceptsF8(t *testing.T) {
	stub := &stubHotkeyService{}
	app := &App{
		Config:        config.Config{ScreenshotHotkey: "Ctrl+Alt+S"},
		Logger:        slog.Default(),
		HotkeyService: stub,
	}

	result := app.UpdateScreenshotHotkey("F8")
	if !result.Success {
		t.Fatalf("expected success, got %#v", result)
	}
	if app.Config.ScreenshotHotkey != "F8" {
		t.Fatalf("config not updated: %q", app.Config.ScreenshotHotkey)
	}
}

// Ensure stub satisfies interface at compile time.
var _ services.HotkeyService = (*stubHotkeyService)(nil)
