package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/domain"
)

type fakeScreenshotRepository struct {
	getGameByIDFn func(ctx context.Context, gameID string) (*domain.Game, error)
}

func (repository fakeScreenshotRepository) GetGameByID(ctx context.Context, gameID string) (*domain.Game, error) {
	return repository.getGameByIDFn(ctx, gameID)
}

// fakeProcessIDResolver は ProcessIDResolver のテスト用スタブ。
type fakeProcessIDResolver struct {
	findFn func(exePath string) ([]int, error)
}

func (resolver fakeProcessIDResolver) FindProcessIDsByExe(exePath string) ([]int, error) {
	return resolver.findFn(exePath)
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// resolverReturning は常に指定PIDを返す resolver を作る。
func resolverReturning(pids ...int) fakeProcessIDResolver {
	return fakeProcessIDResolver{findFn: func(string) ([]int, error) {
		return pids, nil
	}}
}

func TestScreenshotServiceCaptureGameScreenshotRejectsEmptyGameID(t *testing.T) {
	t.Parallel()

	service := NewScreenshotService(config.Config{}, fakeScreenshotRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return nil, nil
		},
	}, resolverReturning(1234), newTestLogger())

	_, err := service.CaptureGameScreenshot(context.Background(), "   ")
	if err == nil {
		t.Fatalf("expected empty game id error")
	}
}

func TestScreenshotServiceCaptureGameScreenshotReturnsRepositoryError(t *testing.T) {
	t.Parallel()

	service := NewScreenshotService(config.Config{}, fakeScreenshotRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return nil, errors.New("db down")
		},
	}, resolverReturning(1234), newTestLogger())

	_, err := service.CaptureGameScreenshot(context.Background(), "game-1")
	if err == nil {
		t.Fatalf("expected repository error")
	}
}

func TestScreenshotServiceBuildScreenshotPathsUsesConfiguredExtension(t *testing.T) {
	t.Parallel()

	service := NewScreenshotService(config.Config{ScreenshotLocalJpeg: true}, fakeScreenshotRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return nil, nil
		},
	}, nil, newTestLogger())

	fullPath, err := service.buildScreenshotPaths("game-1", t.TempDir())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.HasSuffix(fullPath, ".jpg") {
		t.Fatalf("expected jpg path, got %s", fullPath)
	}
	if !strings.Contains(fullPath, "game-1") {
		t.Fatalf("expected path to contain game id, got %s", fullPath)
	}
}

func TestScreenshotServiceCaptureGameScreenshotReturnsNotFoundWhenGameMissing(t *testing.T) {
	t.Parallel()

	service := NewScreenshotService(config.Config{}, fakeScreenshotRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return nil, nil
		},
	}, resolverReturning(1234), newTestLogger())

	_, err := service.CaptureGameScreenshot(context.Background(), "game-1")
	if err == nil || err.Error() != "game not found" {
		t.Fatalf("expected game not found error, got %v", err)
	}
}

func TestScreenshotServiceCaptureGameScreenshotFailsWhenProcessNotRunning(t *testing.T) {
	t.Parallel()

	service := NewScreenshotService(config.Config{AppDataDir: t.TempDir()}, fakeScreenshotRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return &domain.Game{ID: gameID, Title: "Game", ExePath: "game.exe"}, nil
		},
	}, resolverReturning(), newTestLogger())

	captured := false
	service.captureFunc = func(ctx context.Context, pid int, outPath string) error {
		captured = true
		return nil
	}

	_, err := service.CaptureGameScreenshot(context.Background(), "game-1")
	if err == nil {
		t.Fatalf("expected process-not-found error")
	}
	if captured {
		t.Fatalf("captureFunc must not run when process is not found")
	}
}

func TestScreenshotServiceCaptureGameScreenshotPassesResolvedPID(t *testing.T) {
	t.Parallel()

	service := NewScreenshotService(config.Config{AppDataDir: t.TempDir()}, fakeScreenshotRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return &domain.Game{ID: gameID, Title: "Game", ExePath: "game.exe"}, nil
		},
	}, resolverReturning(4242, 9999), newTestLogger())

	gotPID := -1
	service.captureFunc = func(ctx context.Context, pid int, outPath string) error {
		gotPID = pid
		return nil
	}

	if _, err := service.CaptureGameScreenshot(context.Background(), "game-1"); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	// 複数PIDでは先頭を採用する。
	if gotPID != 4242 {
		t.Fatalf("expected pid 4242, got %d", gotPID)
	}
}

