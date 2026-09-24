package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"CloudLaunch_Go/internal/domain"
)

// injectingFileOps は rename 失敗などを決定的に注入する。
type injectingFileOps struct {
	pullFileOps
	renameFailOn int // 1-based。この回数目の Rename で失敗する。0 なら失敗しない。
	renameCalls  int
	renameErr    error
}

func (f *injectingFileOps) Rename(oldpath, newpath string) error {
	f.renameCalls++
	if f.renameFailOn > 0 && f.renameCalls == f.renameFailOn {
		if f.renameErr != nil {
			return f.renameErr
		}
		return errors.New("injected rename failure: file locked")
	}
	return f.pullFileOps.Rename(oldpath, newpath)
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return string(b)
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestContentSyncServicePullStagingLeavesLiveOnStageFailure(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	saveDir := filepath.Join(parent, "saves")
	mustWrite(t, filepath.Join(saveDir, "save.dat"), "local-original")

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, nil, saveDir)
	mustWrite(t, filepath.Join(saveDir, "save.dat"), "local-original")

	bstore.downloadErr = errors.New("injected download failure")
	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	if _, err := svc.Pull(context.Background(), game.ID, nil, false); err == nil {
		t.Fatal("expected Pull to fail on stage download")
	}
	if got := readFileString(t, filepath.Join(saveDir, "save.dat")); got != "local-original" {
		t.Fatalf("live must be unchanged, got %q", got)
	}
	if repo.upsertedGame != nil || repo.localSyncHeadSet != "" {
		t.Fatal("DB must not be updated on stage failure")
	}
	if len(repo.pullOps) != 0 {
		t.Fatalf("journal must not be written before successful stage, got %#v", repo.pullOps)
	}
	entries, _ := os.ReadDir(parent)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".cloudlaunch-stage-") || strings.HasPrefix(e.Name(), ".cloudlaunch-backup-") {
			t.Fatalf("leftover staging dir: %s", e.Name())
		}
	}
}

func TestContentSyncServicePullRenameFailureDoesNotUpdateDB(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	saveDir := filepath.Join(parent, "saves")
	mustWrite(t, filepath.Join(saveDir, "save.dat"), "remote-data")

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, nil, saveDir)
	mustWrite(t, filepath.Join(saveDir, "save.dat"), "local-old")

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)
	svc.fileOps = &injectingFileOps{
		pullFileOps:  osPullFileOps{},
		renameFailOn: 1, // live→backup を失敗させる（Windows ファイルロック相当）
	}

	if _, err := svc.Pull(context.Background(), game.ID, nil, false); err == nil {
		t.Fatal("expected Pull to fail on rename")
	}
	if got := readFileString(t, filepath.Join(saveDir, "save.dat")); got != "local-old" {
		t.Fatalf("live must remain old content, got %q", got)
	}
	if repo.upsertedGame != nil || repo.localSyncHeadSet != "" {
		t.Fatal("DB must not update when rename fails")
	}
	if len(repo.pullOps) != 0 {
		t.Fatalf("failed rename should clear PREPARED journal, got %#v", repo.pullOps)
	}
}

func TestContentSyncServicePullDBFailureRestoresBackup(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	mustWrite(t, filepath.Join(saveDir, "save.dat"), "remote-data")

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, nil, saveDir)
	mustWrite(t, filepath.Join(saveDir, "save.dat"), "local-old")

	repo := newFakeRepo(&game, nil)
	// setupRemoteState は v2 commit を書くので ApplyPullResultV2 側へ注入する。
	repo.applyPullV2Err = errors.New("injected ApplyPullResultV2 failure")
	svc := newTestService(repo, bstore)

	if _, err := svc.Pull(context.Background(), game.ID, nil, false); err == nil {
		t.Fatal("expected Pull to fail on DB apply")
	}
	if got := readFileString(t, filepath.Join(saveDir, "save.dat")); got != "local-old" {
		t.Fatalf("live must be restored from backup, got %q", got)
	}
	if repo.localSyncHeadSet != "" {
		t.Fatal("baseline must not stick after DB failure")
	}
	if len(repo.pullOps) != 0 {
		t.Fatalf("journal should be cleared after DB failure restore, got %#v", repo.pullOps)
	}
}

