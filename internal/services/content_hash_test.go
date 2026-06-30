package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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
	// "hello world" の SHA-256
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
	// base tree に stale.sav を含めると tracked 削除に分類される。
	baseTree := map[string]struct{}{
		"nested/keep.sav":  {},
		"nested/stale.sav": {},
	}
	tracked, untracked, err := planDeletions(dir, snapshot, baseTree)
	if err != nil {
		t.Fatalf("planDeletions: %v", err)
	}
	if len(tracked) != 1 || tracked[0] != "nested/stale.sav" {
		t.Fatalf("tracked should be [nested/stale.sav], got %v", tracked)
	}
	if len(untracked) != 0 {
		t.Fatalf("untracked should be empty, got %v", untracked)
	}
	if err := applyDeletions(dir, tracked); err != nil {
		t.Fatalf("applyDeletions: %v", err)
	}
	if _, err := os.Stat(keepPath); err != nil {
		t.Fatalf("keep file should remain: %v", err)
	}
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("stale file should be removed, stat err: %v", err)
	}
}

// TestPlanDeletionsClassifiesUntracked は base tree に無いファイルが untracked に
// 分類され、planDeletions 自体はファイルを削除しないことを確認する。
func TestPlanDeletionsClassifiesUntracked(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	unknownPath := filepath.Join(dir, "unrelated.txt")
	if err := os.WriteFile(unknownPath, []byte("user file"), 0o600); err != nil {
		t.Fatal(err)
	}

	// 新スナップショットにも base tree にも無いファイル → untracked
	tracked, untracked, err := planDeletions(dir, domain.SaveSnapshot{Files: map[string]domain.BlobHash{}}, map[string]struct{}{})
	if err != nil {
		t.Fatalf("planDeletions: %v", err)
	}
	if len(tracked) != 0 {
		t.Fatalf("tracked should be empty, got %v", tracked)
	}
	if len(untracked) != 1 || untracked[0] != "unrelated.txt" {
		t.Fatalf("untracked should be [unrelated.txt], got %v", untracked)
	}
	// planDeletions は削除しない
	if _, err := os.Stat(unknownPath); err != nil {
		t.Fatalf("planDeletions must not delete files: %v", err)
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

	// stale/only.sav を tracked 削除として削除 → 空になった stale/ も除去される
	if err := applyDeletions(dir, []string{"stale/only.sav"}); err != nil {
		t.Fatalf("applyDeletions: %v", err)
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

	result, err := buildMetaSnapshot(game, sessions, "", "sha256_of_saves", "TestPC", 0, 0)
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

// TestApplyDeletionsPreservesUnrelatedEmptyDir は、削除したファイルの祖先でない
// 「元から空のディレクトリ」を applyDeletions が消さないことを確認する。
func TestApplyDeletionsPreservesUnrelatedEmptyDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// 削除対象のファイルとその親
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "old.sav"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// 削除と無関係な、ユーザーが置いた空ディレクトリ
	userEmpty := filepath.Join(dir, "userempty")
	if err := os.MkdirAll(userEmpty, 0o700); err != nil {
		t.Fatal(err)
	}

	if err := applyDeletions(dir, []string{"data/old.sav"}); err != nil {
		t.Fatalf("applyDeletions: %v", err)
	}

	// 削除ファイルの親 data/ は空になったので消える
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Fatalf("emptied data dir should be removed, stat err: %v", err)
	}
	// 無関係な空ディレクトリは残る
	if _, err := os.Stat(userEmpty); err != nil {
		t.Fatalf("unrelated empty dir should be preserved: %v", err)
	}
}

// TestApplyDeletionsNoopOnEmptyInput は relPaths が空のとき何も走査・削除しないことを確認する。
func TestApplyDeletionsNoopOnEmptyInput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	emptyDir := filepath.Join(dir, "keepme")
	if err := os.MkdirAll(emptyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := applyDeletions(dir, nil); err != nil {
		t.Fatalf("applyDeletions(nil): %v", err)
	}
	if _, err := os.Stat(emptyDir); err != nil {
		t.Fatalf("empty input must not prune any dir: %v", err)
	}
}

func TestBuildMetaSnapshotImageHashOmittedWhenEmpty(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	game := domain.Game{ID: "g1", Title: "T", PlayStatus: domain.PlayStatusUnplayed, CreatedAt: now, UpdatedAt: now}

	result, err := buildMetaSnapshot(game, nil, "", "savehash", "PC", 0, 0)
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

// TestHashFileStreamMatchesHashBytes は hashFileStream が hashBytes と同じハッシュを返すことを確認する。
func TestHashFileStreamMatchesHashBytes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "f.bin")
	content := []byte("streaming hash content 0123456789")
	if err := os.WriteFile(p, content, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := hashFileStream(p)
	if err != nil {
		t.Fatalf("hashFileStream: %v", err)
	}
	if want := hashBytes(content); got != want {
		t.Fatalf("hash mismatch: got %s want %s", got, want)
	}
}

// TestBuildSaveTreeMatchesSnapshot は buildSaveTree が buildSaveSnapshot と同じ
// パス→ハッシュ集合を返すことを確認する。
func TestBuildSaveTreeMatchesSnapshot(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.sav"), []byte("aaa"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nested", "b.sav"), []byte("bbb"), 0o600); err != nil {
		t.Fatal(err)
	}

	tree, err := buildSaveTree(dir)
	if err != nil {
		t.Fatalf("buildSaveTree: %v", err)
	}
	snap, _, err := buildSaveSnapshot(dir)
	if err != nil {
		t.Fatalf("buildSaveSnapshot: %v", err)
	}
	if len(tree.Files) != len(snap.Files) {
		t.Fatalf("file count mismatch: tree=%d snap=%d", len(tree.Files), len(snap.Files))
	}
	for k, v := range snap.Files {
		if tree.Files[k] != v {
			t.Fatalf("hash mismatch for %s: tree=%s snap=%s", k, tree.Files[k], v)
		}
	}
}