func TestScreenshotServiceCaptureGameScreenshotReturnsResolverError(t *testing.T) {
	t.Parallel()

	service := NewScreenshotService(config.Config{AppDataDir: t.TempDir()}, fakeScreenshotRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return &domain.Game{ID: gameID, Title: "Game", ExePath: "game.exe"}, nil
		},
	}, fakeProcessIDResolver{findFn: func(string) ([]int, error) {
		return nil, errors.New("boom")
	}}, newTestLogger())

	captured := false
	service.captureFunc = func(ctx context.Context, pid int, outPath string) error {
		captured = true
		return nil
	}

	_, err := service.CaptureGameScreenshot(context.Background(), "game-1")
	if err == nil {
		t.Fatalf("expected resolver error")
	}
	// プロセス一覧取得失敗は「起動していない」と誤診せず ServiceError として表面化する。
	serviceErr := &ServiceError{}
	if !errors.As(err, &serviceErr) || serviceErr.Message != "プロセス一覧の取得に失敗しました" {
		t.Fatalf("expected service error, got %v", err)
	}
	if captured {
		t.Fatalf("captureFunc must not run when resolver fails")
	}
}

func TestScreenshotServiceCaptureGameScreenshotReturnsCaptureError(t *testing.T) {
	t.Parallel()

	captureErr := errors.New("capture failed")
	service := NewScreenshotService(config.Config{AppDataDir: t.TempDir()}, fakeScreenshotRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return &domain.Game{ID: gameID, Title: "Game", ExePath: "game.exe"}, nil
		},
	}, resolverReturning(1234), newTestLogger())
	service.captureFunc = func(ctx context.Context, pid int, outPath string) error {
		return captureErr
	}

	_, err := service.CaptureGameScreenshot(context.Background(), "game-1")
	if !errors.Is(err, captureErr) {
		t.Fatalf("expected capture error, got %v", err)
	}
}

func TestScreenshotServiceCaptureGameScreenshotReturnsPathOnSuccess(t *testing.T) {
	t.Parallel()

	service := NewScreenshotService(config.Config{AppDataDir: t.TempDir()}, fakeScreenshotRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return &domain.Game{ID: gameID, Title: "Game", ExePath: "game.exe"}, nil
		},
	}, resolverReturning(1234), newTestLogger())
	service.captureFunc = func(ctx context.Context, pid int, outPath string) error {
		return nil
	}

	path, err := service.CaptureGameScreenshot(context.Background(), "game-1")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !strings.HasSuffix(path, ".png") {
		t.Fatalf("expected png path, got %s", path)
	}
	if !strings.Contains(path, "game-1") {
		t.Fatalf("expected path to contain game id, got %s", path)
	}
}

// TestScreenshotServiceResolvePID は PID 解決の分岐を検証する。
// 列挙失敗（error）と「見つからない（pid 0）」を明確に区別することが要点。
func TestScreenshotServiceResolvePID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resolver ProcessIDResolver
		exePath  string
		wantPID  int
		wantErr  bool
	}{
		{
			name:     "found returns first pid",
			resolver: resolverReturning(11, 22),
			exePath:  "game.exe",
			wantPID:  11,
		},
		{
			name:     "no process returns zero",
			resolver: resolverReturning(),
			exePath:  "game.exe",
			wantPID:  0,
		},
		{
			name:     "nil resolver returns zero",
			resolver: nil,
			exePath:  "game.exe",
			wantPID:  0,
		},
		{
			name:     "empty exe path returns zero",
			resolver: resolverReturning(11),
			exePath:  "   ",
			wantPID:  0,
		},
		{
			name: "resolver error propagates",
			resolver: fakeProcessIDResolver{findFn: func(string) ([]int, error) {
				return nil, errors.New("boom")
			}},
			exePath: "game.exe",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			service := NewScreenshotService(config.Config{}, nil, tc.resolver, newTestLogger())
			pid, err := service.resolvePID(tc.exePath)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pid != tc.wantPID {
				t.Fatalf("pid: want %d got %d", tc.wantPID, pid)
			}
		})
	}
}

