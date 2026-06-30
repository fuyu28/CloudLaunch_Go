package storage

import (
	"path/filepath"
	"testing"
)

func TestBlobHashBytesReturnsSHA256Hex(t *testing.T) {
	t.Parallel()

	got := blobHashBytes([]byte("hello world"))
	want := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if got != want {
		t.Fatalf("hash = %q, want %q", got, want)
	}
}

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