// TestBuildSaveTreeRejectsMissingDir は存在しないディレクトリでエラーを返すことを確認する。
func TestBuildSaveTreeRejectsMissingDir(t *testing.T) {
	t.Parallel()
	if _, err := buildSaveTree(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error for missing dir")
	}
}

// TestBuildSaveTreeIgnoresSymlinks は、セーブフォルダ内のシンボリックリンクが
// スナップショットに含まれない（リンク先実体が読まれない）ことを確認する。
func TestBuildSaveTreeIgnoresSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink 権限の都合で Windows ではスキップ")
	}
	t.Parallel()

	// リンク先となる「外部の機密ファイル」
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("TOP SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "real.sav"), []byte("real"), 0o600); err != nil {
		t.Fatal(err)
	}
	// セーブフォルダ内に外部機密ファイルへのシンボリックリンクを置く
	if err := os.Symlink(secret, filepath.Join(saveDir, "link.txt")); err != nil {
		t.Skipf("symlink を作成できない環境: %v", err)
	}

	tree, err := buildSaveTree(saveDir)
	if err != nil {
		t.Fatalf("buildSaveTree: %v", err)
	}
	if _, ok := tree.Files["link.txt"]; ok {
		t.Fatal("symlink はスナップショットに含めてはならない")
	}
	if _, ok := tree.Files["real.sav"]; !ok {
		t.Fatal("通常ファイルは含まれるべき")
	}

	// buildSaveSnapshot 側も同様にリンクを無視すること
	snap, blobs, err := buildSaveSnapshot(saveDir)
	if err != nil {
		t.Fatalf("buildSaveSnapshot: %v", err)
	}
	if _, ok := snap.Files["link.txt"]; ok {
		t.Fatal("buildSaveSnapshot も symlink を含めてはならない")
	}
	// リンク先の機密内容が blob 化されていないこと
	for _, data := range blobs {
		if string(data) == "TOP SECRET" {
			t.Fatal("リンク先の機密内容が blob 化されている（情報漏洩）")
		}
	}
}

// TestBuildSaveTreeFollowsSymlinkedRoot は saveFolderPath 自体がディレクトリへの
// シンボリックリンクである場合でも、配下の通常ファイルがちゃんとスナップショットに
// 入ることを確認する。これが崩れると Push が空の SaveSnapshot をアップロードして
// 他端末で Pull した時に全セーブが消える。
func TestBuildSaveTreeFollowsSymlinkedRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink 権限の都合で Windows ではスキップ")
	}
	t.Parallel()

	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "save.dat"), []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "sub", "deep.dat"), []byte("nested"), 0o600); err != nil {
		t.Fatal(err)
	}

	linkParent := t.TempDir()
	linkPath := filepath.Join(linkParent, "saves")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Skipf("symlink を作成できない環境: %v", err)
	}

	tree, err := buildSaveTree(linkPath)
	if err != nil {
		t.Fatalf("buildSaveTree: %v", err)
	}
	if _, ok := tree.Files["save.dat"]; !ok {
		t.Fatal("symlink root 配下の通常ファイルがスナップショットから抜けている")
	}
	if _, ok := tree.Files["sub/deep.dat"]; !ok {
		t.Fatal("symlink root 配下のサブディレクトリ内ファイルが抜けている")
	}

	snap, blobs, err := buildSaveSnapshot(linkPath)
	if err != nil {
		t.Fatalf("buildSaveSnapshot: %v", err)
	}
	if len(snap.Files) != 2 {
		t.Fatalf("buildSaveSnapshot の Files = %d, want 2", len(snap.Files))
	}
	if len(blobs) != 2 {
		t.Fatalf("buildSaveSnapshot の blobs = %d, want 2", len(blobs))
	}
}
