package app

import (
	"testing"
	"time"
)

func TestResolveSaveHashUpdatedAtUsesNowWhenEmpty(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 4, 12, 34, 56, 0, time.FixedZone("JST", 9*60*60))

	resolved, err := resolveSaveHashUpdatedAt("", func() time.Time { return now })
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !resolved.Equal(now) {
		t.Fatalf("expected %v, got %v", now, resolved)
	}
}

func TestResolveSaveHashUpdatedAtParsesRFC3339(t *testing.T) {
	t.Parallel()

	expected := time.Date(2026, 5, 4, 12, 34, 56, 123000000, time.UTC)

	resolved, err := resolveSaveHashUpdatedAt("2026-05-04T12:34:56.123Z", time.Now)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !resolved.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, resolved)
	}
}

func TestResolveSaveHashUpdatedAtRejectsInvalidFormat(t *testing.T) {
	t.Parallel()

	_, err := resolveSaveHashUpdatedAt("2026/05/04 12:34:56", time.Now)
	if err == nil {
		t.Fatal("expected parse error")
	}
}
