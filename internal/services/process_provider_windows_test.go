//go:build windows

package services

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func TestProcessesFromPowerShellCSV(t *testing.T) {
	t.Parallel()

	csv := "\"ProcessName\",\"Id\",\"Path\"\n" +
		"\"game\",\"123\",\"C:\\games\\game.exe\"\n" +
		"\"helper\",\"0\",\"C:\\games\\helper.exe\"\n" +
		"\"note\",\"10\",\"C:\\docs\\note.txt\"\n" +
		"\"bare\",\"45\",\"\"\n"

	got, err := processesFromPowerShellCSV([]byte(csv))
	if err != nil {
		t.Fatalf("processesFromPowerShellCSV() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %#v", len(got), got)
	}
	if got[0] != (ProcessInfo{Name: "game.exe", Pid: 123, Cmd: `C:\games\game.exe`}) {
		t.Fatalf("got[0] = %#v", got[0])
	}
	if got[1] != (ProcessInfo{Name: "bare.exe", Pid: 45, Cmd: "bare.exe"}) {
		t.Fatalf("got[1] = %#v", got[1])
	}
}

func TestProcessesFromWmicCSV(t *testing.T) {
	t.Parallel()

	csv := "Node,Name,ProcessId,ExecutablePath\n" +
		"HOST,game.exe,123,C:\\games\\game.exe\n" +
		"HOST,skip.exe,abc,C:\\games\\skip.exe\n" +
		"HOST,tool,77,\n"

	got, err := processesFromWmicCSV([]byte(csv))
	if err != nil {
		t.Fatalf("processesFromWmicCSV() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %#v", len(got), got)
	}
	if got[0] != (ProcessInfo{Name: "game.exe", Pid: 123, Cmd: `C:\games\game.exe`}) {
		t.Fatalf("got[0] = %#v", got[0])
	}
	if got[1] != (ProcessInfo{Name: "tool.exe", Pid: 77, Cmd: "tool.exe"}) {
		t.Fatalf("got[1] = %#v", got[1])
	}
}

func TestParseCSVBytesUTF16LE(t *testing.T) {
	t.Parallel()

	text := "Name,Id\r\ngame.exe,1\r\n"
	encoded, err := io.ReadAll(transform.NewReader(
		strings.NewReader(text),
		unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder(),
	))
	if err != nil {
		t.Fatalf("encode UTF-16LE: %v", err)
	}
	if !bytes.Contains(encoded, []byte{0x00}) {
		t.Fatalf("expected UTF-16LE output to contain NUL bytes")
	}

	records, err := parseCSVBytes(encoded)
	if err != nil {
		t.Fatalf("parseCSVBytes() error = %v", err)
	}
	if len(records) != 2 || records[1][0] != "game.exe" {
		t.Fatalf("unexpected records: %#v", records)
	}
}

func TestParseCSVBytesShiftJISFallback(t *testing.T) {
	t.Parallel()

	sjisName, err := io.ReadAll(transform.NewReader(
		strings.NewReader("ゲーム"),
		japanese.ShiftJIS.NewEncoder(),
	))
	if err != nil {
		t.Fatalf("encode Shift-JIS: %v", err)
	}
	output := append([]byte("Name,Id\n"), sjisName...)
	output = append(output, []byte(",1\n")...)
	if utf8.Valid(output) {
		t.Fatalf("expected Shift-JIS payload to be invalid UTF-8")
	}

	records, err := parseCSVBytes(output)
	if err != nil {
		t.Fatalf("parseCSVBytes() error = %v", err)
	}
	if len(records) != 2 || records[1][0] != "ゲーム" {
		t.Fatalf("unexpected records: %#v", records)
	}
}

func TestListWindowsProcessesFallsBack(t *testing.T) {
	t.Parallel()

	var warnBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&warnBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	got, source := listWindowsProcesses(
		logger,
		func() ([]ProcessInfo, error) { return nil, errors.New("native failed") },
		func() ([]ProcessInfo, error) {
			return []ProcessInfo{{Name: "game.exe", Pid: 9, Cmd: `C:\games\game.exe`}}, nil
		},
	)
	if source != "fallback" {
		t.Fatalf("source = %q, want fallback", source)
	}
	if len(got) != 1 || got[0].Pid != 9 {
		t.Fatalf("unexpected processes: %#v", got)
	}
	if !strings.Contains(warnBuf.String(), "フォールバックを使用します") {
		t.Fatalf("expected fallback warn log, got %q", warnBuf.String())
	}
}

func TestListWindowsProcessesNativeSuccess(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	got, source := listWindowsProcesses(
		logger,
		func() ([]ProcessInfo, error) {
			return []ProcessInfo{{Name: "game.exe", Pid: 1, Cmd: `C:\games\game.exe`}}, nil
		},
		func() ([]ProcessInfo, error) {
			t.Fatal("fallback should not be called")
			return nil, nil
		},
	)
	if source != "native" || len(got) != 1 {
		t.Fatalf("source=%q processes=%#v", source, got)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no warn/error logs, got %q", buf.String())
	}
}

func TestListWindowsProcessesBothFail(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	got, source := listWindowsProcesses(
		logger,
		func() ([]ProcessInfo, error) { return nil, errors.New("native failed") },
		func() ([]ProcessInfo, error) { return nil, errors.New("fallback failed") },
	)
	if source != "fallback" || len(got) != 0 {
		t.Fatalf("source=%q processes=%#v", source, got)
	}
	logText := buf.String()
	if !strings.Contains(logText, "フォールバックを使用します") || !strings.Contains(logText, "フォールバックも失敗しました") {
		t.Fatalf("expected warn+error logs, got %q", logText)
	}
}

func TestDefaultProcessProviderReturnsNativeOrFallback(t *testing.T) {
	t.Parallel()

	provider := defaultProcessProvider(slog.New(slog.NewTextHandler(io.Discard, nil)))
	_, source := provider()
	if source != "native" && source != "fallback" {
		t.Fatalf("source = %q, want native or fallback", source)
	}
}
