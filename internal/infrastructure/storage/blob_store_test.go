package storage

import (
	"path/filepath"
	"testing"
)

func TestResolveSafeRelativePathAcceptsNestedRelativePath(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	got, err := ResolveSafeRelativePath(base, "slot/001.sav")
	if err != nil {
		t.Fatalf("ResolveSafeRelativePath returned error: %v", err)
	}

	want := filepath.Join(base, "slot", "001.sav")
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestResolveSafeRelativePathRejectsEscapingPath(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	cases := []string{
		"",
		"..",
		"../outside.sav",
		"slot/../../outside.sav",
		"/absolute.sav",
	}

	for _, tc := range cases {
		if got, err := ResolveSafeRelativePath(base, tc); err == nil {
			t.Fatalf("ResolveSafeRelativePath(%q) = %q, want error", tc, got)
		}
	}
}