// TestScreenshotServiceCaptureHotkeyRequiresPIDForTargetGame は、対象ゲームがある場合に
// PID が引けないとエラーになり、無関係なフォアグラウンドウィンドウを撮らないことを検証する。
func TestScreenshotServiceCaptureHotkeyRequiresPIDForTargetGame(t *testing.T) {
	t.Parallel()

	service := NewScreenshotService(config.Config{AppDataDir: t.TempDir()}, fakeScreenshotRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return &domain.Game{ID: gameID, Title: "Game", ExePath: "game.exe"}, nil
		},
	}, resolverReturning(), newTestLogger())

	captured := false
	service.captureFunc = func(ctx context.Context, pid int, outPath string) error {
		captured = true
		return nil
	}

	_, _, err := service.CaptureHotkey(context.Background(), "game-1")
	if err == nil {
		t.Fatalf("expected error when target game process is not found")
	}
	if captured {
		t.Fatalf("captureFunc must not run when target game PID is unresolved")
	}
}

// TestScreenshotServiceCaptureHotkeyPropagatesResolverError は、対象ゲームありで
// プロセス列挙が失敗したとき、その error をそのまま返すことを検証する。
func TestScreenshotServiceCaptureHotkeyPropagatesResolverError(t *testing.T) {
	t.Parallel()

	service := NewScreenshotService(config.Config{AppDataDir: t.TempDir()}, fakeScreenshotRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return &domain.Game{ID: gameID, Title: "Game", ExePath: "game.exe"}, nil
		},
	}, fakeProcessIDResolver{findFn: func(string) ([]int, error) {
		return nil, errors.New("boom")
	}}, newTestLogger())

	service.captureFunc = func(ctx context.Context, pid int, outPath string) error {
		return nil
	}

	_, _, err := service.CaptureHotkey(context.Background(), "game-1")
	if err == nil {
		t.Fatalf("expected resolver error to propagate")
	}
}

// TestScreenshotServiceCaptureHotkeyNoTargetUsesForeground は、対象ゲームが無いとき
// pid 0（フォアグラウンド）で captureFunc が呼ばれ、gameID 空で返ることを検証する。
func TestScreenshotServiceCaptureHotkeyNoTargetUsesForeground(t *testing.T) {
	t.Parallel()

	service := NewScreenshotService(config.Config{AppDataDir: t.TempDir()}, fakeScreenshotRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return nil, nil
		},
	}, resolverReturning(4242), newTestLogger())

	gotPID := -1
	service.captureFunc = func(ctx context.Context, pid int, outPath string) error {
		gotPID = pid
		return nil
	}

	gameID, path, err := service.CaptureHotkey(context.Background(), "")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if gotPID != 0 {
		t.Fatalf("expected foreground pid 0, got %d", gotPID)
	}
	if gameID != "" {
		t.Fatalf("expected empty gameID for foreground capture, got %q", gameID)
	}
	if path == "" {
		t.Fatalf("expected non-empty path")
	}
}

// TestScreenshotServiceCaptureHotkeyPassesResolvedPID は、対象ゲームありで PID 解決成功時に
// その pid が captureFunc に渡り、gameID が返ることを検証する。
func TestScreenshotServiceCaptureHotkeyPassesResolvedPID(t *testing.T) {
	t.Parallel()

	service := NewScreenshotService(config.Config{AppDataDir: t.TempDir()}, fakeScreenshotRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*domain.Game, error) {
			return &domain.Game{ID: gameID, Title: "Game", ExePath: "game.exe"}, nil
		},
	}, resolverReturning(7777, 8888), newTestLogger())

	gotPID := -1
	service.captureFunc = func(ctx context.Context, pid int, outPath string) error {
		gotPID = pid
		return nil
	}

	gameID, _, err := service.CaptureHotkey(context.Background(), "game-1")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if gotPID != 7777 {
		t.Fatalf("expected pid 7777, got %d", gotPID)
	}
	if gameID != "game-1" {
		t.Fatalf("expected gameID game-1, got %q", gameID)
	}
}

func TestBuildScreencapArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		pid         int
		outPath     string
		localJpeg   bool
		jpegQuality int
		want        []string
	}{
		{
			name:    "pid png",
			pid:     1234,
			outPath: "out.png",
			want:    []string{"cap", "--method", "wgc-window", "--pid", "1234", "--out", "out.png", "--json", "--overwrite", "--no-log"},
		},
		{
			name:    "foreground png",
			pid:     0,
			outPath: "out.png",
			want:    []string{"cap", "--method", "wgc-window", "--foreground", "--out", "out.png", "--json", "--overwrite", "--no-log"},
		},
		{
			name:        "jpeg valid quality",
			pid:         5,
			outPath:     "out.jpg",
			localJpeg:   true,
			jpegQuality: 70,
			want:        []string{"cap", "--method", "wgc-window", "--pid", "5", "--out", "out.jpg", "--json", "--overwrite", "--no-log", "--format", "jpg", "--quality", "70"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := buildScreencapArgs(tc.pid, tc.outPath, tc.localJpeg, tc.jpegQuality)
			if strings.Join(got, " ") != strings.Join(tc.want, " ") {
				t.Fatalf("args mismatch\n want: %v\n got:  %v", tc.want, got)
			}
		})
	}
}

