package services

import "testing"

func TestWindowsPathBase(t *testing.T) {
	t.Parallel()

	got := windowsPathBase(`C:\hoge\hogehoge.exe`)
	if got != "hogehoge.exe" {
		t.Fatalf("expected hogehoge.exe, got %q", got)
	}
}

func TestWindowsPathDir(t *testing.T) {
	t.Parallel()

	got := windowsPathDir(`C:\hoge\hogehoge.exe`)
	if got != "C:/hoge" {
		t.Fatalf("expected C:/hoge, got %q", got)
	}
}
