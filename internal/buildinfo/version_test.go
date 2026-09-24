package buildinfo

import (
	"runtime/debug"
	"testing"
)

func TestRevisionFromSettingsReturnsCommitHash(t *testing.T) {
	revision := revisionFromSettings([]debug.BuildSetting{
		{Key: "vcs.time", Value: "2026-09-24T15:44:45Z"},
		{Key: "vcs.revision", Value: "f94ba468acd095752b20d0e258322319a9726836"},
	})

	if revision != "f94ba468acd095752b20d0e258322319a9726836" {
		t.Fatalf("revision = %q", revision)
	}
}

func TestRevisionFromSettingsReturnsUnknownWithoutCommitHash(t *testing.T) {
	revision := revisionFromSettings([]debug.BuildSetting{{Key: "vcs.revision", Value: ""}})

	if revision != unknownRevision {
		t.Fatalf("revision = %q, want %q", revision, unknownRevision)
	}
}
