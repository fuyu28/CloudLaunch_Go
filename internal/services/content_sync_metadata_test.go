package services

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"CloudLaunch_Go/internal/domain"
)

func writeSaveFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func testSession(id string) domain.PlaySession {
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return domain.PlaySession{ID: id, GameID: "game-1", PlayedAt: at, Duration: 600, UpdatedAt: at}
}

// localBaselineFingerprint は現在のセーブフォルダと sessions を「最後に同期した状態」とみなした fingerprint を返す。
func localBaselineFingerprint(t *testing.T, game domain.Game, sessions []domain.PlaySession, saveDir string) string {
	t.Helper()
	snap, _, err := buildSaveSnapshot(saveDir)
	if err != nil {
		t.Fatal(err)
	}
	snapJSON, _ := json.Marshal(snap)
	meta, err := buildMetaSnapshot(game, sessions, nil, "", hashBytes(snapJSON), "testdevice", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	return contentFingerprint(meta.Snapshot)
}

func TestSyncMetadataSkipsPushWhenLocalSavesChanged(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	writeSaveFile(t, saveDir, "save.dat", "original")
	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	remoteMeta := setupRemoteState(t, bstore, game.ID, game, nil, saveDir)
	fp := contentFingerprint(remoteMeta)
	game.LocalSyncHead = &fp
	headBefore := bstore.heads[game.ID]

	writeSaveFile(t, saveDir, "save.dat", "played")
	repo := newFakeRepo(&game, []domain.PlaySession{testSession("s1")})
	svc := newTestService(repo, bstore)

	err := svc.SyncMetadata(context.Background(), game.ID)
	if !errors.Is(err, ErrSavesDiffer) {
		t.Fatalf("SyncMetadata err = %v, want ErrSavesDiffer", err)
	}
	if bstore.heads[game.ID] != headBefore {
		t.Fatal("remote HEAD must not change when saves differ")
	}
	if repo.localSyncHeadSet != "" {
		t.Fatal("local baseline must not change when saves differ")
	}
}

func TestSyncMetadataPushesWhenOnlySessionsChanged(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	writeSaveFile(t, saveDir, "save.dat", "unchanged")
	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	remoteMeta := setupRemoteState(t, bstore, game.ID, game, nil, saveDir)
	fp := contentFingerprint(remoteMeta)
	game.LocalSyncHead = &fp
	headBefore := bstore.heads[game.ID]

	repo := newFakeRepo(&game, []domain.PlaySession{testSession("s1")})
	svc := newTestService(repo, bstore)

	if err := svc.SyncMetadata(context.Background(), game.ID); err != nil {
		t.Fatalf("SyncMetadata: %v", err)
	}
	if bstore.heads[game.ID] == headBefore {
		t.Fatal("remote HEAD should advance for metadata-only change")
	}
	detail, err := svc.Status(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if detail.Status != domain.SyncStatusInSync {
		t.Fatalf("Status after SyncMetadata = %q, want in_sync", detail.Status)
	}
}

func TestSyncMetadataDoesNotUploadWhenNeverSynced(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	writeSaveFile(t, saveDir, "save.dat", "first")
	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	svc := newTestService(newFakeRepo(&game, nil), bstore)

	err := svc.SyncMetadata(context.Background(), game.ID)
	if !errors.Is(err, ErrSavesDiffer) {
		t.Fatalf("SyncMetadata err = %v, want ErrSavesDiffer", err)
	}
	if bstore.heads[game.ID] != "" {
		t.Fatal("first upload must wait for user confirmation")
	}
}

func TestSyncMetadataPullsSessionsWithoutTouchingSaves(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	writeSaveFile(t, saveDir, "save.dat", "same")
	game := baseGame(saveDir)
	localFP := localBaselineFingerprint(t, game, nil, saveDir)
	game.LocalSyncHead = &localFP

	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, []domain.PlaySession{testSession("remote-s1")}, saveDir)

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	if err := svc.SyncMetadata(context.Background(), game.ID); err != nil {
		t.Fatalf("SyncMetadata: %v", err)
	}
	if len(repo.upsertedSessions) != 1 || repo.upsertedSessions[0].ID != "remote-s1" {
		t.Fatalf("remote sessions not applied: %+v", repo.upsertedSessions)
	}
	if repo.lastPullOpID != "" {
		t.Fatal("metadata-only pull must not stage/swap the save folder")
	}
	if len(repo.pullOps) != 0 {
		t.Fatal("metadata-only pull must not leave pull journal")
	}
}

func TestSyncMetadataSkipsPullWhenRemoteSavesChanged(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	writeSaveFile(t, saveDir, "save.dat", "local")
	game := baseGame(saveDir)
	localFP := localBaselineFingerprint(t, game, nil, saveDir)
	game.LocalSyncHead = &localFP

	remoteDir := t.TempDir()
	writeSaveFile(t, remoteDir, "save.dat", "remote progress")
	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, nil, remoteDir)

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	err := svc.SyncMetadata(context.Background(), game.ID)
	if !errors.Is(err, ErrSavesDiffer) {
		t.Fatalf("SyncMetadata err = %v, want ErrSavesDiffer", err)
	}
	if repo.upsertedGame != nil {
		t.Fatal("DB must not be updated when remote saves differ")
	}
	data, err := os.ReadFile(filepath.Join(saveDir, "save.dat"))
	if err != nil || string(data) != "local" {
		t.Fatalf("local save must stay untouched: %q err=%v", data, err)
	}
}

func TestMetadataOnlyPushAbortsWhenSavesChangeAfterStatus(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	writeSaveFile(t, saveDir, "save.dat", "unchanged")
	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	remoteMeta := setupRemoteState(t, bstore, game.ID, game, nil, saveDir)
	fp := contentFingerprint(remoteMeta)
	game.LocalSyncHead = &fp
	headBefore := bstore.heads[game.ID]

	// Status 判定の後でゲームがセーブを書き換えたケースを、push を直接呼んで再現する。
	writeSaveFile(t, saveDir, "save.dat", "written during sync")
	svc := newTestService(newFakeRepo(&game, nil), bstore)

	err := svc.push(context.Background(), game.ID, nil, false, true)
	if !errors.Is(err, ErrSavesDiffer) {
		t.Fatalf("push err = %v, want ErrSavesDiffer", err)
	}
	if bstore.heads[game.ID] != headBefore {
		t.Fatal("remote HEAD must not change")
	}
}

func TestStatusTreatsMissingSaveFolderAsEmptyForDownloadPrompt(t *testing.T) {
	t.Parallel()

	remoteDir := t.TempDir()
	writeSaveFile(t, remoteDir, "save.dat", "cloud save")
	game := baseGame(filepath.Join(t.TempDir(), "not-created-yet"))
	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, nil, remoteDir)

	svc := newTestService(newFakeRepo(&game, nil), bstore)
	detail, err := svc.Status(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if detail.Status != domain.SyncStatusPullNeeded || !detail.SavesDiffer {
		t.Fatalf("Status = %q savesDiffer=%v, want pull_needed with savesDiffer", detail.Status, detail.SavesDiffer)
	}
}