func TestContentSyncServicePullAbortsSwapOnExternalModification(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	mustWrite(t, filepath.Join(saveDir, "save.dat"), "remote-data")

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, nil, saveDir)
	mustWrite(t, filepath.Join(saveDir, "save.dat"), "local-old")

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)
	bstore.onDownloadBlobs = func() {
		mustWrite(t, filepath.Join(saveDir, "save.dat"), "game-wrote-during-pull")
	}

	_, err := svc.Pull(context.Background(), game.ID, nil, false)
	if !errors.Is(err, ErrSaveDirChangedDuringPull) {
		t.Fatalf("err = %v, want ErrSaveDirChangedDuringPull", err)
	}
	if got := readFileString(t, filepath.Join(saveDir, "save.dat")); got != "game-wrote-during-pull" {
		t.Fatalf("external write must remain (no swap), got %q", got)
	}
	if repo.upsertedGame != nil || len(repo.pullOps) != 0 {
		t.Fatal("no journal/DB side effects on external modification")
	}
}

func TestContentSyncServicePullNoSaveFolderIsNoOpForSaves(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	mustWrite(t, filepath.Join(tmp, "save.dat"), "remote")
	game := baseGame(tmp)
	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, nil, tmp)
	// ローカル設定だけセーブフォルダ未設定にする（リモート構築には一時 dir を使う）。
	game.SaveFolderPath = nil

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	res, err := svc.Pull(context.Background(), game.ID, nil, false)
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if !res.Applied {
		t.Fatal("expected Applied=true even without save folder")
	}
	if len(bstore.downloadedBlobs) != 0 {
		t.Fatalf("no save download expected, got %#v", bstore.downloadedBlobs)
	}
	if repo.lastPullOpID != "" {
		t.Fatalf("no pull journal expected, got %q", repo.lastPullOpID)
	}
}

func TestContentSyncServiceRecoverPullPreparedBeforeRename(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	saveDir := filepath.Join(parent, "saves")
	mustWrite(t, filepath.Join(saveDir, "save.dat"), "old-live")
	opID := "op-before-rename"
	stagePath := filepath.Join(parent, ".cloudlaunch-stage-"+opID)
	backupPath := filepath.Join(parent, ".cloudlaunch-backup-"+opID)
	mustWrite(t, filepath.Join(stagePath, "save.dat"), "staged-new")

	game := baseGame(saveDir)
	repo := newFakeRepo(&game, nil)
	repo.pullOps[opID] = domain.PullOperation{
		OperationID: opID,
		GameID:      game.ID,
		LivePath:    saveDir,
		StagePath:   stagePath,
		BackupPath:  backupPath,
		CommitHash:  "commit",
		Status:      domain.PullOperationPrepared,
		HadLive:     true,
	}
	svc := newTestService(repo, newFakeBlobStore())

	if err := svc.RecoverPullOperations(context.Background()); err != nil {
		t.Fatalf("RecoverPullOperations: %v", err)
	}
	if got := readFileString(t, filepath.Join(saveDir, "save.dat")); got != "old-live" {
		t.Fatalf("live must stay, got %q", got)
	}
	if _, err := os.Stat(stagePath); !os.IsNotExist(err) {
		t.Fatal("stage must be removed")
	}
	if len(repo.pullOps) != 0 {
		t.Fatalf("journal must be cleared, got %#v", repo.pullOps)
	}
}

