package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"CloudLaunch_Go/internal/domain"
)

func TestHashBytesIsDeterministic(t *testing.T) {
	t.Parallel()

	data := []byte("hello world")
	h1 := hashBytes(data)
	h2 := hashBytes(data)

	if h1 != h2 {
		t.Fatalf("expected same hash, got %q and %q", h1, h2)
	}
	// SHA-256 of "hello world"
	want := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if h1 != want {
		t.Fatalf("hash = %q, want %q", h1, want)
	}
}

func TestHashBytesDifferentInputProducesDifferentHash(t *testing.T) {
	t.Parallel()

	if hashBytes([]byte("a")) == hashBytes([]byte("b")) {
		t.Fatal("different inputs produced same hash")
	}
}

func TestBuildSaveSnapshotReturnsErrorForMissingDir(t *testing.T) {
	t.Parallel()

	_, _, err := buildSaveSnapshot("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestBuildSaveSnapshotReturnsErrorForEmptyPath(t *testing.T) {
	t.Parallel()

	_, _, err := buildSaveSnapshot("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestBuildSaveSnapshotReturnsErrorForFile(t *testing.T) {
	t.Parallel()

	f, err := os.CreateTemp(t.TempDir(), "not-a-dir")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	_, _, err = buildSaveSnapshot(f.Name())
	if err == nil {
		t.Fatal("expected error when path is a file, not directory")
	}
}

func TestBuildSaveSnapshotWalksFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "slot1.sav"), []byte("save1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "slot2.sav"), []byte("save2"), 0o600); err != nil {
		t.Fatal(err)
	}

	snap, blobs, err := buildSaveSnapshot(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snap.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(snap.Files))
	}
	if len(blobs) != 2 {
		t.Fatalf("expected 2 blobs, got %d", len(blobs))
	}

	// ハッシュと blobs が一致しているか確認
	for relPath, hash := range snap.Files {
		data, ok := blobs[hash]
		if !ok {
			t.Fatalf("blob missing for %s (hash %s)", relPath, hash)
		}
		if hashBytes(data) != hash {
			t.Fatalf("blob content does not match hash for %s", relPath)
		}
	}
}

func TestBuildSaveSnapshotIsDeteministic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.sav"), []byte("aaa"), 0o600); err != nil {
		t.Fatal(err)
	}

	snap1, _, err := buildSaveSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	snap2, _, err := buildSaveSnapshot(dir)
	if err != nil {
		t.Fatal(err)
	}

	b1, _ := json.Marshal(snap1)
	b2, _ := json.Marshal(snap2)
	if string(b1) != string(b2) {
		t.Fatal("buildSaveSnapshot is not deterministic")
	}
}

func TestRemoveFilesNotInSnapshotRemovesStaleFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	nestedDir := filepath.Join(dir, "nested")
	if err := os.MkdirAll(nestedDir, 0o700); err != nil {
		t.Fatal(err)
	}
	keepPath := filepath.Join(nestedDir, "keep.sav")
	stalePath := filepath.Join(nestedDir, "stale.sav")
	if err := os.WriteFile(keepPath, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stalePath, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}

	snapshot := domain.SaveSnapshot{Files: map[string]domain.BlobHash{
		"nested/keep.sav": hashBytes([]byte("keep")),
	}}
	if err := removeFilesNotInSnapshot(dir, snapshot); err != nil {
		t.Fatalf("removeFilesNotInSnapshot: %v", err)
	}
	if _, err := os.Stat(keepPath); err != nil {
		t.Fatalf("keep file should remain: %v", err)
	}
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("stale file should be removed, stat err: %v", err)
	}
}

func TestRemoveFilesNotInSnapshotRemovesEmptyDirectories(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	staleDir := filepath.Join(dir, "stale")
	if err := os.MkdirAll(staleDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staleDir, "only.sav"), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := removeFilesNotInSnapshot(dir, domain.SaveSnapshot{Files: map[string]domain.BlobHash{}}); err != nil {
		t.Fatalf("removeFilesNotInSnapshot: %v", err)
	}
	if _, err := os.Stat(staleDir); !os.IsNotExist(err) {
		t.Fatalf("empty stale directory should be removed, stat err: %v", err)
	}
}

func TestBuildMetaSnapshotReturnsConsistentHashes(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	game := domain.Game{
		ID:            "game-1",
		Title:         "Test Game",
		Publisher:     "Test Publisher",
		PlayStatus:    domain.PlayStatusPlayed,
		TotalPlayTime: 3600,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	sessions := []domain.PlaySession{
		{ID: "s1", GameID: "game-1", PlayedAt: now, Duration: 3600, UpdatedAt: now},
	}

	result, err := buildMetaSnapshot(game, sessions, "", "sha256_of_saves", "TestPC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// gameJSON のハッシュが MetaSnapshot の GameJSON フィールドと一致するか
	if hashBytes(result.GameJSON) != result.Snapshot.GameJSON {
		t.Error("GameJSON hash mismatch")
	}
	// sessionsJSON のハッシュが MetaSnapshot の SessionsJSON フィールドと一致するか
	if hashBytes(result.SessionsJSON) != result.Snapshot.SessionsJSON {
		t.Error("SessionsJSON hash mismatch")
	}
	// Saves ハッシュが引数と一致するか
	if result.Snapshot.Saves != "sha256_of_saves" {
		t.Errorf("expected Saves=%q, got %q", "sha256_of_saves", result.Snapshot.Saves)
	}
	// SnapshotBytes が MetaSnapshot の JSON と一致するか
	want, _ := json.Marshal(result.Snapshot)
	if string(result.SnapshotBytes) != string(want) {
		t.Error("SnapshotBytes does not match marshaled Snapshot")
	}
}

func TestBuildMetaSnapshotImageHashOmittedWhenEmpty(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	game := domain.Game{ID: "g1", Title: "T", PlayStatus: domain.PlayStatusUnplayed, CreatedAt: now, UpdatedAt: now}

	result, err := buildMetaSnapshot(game, nil, "", "savehash", "PC")
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(result.GameJSON, &parsed); err != nil {
		t.Fatal(err)
	}
	if _, ok := parsed["imageHash"]; ok {
		t.Error("imageHash should be omitted when empty")
	}
}