func TestNormalizeJpegQuality(t *testing.T) {
	t.Parallel()

	cases := map[int]int{0: 85, -5: 85, 101: 85, 1: 1, 100: 100, 85: 85, 50: 50}
	for in, want := range cases {
		if got := normalizeJpegQuality(in); got != want {
			t.Fatalf("normalizeJpegQuality(%d): want %d got %d", in, want, got)
		}
	}
}

func TestScreencapErrorFromResult(t *testing.T) {
	t.Parallel()

	t.Run("with error info", func(t *testing.T) {
		t.Parallel()
		result := &screencapResult{Error: &screencapError{
			Message: "no window",
			Where:   "wgc",
			HResult: []byte(`"0x80070005"`),
		}}
		err := screencapErrorFromResult(result)
		if err == nil {
			t.Fatalf("expected error")
		}
		for _, want := range []string{"no window", "wgc", "0x80070005"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error %q missing %q", err.Error(), want)
			}
		}
	})

	t.Run("without error info", func(t *testing.T) {
		t.Parallel()
		err := screencapErrorFromResult(&screencapResult{})
		if err == nil {
			t.Fatalf("expected generic error")
		}
	})
}

func TestScreencapExitError(t *testing.T) {
	t.Parallel()

	t.Run("prefers stderr", func(t *testing.T) {
		t.Parallel()
		err := screencapExitError(3, "out text", "err text")
		if err == nil || !strings.Contains(err.Error(), "err text") || !strings.Contains(err.Error(), "exit=3") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("falls back to stdout", func(t *testing.T) {
		t.Parallel()
		err := screencapExitError(1, "out text", "   ")
		if err == nil || !strings.Contains(err.Error(), "out text") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestParseScreencapResult(t *testing.T) {
	t.Parallel()

	t.Run("success json", func(t *testing.T) {
		t.Parallel()
		stdout := []byte(`{"ok":true,"out_path":"C:/shot.png","image_stats":{"black_ratio":0.12,"transparent_ratio":0.0,"avg_luma":120.5},"error":null}`)
		result, err := parseScreencapResult(stdout)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.OK {
			t.Fatalf("expected ok=true")
		}
		if result.OutPath != "C:/shot.png" {
			t.Fatalf("unexpected out_path: %s", result.OutPath)
		}
		if result.ImageStats.BlackRatio != 0.12 {
			t.Fatalf("unexpected black_ratio: %v", result.ImageStats.BlackRatio)
		}
		if result.Error != nil {
			t.Fatalf("expected nil error, got %v", result.Error)
		}
	})

	t.Run("failure json with string hresult", func(t *testing.T) {
		t.Parallel()
		stdout := []byte(`{"ok":false,"error":{"message":"no window","where":"wgc","hresult":"0x80070005"}}`)
		result, err := parseScreencapResult(stdout)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.OK {
			t.Fatalf("expected ok=false")
		}
		if result.Error == nil || result.Error.Message != "no window" {
			t.Fatalf("unexpected error info: %+v", result.Error)
		}
		if string(result.Error.HResult) != `"0x80070005"` {
			t.Fatalf("unexpected hresult: %s", string(result.Error.HResult))
		}
	})

	t.Run("failure json with numeric hresult", func(t *testing.T) {
		t.Parallel()
		stdout := []byte(`{"ok":false,"error":{"message":"fail","where":"capture","hresult":-2147024891}}`)
		result, err := parseScreencapResult(stdout)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Error == nil || string(result.Error.HResult) != "-2147024891" {
			t.Fatalf("unexpected hresult: %+v", result.Error)
		}
	})

	t.Run("empty output", func(t *testing.T) {
		t.Parallel()
		if _, err := parseScreencapResult([]byte("   ")); err == nil {
			t.Fatalf("expected error for empty output")
		}
	})

	t.Run("broken json", func(t *testing.T) {
		t.Parallel()
		if _, err := parseScreencapResult([]byte(`{"ok":true,`)); err == nil {
			t.Fatalf("expected error for broken json")
		}
	})
}