func TestContentSyncServiceRecoverPullPreparedBetweenRenames(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	saveDir := filepath.Join(parent, "saves")
	opID := "op-mid-rename"
	stagePath := filepath.Join(parent, ".cloudlaunch-stage-"+opID)
	backupPath := filepath.Join(parent, ".cloudlaunch-backup-"+opID)
	mustWrite(t, filepath.Join(backupPath, "save.dat"), "old-live")
	mustWrite(t, filepath.Join(stagePath, "save.dat"), "staged-new")
	// live パスは空（live→backup 済み、stage→live 前）

	game := baseGame(saveDir)
	repo := newFakeRepo(&game, nil)
	repo.pullOps[opID] = domain.PullOperation{
		OperationID: opID,
		GameID:      game.ID,
		LivePath:    saveDir,
		StagePath:   stagePath,
		BackupPath:  backupPath,
		CommitHash:  "commit",
		Status:      domain.PullOperationPrepared,
		HadLive:     true,
	}
	svc := newTestService(repo, newFakeBlobStore())

	if err := svc.RecoverPullOperations(context.Background()); err != nil {
		t.Fatalf("RecoverPullOperations: %v", err)
	}
	if got := readFileString(t, filepath.Join(saveDir, "save.dat")); got != "old-live" {
		t.Fatalf("backup must become live, got %q", got)
	}
	if _, err := os.Stat(stagePath); !os.IsNotExist(err) {
		t.Fatal("stage must be removed")
	}
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Fatal("backup path must be gone after rename")
	}
}

func TestContentSyncServiceRecoverPullPreparedAfterSwapBeforeDB(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	saveDir := filepath.Join(parent, "saves")
	opID := "op-after-swap"
	stagePath := filepath.Join(parent, ".cloudlaunch-stage-"+opID)
	backupPath := filepath.Join(parent, ".cloudlaunch-backup-"+opID)
	mustWrite(t, filepath.Join(saveDir, "save.dat"), "new-live")
	mustWrite(t, filepath.Join(backupPath, "save.dat"), "old-live")

	game := baseGame(saveDir)
	repo := newFakeRepo(&game, nil)
	repo.pullOps[opID] = domain.PullOperation{
		OperationID: opID,
		GameID:      game.ID,
		LivePath:    saveDir,
		StagePath:   stagePath,
		BackupPath:  backupPath,
		CommitHash:  "commit",
		Status:      domain.PullOperationPrepared,
		HadLive:     true,
	}
	svc := newTestService(repo, newFakeBlobStore())

	if err := svc.RecoverPullOperations(context.Background()); err != nil {
		t.Fatalf("RecoverPullOperations: %v", err)
	}
	if got := readFileString(t, filepath.Join(saveDir, "save.dat")); got != "old-live" {
		t.Fatalf("must restore old live, got %q", got)
	}
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Fatal("backup must be consumed")
	}
}

func TestContentSyncServiceRecoverPullAppliedRemovesBackup(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	saveDir := filepath.Join(parent, "saves")
	opID := "op-applied"
	stagePath := filepath.Join(parent, ".cloudlaunch-stage-"+opID)
	backupPath := filepath.Join(parent, ".cloudlaunch-backup-"+opID)
	mustWrite(t, filepath.Join(saveDir, "save.dat"), "new-live")
	mustWrite(t, filepath.Join(backupPath, "save.dat"), "old-live")

	game := baseGame(saveDir)
	repo := newFakeRepo(&game, nil)
	repo.pullOps[opID] = domain.PullOperation{
		OperationID: opID,
		GameID:      game.ID,
		LivePath:    saveDir,
		StagePath:   stagePath,
		BackupPath:  backupPath,
		CommitHash:  "commit",
		Status:      domain.PullOperationApplied,
		HadLive:     true,
	}
	svc := newTestService(repo, newFakeBlobStore())

	if err := svc.RecoverPullOperations(context.Background()); err != nil {
		t.Fatalf("RecoverPullOperations: %v", err)
	}
	if got := readFileString(t, filepath.Join(saveDir, "save.dat")); got != "new-live" {
		t.Fatalf("applied recovery must keep new live, got %q", got)
	}
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Fatal("backup must be removed after APPLIED recovery")
	}
	if len(repo.pullOps) != 0 {
		t.Fatalf("journal must be cleared, got %#v", repo.pullOps)
	}
}

func TestContentSyncServicePullSuccessClearsJournalAndBackup(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	saveDir := filepath.Join(parent, "saves")
	mustWrite(t, filepath.Join(saveDir, "save.dat"), "remote-data")

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, nil, saveDir)
	mustWrite(t, filepath.Join(saveDir, "save.dat"), "local-old")

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	res, err := svc.Pull(context.Background(), game.ID, nil, false)
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if !res.Applied {
		t.Fatal("expected Applied")
	}
	if got := readFileString(t, filepath.Join(saveDir, "save.dat")); got != "remote-data" {
		t.Fatalf("live content = %q, want remote-data", got)
	}
	if len(repo.pullOps) != 0 {
		t.Fatalf("journal should be cleared on success, got %#v", repo.pullOps)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".cloudlaunch-") {
			t.Fatalf("leftover temp dir after success: %s", e.Name())
		}
	}
}

func TestContentSyncServicePullUntrackedConfirmHasNoStagingSideEffects(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	saveDir := filepath.Join(parent, "saves")
	mustWrite(t, filepath.Join(saveDir, "save.dat"), "remote data")

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, nil, saveDir)
	mustWrite(t, filepath.Join(saveDir, "user_notes.txt"), "personal")

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	res, err := svc.Pull(context.Background(), game.ID, nil, false)
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if res.Applied {
		t.Fatal("confirmation required")
	}
	if len(repo.pullOps) != 0 {
		t.Fatal("no journal before confirmation")
	}
	entries, _ := os.ReadDir(parent)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".cloudlaunch-") {
			t.Fatalf("no staging dirs before confirmation: %s", e.Name())
		}
	}
	if _, err := os.Stat(filepath.Join(saveDir, "user_notes.txt")); err != nil {
		t.Fatalf("untracked must remain: %v", err)
	}
}

func TestInjectingFileOpsCountsRenames(t *testing.T) {
	t.Parallel()
	ops := &injectingFileOps{pullFileOps: osPullFileOps{}, renameFailOn: 2, renameErr: fmt.Errorf("lock")}
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	c := filepath.Join(dir, "c")
	if err := os.Mkdir(a, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := ops.Rename(a, b); err != nil {
		t.Fatalf("first rename should succeed: %v", err)
	}
	if err := ops.Rename(b, c); err == nil {
		t.Fatal("second rename should fail")
	}
}

// 起動時 Recover が失敗／未実行のまま再 Pull しても、古い PREPARED を先に消化してから
// 新規 stage を作ることを保証する（多重ジャーナルによる誤ロールバック防止）。
func TestContentSyncServicePullRecoversStalePreparedJournalBeforeStaging(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	saveDir := filepath.Join(parent, "saves")
	opID := "stale-prepared"
	stagePath := filepath.Join(parent, ".cloudlaunch-stage-"+opID)
	backupPath := filepath.Join(parent, ".cloudlaunch-backup-"+opID)

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	// リモート desired tree を先に固定してから、クラッシュ直後の live/backup を載せる。
	mustWrite(t, filepath.Join(saveDir, "save.dat"), "remote-data")
	setupRemoteState(t, bstore, game.ID, game, nil, saveDir)
	mustWrite(t, filepath.Join(saveDir, "save.dat"), "half-applied-new")
	mustWrite(t, filepath.Join(backupPath, "save.dat"), "pre-crash-old")

	repo := newFakeRepo(&game, nil)
	repo.pullOps[opID] = domain.PullOperation{
		OperationID: opID,
		GameID:      game.ID,
		LivePath:    saveDir,
		StagePath:   stagePath,
		BackupPath:  backupPath,
		CommitHash:  "old-commit",
		Status:      domain.PullOperationPrepared,
		HadLive:     true,
	}
	svc := newTestService(repo, bstore)

	res, err := svc.Pull(context.Background(), game.ID, nil, false)
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if !res.Applied {
		t.Fatal("expected Applied after recovering stale journal")
	}
	if got := readFileString(t, filepath.Join(saveDir, "save.dat")); got != "remote-data" {
		t.Fatalf("live = %q, want remote-data (fresh pull after rollback)", got)
	}
	if _, ok := repo.pullOps[opID]; ok {
		t.Fatal("stale PREPARED journal must be cleared")
	}
	if len(repo.pullOps) != 0 {
		t.Fatalf("no leftover journals expected, got %#v", repo.pullOps)
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".cloudlaunch-") {
			t.Fatalf("leftover temp dir: %s", e.Name())
		}
	}
}
