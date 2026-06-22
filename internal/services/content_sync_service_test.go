package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/infrastructure/storage"
)

// ─── fakeContentSyncRepository ──────────────────────────────────────────────

type fakeContentSyncRepository struct {
	mu sync.Mutex

	game      *domain.Game
	sessions  []domain.PlaySession
	settings  map[string]string
	saveTree  string

	// 記録された呼び出し
	localSyncHeadSet string
	saveTreeSet      string
	upsertedGame     *domain.Game
	deletedSessions  bool
	upsertedSessions []domain.PlaySession

	// エラー注入
	getGameErr error
}

func newFakeRepo(game *domain.Game, sessions []domain.PlaySession) *fakeContentSyncRepository {
	return &fakeContentSyncRepository{
		game:     game,
		sessions: sessions,
		settings: make(map[string]string),
	}
}

func (r *fakeContentSyncRepository) GetGameByID(_ context.Context, _ string) (*domain.Game, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.getGameErr != nil {
		return nil, r.getGameErr
	}
	return r.game, nil
}

func (r *fakeContentSyncRepository) ListPlaySessionsByGame(_ context.Context, _ string) ([]domain.PlaySession, error) {
	return r.sessions, nil
}

func (r *fakeContentSyncRepository) SetLocalSyncHead(_ context.Context, _ string, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.localSyncHeadSet = hash
	if r.game != nil {
		r.game.LocalSyncHead = &hash
	}
	return nil
}

func (r *fakeContentSyncRepository) GetLocalSaveTree(_ context.Context, _ string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saveTree, nil
}

func (r *fakeContentSyncRepository) SetLocalSaveTree(_ context.Context, _ string, tree string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saveTree = tree
	r.saveTreeSet = tree
	return nil
}

func (r *fakeContentSyncRepository) UpsertGameSync(_ context.Context, game domain.Game) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.upsertedGame = &game
	return nil
}

func (r *fakeContentSyncRepository) DeletePlaySessionsByGame(_ context.Context, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deletedSessions = true
	return nil
}

func (r *fakeContentSyncRepository) UpsertPlaySessionSync(_ context.Context, session domain.PlaySession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.upsertedSessions = append(r.upsertedSessions, session)
	return nil
}

func (r *fakeContentSyncRepository) ApplyPullResult(
	_ context.Context,
	game domain.Game,
	sessions []domain.PlaySession,
	syncHead, saveTree string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.upsertedGame = &game
	r.deletedSessions = true
	r.upsertedSessions = append([]domain.PlaySession{}, sessions...)
	r.localSyncHeadSet = syncHead
	if r.game != nil {
		r.game.LocalSyncHead = &syncHead
	}
	r.saveTree = saveTree
	r.saveTreeSet = saveTree
	return nil
}

func (r *fakeContentSyncRepository) GetSetting(_ context.Context, key string) (string, error) {
	return r.settings[key], nil
}

func (r *fakeContentSyncRepository) UpsertSetting(_ context.Context, key, value string) error {
	r.settings[key] = value
	return nil
}

// ─── fakeBlobStore ───────────────────────────────────────────────────────────

type fakeBlobStore struct {
	mu sync.Mutex

	blobs map[string][]byte // キー: "gameID/hash"
	heads map[string]string // gameID → metaHash

	// 記録された呼び出し
	downloadedBlobs []map[string]string // 各呼び出しの blobs 引数
	deletedPrefixes []string
}

func newFakeBlobStore() *fakeBlobStore {
	return &fakeBlobStore{
		blobs: make(map[string][]byte),
		heads: make(map[string]string),
	}
}

func (f *fakeBlobStore) blobKey(gameID, kind, hash string) string {
	return gameID + "/" + kind + "/" + hash
}

func (f *fakeBlobStore) readHEAD(_ context.Context, gameID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.heads[gameID], nil
}

func (f *fakeBlobStore) writeHEAD(_ context.Context, gameID, hash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.heads[gameID] = hash
	return nil
}

func (f *fakeBlobStore) getBlob(_ context.Context, gameID, kind, hash string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.blobs[f.blobKey(gameID, kind, hash)]
	if !ok {
		return nil, fmt.Errorf("blob not found: %s/%s/%s", gameID, kind, hash)
	}
	return data, nil
}

func (f *fakeBlobStore) putBlob(_ context.Context, gameID, kind, hash string, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.blobs[f.blobKey(gameID, kind, hash)] = data
	return nil
}

func (f *fakeBlobStore) putBlobs(_ context.Context, gameID string, blobs map[string][]byte, _ int, onProgress func(int, int)) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	total := len(blobs)
	done := 0
	for hash, data := range blobs {
		f.blobs[f.blobKey(gameID, storage.BlobKindObject, hash)] = data
		done++
		if onProgress != nil {
			onProgress(done, total)
		}
	}
	return nil
}

func (f *fakeBlobStore) downloadBlobs(_ context.Context, gameID, saveDir string, blobs map[string]string, _ int, onProgress func(int, int)) error {
	// 呼び出しを記録する
	snapshot := make(map[string]string, len(blobs))
	for k, v := range blobs {
		snapshot[k] = v
	}
	f.mu.Lock()
	f.downloadedBlobs = append(f.downloadedBlobs, snapshot)
	f.mu.Unlock()

	total := len(blobs)
	done := 0
	for relPath, hash := range blobs {
		f.mu.Lock()
		data, ok := f.blobs[f.blobKey(gameID, storage.BlobKindObject, hash)]
		f.mu.Unlock()
		if !ok {
			return fmt.Errorf("blob not found: %s/%s/%s", gameID, storage.BlobKindObject, hash)
		}
		targetPath, err := storage.ResolveSafeRelativePath(saveDir, relPath)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(targetPath, data, 0o600); err != nil {
			return err
		}
		done++
		if onProgress != nil {
			onProgress(done, total)
		}
	}
	return nil
}

func (f *fakeBlobStore) deleteByPrefix(_ context.Context, prefix string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deletedPrefixes = append(f.deletedPrefixes, prefix)
	return nil
}

func (f *fakeBlobStore) listGameIDs(_ context.Context) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var ids []string
	for gameID := range f.heads {
		ids = append(ids, gameID)
	}
	return ids, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newTestService(repo *fakeContentSyncRepository, bstore *fakeBlobStore) *ContentSyncService {
	svc := &ContentSyncService{
		config:     config.Config{S3UploadConcurrency: 2},
		repository: repo,
		logger:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}
	svc.newBlobStore = func(_ context.Context) (contentBlobStore, error) {
		return bstore, nil
	}
	return svc
}

func strPtr(s string) *string { return &s }

// setupRemoteState はゲームの現在のセーブフォルダ状態をリモートとして fakeBlobStore に書き込む。
// テストで「remote = 現在のローカル」という基準点を作るために使う。
func setupRemoteState(
	t *testing.T,
	bstore *fakeBlobStore,
	gameID string,
	game domain.Game,
	sessions []domain.PlaySession,
	saveDir string,
) domain.MetaSnapshot {
	t.Helper()
	ctx := context.Background()

	saveSnap, saveBlobs, err := buildSaveSnapshot(saveDir)
	if err != nil {
		t.Fatalf("buildSaveSnapshot: %v", err)
	}
	saveSnapJSON, _ := json.Marshal(saveSnap)
	savesHash := hashBytes(saveSnapJSON)

	for h, data := range saveBlobs {
		if err := bstore.putBlob(ctx, gameID, storage.BlobKindObject, h, data); err != nil {
			t.Fatalf("putBlob: %v", err)
		}
	}
	if err := bstore.putBlob(ctx, gameID, storage.BlobKindTree, savesHash, saveSnapJSON); err != nil {
		t.Fatalf("putBlob saveSnap: %v", err)
	}

	meta, err := buildMetaSnapshot(game, sessions, "", savesHash, "testdevice")
	if err != nil {
		t.Fatalf("buildMetaSnapshot: %v", err)
	}
	metaHash := hashBytes(meta.SnapshotBytes)

	if err := bstore.putBlob(ctx, gameID, storage.BlobKindMeta, meta.Snapshot.GameJSON, meta.GameJSON); err != nil {
		t.Fatalf("putBlob gameJSON: %v", err)
	}
	if err := bstore.putBlob(ctx, gameID, storage.BlobKindMeta, meta.Snapshot.SessionsJSON, meta.SessionsJSON); err != nil {
		t.Fatalf("putBlob sessionsJSON: %v", err)
	}
	if err := bstore.putBlob(ctx, gameID, storage.BlobKindCommit, metaHash, meta.SnapshotBytes); err != nil {
		t.Fatalf("putBlob meta: %v", err)
	}
	if err := bstore.writeHEAD(ctx, gameID, metaHash); err != nil {
		t.Fatalf("writeHEAD: %v", err)
	}

	return meta.Snapshot
}

func baseGame(saveDir string) domain.Game {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return domain.Game{
		ID:             "game-1",
		Title:          "Test Game",
		Publisher:      "Test Publisher",
		PlayStatus:     domain.PlayStatusUnplayed,
		SaveFolderPath: strPtr(saveDir),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// ─── Push tests ──────────────────────────────────────────────────────────────

func TestContentSyncServicePushUploadsDataAndSetsHead(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("game data"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	repo := newFakeRepo(&game, nil)
	bstore := newFakeBlobStore()
	svc := newTestService(repo, bstore)

	if err := svc.Push(context.Background(), game.ID, nil); err != nil {
		t.Fatalf("Push returned unexpected error: %v", err)
	}

	// HEAD が書き込まれた
	if bstore.heads[game.ID] == "" {
		t.Error("expected HEAD to be set")
	}
	// ローカル同期ヘッドが更新された
	if repo.localSyncHeadSet == "" {
		t.Error("expected SetLocalSyncHead to be called")
	}

	// セーブファイルのブロブが格納されている
	saveData := []byte("game data")
	saveHash := hashBytes(saveData)
	bstore.mu.Lock()
	_, ok := bstore.blobs[bstore.blobKey(game.ID, storage.BlobKindObject, saveHash)]
	bstore.mu.Unlock()
	if !ok {
		t.Error("expected save file blob to be stored")
	}
}

func TestContentSyncServicePushSetsProgressCallback(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	for i := range 3 {
		if err := os.WriteFile(filepath.Join(saveDir, fmt.Sprintf("save%d.dat", i)), []byte(fmt.Sprintf("data%d", i)), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	game := baseGame(saveDir)
	repo := newFakeRepo(&game, nil)
	bstore := newFakeBlobStore()
	svc := newTestService(repo, bstore)

	var progressCalled bool
	err := svc.Push(context.Background(), game.ID, func(current, total int) {
		progressCalled = true
	})
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if !progressCalled {
		t.Error("expected progress callback to be called")
	}
}

func TestContentSyncServicePushReturnsErrorWhenGameNotFound(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo(nil, nil) // game = nil
	bstore := newFakeBlobStore()
	svc := newTestService(repo, bstore)

	err := svc.Push(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error when game not found")
	}
}

func TestContentSyncServicePushReturnsErrorWhenSaveFolderNotSet(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	game := domain.Game{
		ID:         "game-1",
		PlayStatus: domain.PlayStatusUnplayed,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	repo := newFakeRepo(&game, nil)
	bstore := newFakeBlobStore()
	svc := newTestService(repo, bstore)

	err := svc.Push(context.Background(), game.ID, nil)
	if err == nil {
		t.Fatal("expected error when SaveFolderPath is nil")
	}
}

func TestContentSyncServicePushReturnsErrorWhenSaveDirMissing(t *testing.T) {
	t.Parallel()

	game := baseGame("/nonexistent/save/path/that/does/not/exist")
	repo := newFakeRepo(&game, nil)
	bstore := newFakeBlobStore()
	svc := newTestService(repo, bstore)

	err := svc.Push(context.Background(), game.ID, nil)
	if err == nil {
		t.Fatal("expected error when save directory does not exist")
	}
}

func TestContentSyncServicePushRejectsChangedRemoteHead(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("local"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	baseMeta := setupRemoteState(t, bstore, game.ID, game, nil, saveDir)
	baseFP := contentFingerprint(baseMeta)
	game.LocalSyncHead = &baseFP

	remoteDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(remoteDir, "save.dat"), []byte("remote changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	setupRemoteState(t, bstore, game.ID, game, nil, remoteDir)
	remoteHead := bstore.heads[game.ID]

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	err := svc.Push(context.Background(), game.ID, nil)
	if err == nil {
		t.Fatal("expected Push to reject changed remote head")
	}
	if bstore.heads[game.ID] != remoteHead {
		t.Fatal("remote HEAD should not be overwritten")
	}
}

// ─── Pull tests ───────────────────────────────────────────────────────────────

func TestContentSyncServicePullRestoresFilesAndMetadata(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("remote data"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	sessions := []domain.PlaySession{
		{
			ID:        "s1",
			GameID:    game.ID,
			PlayedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Duration:  3600,
			UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, sessions, saveDir)

	// ローカルのセーブファイルを別の状態にしてダウンロードが発生するようにする
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("old local data"), 0o600); err != nil {
		t.Fatal(err)
	}
	stalePath := filepath.Join(saveDir, "stale.dat")
	if err := os.WriteFile(stalePath, []byte("local only"), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := newFakeRepo(&game, nil)
	// 前回同期時点では stale.dat も存在した（base tree に含める）ため、
	// リモートから消えた stale.dat は tracked 削除として確認なしに削除される。
	baseTreeJSON, _ := json.Marshal(domain.SaveSnapshot{Files: map[string]domain.BlobHash{
		"save.dat":  hashBytes([]byte("old local data")),
		"stale.dat": hashBytes([]byte("local only")),
	}})
	repo.saveTree = string(baseTreeJSON)
	svc := newTestService(repo, bstore)

	res, err := svc.Pull(context.Background(), game.ID, nil, false)
	if err != nil {
		t.Fatalf("Pull returned unexpected error: %v", err)
	}
	if !res.Applied {
		t.Fatalf("expected Pull to apply changes, got %+v", res)
	}

	// セーブファイルが復元された
	got, err := os.ReadFile(filepath.Join(saveDir, "save.dat"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "remote data" {
		t.Errorf("save.dat = %q, want %q", string(got), "remote data")
	}
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("stale file should be removed, stat err: %v", err)
	}

	// ゲームが更新された
	if repo.upsertedGame == nil {
		t.Error("expected UpsertGameSync to be called")
	}

	// セッションが差し替えられた
	if !repo.deletedSessions {
		t.Error("expected DeletePlaySessionsByGame to be called")
	}
	if len(repo.upsertedSessions) == 0 {
		t.Error("expected sessions to be upserted")
	}

	// 同期ヘッドが更新された
	if repo.localSyncHeadSet == "" {
		t.Error("expected SetLocalSyncHead to be called")
	}
}

func TestContentSyncServicePullSkipsUnchangedFiles(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	remoteContent := []byte("remote save data")
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), remoteContent, 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, nil, saveDir)

	// ローカルファイルはリモートと同一のまま（上書きしない）
	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	if _, err := svc.Pull(context.Background(), game.ID, nil, false); err != nil {
		t.Fatalf("Pull: %v", err)
	}

	// downloadBlobs が呼ばれても空マップで呼ばれているはず（スキップ）
	// OR 呼ばれていない（0件ならスキップ）
	bstore.mu.Lock()
	calls := bstore.downloadedBlobs
	bstore.mu.Unlock()

	for _, call := range calls {
		if len(call) > 0 {
			t.Errorf("expected no blobs to be downloaded, but got: %v", call)
		}
	}
}

func TestContentSyncServicePullReturnsErrorWhenNoRemoteHead(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	game := baseGame(saveDir)
	repo := newFakeRepo(&game, nil)
	bstore := newFakeBlobStore() // HEAD 未設定
	svc := newTestService(repo, bstore)

	_, err := svc.Pull(context.Background(), game.ID, nil, false)
	if err == nil {
		t.Fatal("expected error when remote HEAD is empty")
	}
}

func TestContentSyncServicePullRejectsEscapingSavePath(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	game := baseGame(saveDir)
	repo := newFakeRepo(&game, nil)
	bstore := newFakeBlobStore()
	svc := newTestService(repo, bstore)

	evilData := []byte("evil")
	evilHash := hashBytes(evilData)
	saveSnap := domain.SaveSnapshot{Files: map[string]domain.BlobHash{
		"../outside.txt": evilHash,
	}}
	saveSnapJSON, err := json.Marshal(saveSnap)
	if err != nil {
		t.Fatal(err)
	}
	savesHash := hashBytes(saveSnapJSON)
	if err := bstore.putBlob(context.Background(), game.ID, storage.BlobKindObject, evilHash, evilData); err != nil {
		t.Fatal(err)
	}
	if err := bstore.putBlob(context.Background(), game.ID, storage.BlobKindTree, savesHash, saveSnapJSON); err != nil {
		t.Fatal(err)
	}

	meta, err := buildMetaSnapshot(game, nil, "", savesHash, "testdevice")
	if err != nil {
		t.Fatal(err)
	}
	metaHash := hashBytes(meta.SnapshotBytes)
	if err := bstore.putBlob(context.Background(), game.ID, storage.BlobKindMeta, meta.Snapshot.GameJSON, meta.GameJSON); err != nil {
		t.Fatal(err)
	}
	if err := bstore.putBlob(context.Background(), game.ID, storage.BlobKindMeta, meta.Snapshot.SessionsJSON, meta.SessionsJSON); err != nil {
		t.Fatal(err)
	}
	if err := bstore.putBlob(context.Background(), game.ID, storage.BlobKindCommit, metaHash, meta.SnapshotBytes); err != nil {
		t.Fatal(err)
	}
	if err := bstore.writeHEAD(context.Background(), game.ID, metaHash); err != nil {
		t.Fatal(err)
	}

	_, err = svc.Pull(context.Background(), game.ID, nil, false)
	if err == nil {
		t.Fatal("expected error for escaping save path")
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(saveDir), "outside.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("outside file should not be written, stat err: %v", statErr)
	}
}

func TestContentSyncServicePullRejectsMismatchedRemoteGameID(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("remote"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()

	saveSnap, saveBlobs, err := buildSaveSnapshot(saveDir)
	if err != nil {
		t.Fatal(err)
	}
	for hash, data := range saveBlobs {
		if err := bstore.putBlob(context.Background(), game.ID, storage.BlobKindObject, hash, data); err != nil {
			t.Fatal(err)
		}
	}
	saveSnapJSON, err := json.Marshal(saveSnap)
	if err != nil {
		t.Fatal(err)
	}
	savesHash := hashBytes(saveSnapJSON)
	if err := bstore.putBlob(context.Background(), game.ID, storage.BlobKindTree, savesHash, saveSnapJSON); err != nil {
		t.Fatal(err)
	}

	remoteGame := game
	remoteGame.ID = "other-game"
	meta, err := buildMetaSnapshot(remoteGame, nil, "", savesHash, "testdevice")
	if err != nil {
		t.Fatal(err)
	}
	metaHash := hashBytes(meta.SnapshotBytes)
	if err := bstore.putBlob(context.Background(), game.ID, storage.BlobKindMeta, meta.Snapshot.GameJSON, meta.GameJSON); err != nil {
		t.Fatal(err)
	}
	if err := bstore.putBlob(context.Background(), game.ID, storage.BlobKindMeta, meta.Snapshot.SessionsJSON, meta.SessionsJSON); err != nil {
		t.Fatal(err)
	}
	if err := bstore.putBlob(context.Background(), game.ID, storage.BlobKindCommit, metaHash, meta.SnapshotBytes); err != nil {
		t.Fatal(err)
	}
	if err := bstore.writeHEAD(context.Background(), game.ID, metaHash); err != nil {
		t.Fatal(err)
	}

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	if _, err := svc.Pull(context.Background(), game.ID, nil, false); err == nil {
		t.Fatal("expected Pull to reject mismatched remote game ID")
	}
	if repo.upsertedGame != nil {
		t.Fatal("mismatched remote game should not be upserted")
	}
}

func TestContentSyncServicePullReportsProgress(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, nil, saveDir)

	// ローカルを古い内容に書き換えてダウンロードを発生させる
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	var maxCurrent int
	_, err := svc.Pull(context.Background(), game.ID, func(current, total int) {
		if current > maxCurrent {
			maxCurrent = current
		}
	}, false)
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if maxCurrent == 0 {
		t.Error("expected progress to be reported")
	}
}

// ─── Status tests ────────────────────────────────────────────────────────────

func TestContentSyncServiceStatusReturnsNeverSynced(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	game := baseGame(saveDir)
	repo := newFakeRepo(&game, nil)
	bstore := newFakeBlobStore() // HEAD 未設定
	svc := newTestService(repo, bstore)

	detail, err := svc.Status(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if detail.Status != domain.SyncStatusNeverSynced {
		t.Errorf("Status = %q, want %q", detail.Status, domain.SyncStatusNeverSynced)
	}
}

func TestContentSyncServiceStatusReturnsInSync(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	remoteMeta := setupRemoteState(t, bstore, game.ID, game, nil, saveDir)

	// LocalSyncHead = remote の fingerprint
	fp := contentFingerprint(remoteMeta)
	game.LocalSyncHead = &fp

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	detail, err := svc.Status(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if detail.Status != domain.SyncStatusInSync {
		t.Errorf("Status = %q, want %q", detail.Status, domain.SyncStatusInSync)
	}
}

func TestContentSyncServiceStatusReturnsPushNeeded(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	remoteMeta := setupRemoteState(t, bstore, game.ID, game, nil, saveDir)

	// LocalSyncHead = remote（初期状態は同期済み）
	fp := contentFingerprint(remoteMeta)
	game.LocalSyncHead = &fp

	// ローカルにファイルを追加してローカルが変更されたとみなす
	if err := os.WriteFile(filepath.Join(saveDir, "save2.dat"), []byte("new file"), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	detail, err := svc.Status(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if detail.Status != domain.SyncStatusPushNeeded {
		t.Errorf("Status = %q, want %q", detail.Status, domain.SyncStatusPushNeeded)
	}
}

func TestContentSyncServiceStatusReturnsPullNeeded(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	sessions := []domain.PlaySession{}

	// ローカルの状態を fingerprint として LocalSyncHead に設定
	localSaveSnap, _, err := buildSaveSnapshot(saveDir)
	if err != nil {
		t.Fatal(err)
	}
	localSaveSnapJSON, _ := json.Marshal(localSaveSnap)
	localSavesHash := hashBytes(localSaveSnapJSON)
	localMeta, err := buildMetaSnapshot(game, sessions, "", localSavesHash, "testdevice")
	if err != nil {
		t.Fatal(err)
	}
	localFP := contentFingerprint(localMeta.Snapshot)
	game.LocalSyncHead = &localFP

	// リモートは異なる内容（別ファイル付き）を設定
	remoteDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(remoteDir, "save.dat"), []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(remoteDir, "remote_extra.dat"), []byte("remote only"), 0o600); err != nil {
		t.Fatal(err)
	}

	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, sessions, remoteDir)

	repo := newFakeRepo(&game, sessions)
	svc := newTestService(repo, bstore)

	detail, err := svc.Status(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if detail.Status != domain.SyncStatusPullNeeded {
		t.Errorf("Status = %q, want %q", detail.Status, domain.SyncStatusPullNeeded)
	}
}

func TestContentSyncServiceStatusReturnsConflict(t *testing.T) {
	t.Parallel()

	// LocalSyncHead を古い基準点に設定し、ローカルとリモートの両方が変更されている状態
	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("base"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)

	// 基準: 空のフォルダ状態として LocalSyncHead を設定
	emptyDir := t.TempDir()
	baseSaveSnap, _, err := buildSaveSnapshot(emptyDir)
	if err != nil {
		t.Fatal(err)
	}
	baseSaveSnapJSON, _ := json.Marshal(baseSaveSnap)
	baseSavesHash := hashBytes(baseSaveSnapJSON)
	baseMeta, err := buildMetaSnapshot(game, nil, "", baseSavesHash, "testdevice")
	if err != nil {
		t.Fatal(err)
	}
	baseFP := contentFingerprint(baseMeta.Snapshot)
	game.LocalSyncHead = &baseFP // 基準は「ファイルなし」

	// リモートは「save.dat = remote content」
	remoteDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(remoteDir, "save.dat"), []byte("remote content"), 0o600); err != nil {
		t.Fatal(err)
	}
	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, nil, remoteDir)

	// ローカルは「save.dat = base」（上で既に書き込み済み） → 基準と異なる

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	detail, err := svc.Status(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if detail.Status != domain.SyncStatusConflict {
		t.Errorf("Status = %q, want %q", detail.Status, domain.SyncStatusConflict)
	}
	// コンフリクト時はローカル/リモートのメタが含まれる
	if detail.LocalMeta == nil {
		t.Error("expected LocalMeta to be set on conflict")
	}
	if detail.RemoteMeta == nil {
		t.Error("expected RemoteMeta to be set on conflict")
	}
}

func TestContentSyncServiceResolveConflictUsesLocalCallsPush(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("local"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	repo := newFakeRepo(&game, nil)
	bstore := newFakeBlobStore()
	svc := newTestService(repo, bstore)

	if _, err := svc.ResolveConflict(context.Background(), game.ID, true, false); err != nil {
		t.Fatalf("ResolveConflict(useLocal=true): %v", err)
	}
	if bstore.heads[game.ID] == "" {
		t.Error("expected HEAD to be set (Push was called)")
	}
}

func TestContentSyncServiceResolveConflictUseLocalOverridesChangedRemote(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("local"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	baseMeta := setupRemoteState(t, bstore, game.ID, game, nil, saveDir)
	baseFP := contentFingerprint(baseMeta)
	game.LocalSyncHead = &baseFP

	remoteDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(remoteDir, "save.dat"), []byte("remote changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	setupRemoteState(t, bstore, game.ID, game, nil, remoteDir)
	remoteHead := bstore.heads[game.ID]

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	if _, err := svc.ResolveConflict(context.Background(), game.ID, true, false); err != nil {
		t.Fatalf("ResolveConflict(useLocal=true): %v", err)
	}
	if bstore.heads[game.ID] == remoteHead {
		t.Fatal("expected remote HEAD to be overwritten by local resolution")
	}
}

func TestContentSyncServiceDeleteFromCloudCallsDeleteWithCorrectPrefix(t *testing.T) {
	t.Parallel()

	repo := newFakeRepo(nil, nil)
	bstore := newFakeBlobStore()
	svc := newTestService(repo, bstore)

	if err := svc.DeleteFromCloud(context.Background(), "game-1"); err != nil {
		t.Fatalf("DeleteFromCloud: %v", err)
	}

	bstore.mu.Lock()
	defer bstore.mu.Unlock()
	if len(bstore.deletedPrefixes) == 0 {
		t.Fatal("expected deleteByPrefix to be called")
	}
	want := "games/game-1/"
	if bstore.deletedPrefixes[0] != want {
		t.Errorf("deleteByPrefix called with %q, want %q", bstore.deletedPrefixes[0], want)
	}
}

// TestContentSyncServiceStatusReturnsPullNeededWhenLocalSyncHeadUnset は
// LocalSyncHead が未設定（他PCで一度も同期していない）のとき conflict ではなく
// pull_needed を返すことを確認する。
func TestContentSyncServiceStatusReturnsPullNeededWhenLocalSyncHeadUnset(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("local data"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	// LocalSyncHead は nil のまま（このPCで一度も同期していない）
	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, nil, saveDir)

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	detail, err := svc.Status(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if detail.Status != domain.SyncStatusPullNeeded {
		t.Errorf("Status = %q, want %q (LocalSyncHead unset should not be conflict)", detail.Status, domain.SyncStatusPullNeeded)
	}
}

// TestContentSyncServicePullThenAddSaveThenPushSucceeds は PC-A push → PC-B pull →
// PC-B がセーブを追加 → PC-B が push できることを確認する。
func TestContentSyncServicePullThenAddSaveThenPushSucceeds(t *testing.T) {
	t.Parallel()

	// PC-A 側の初期セーブ
	saveDirA := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDirA, "save.dat"), []byte("pc-a save"), 0o600); err != nil {
		t.Fatal(err)
	}
	gameA := baseGame(saveDirA)
	bstore := newFakeBlobStore()
	repoA := newFakeRepo(&gameA, nil)
	svcA := newTestService(repoA, bstore)

	// PC-A が push
	if err := svcA.Push(context.Background(), gameA.ID, nil); err != nil {
		t.Fatalf("PC-A Push: %v", err)
	}

	// PC-B の環境（同じゲームID、別セーブフォルダ）
	saveDirB := t.TempDir()
	gameB := baseGame(saveDirB)
	gameB.LocalSyncHead = nil // PC-B では未同期
	repoB := newFakeRepo(&gameB, nil)
	svcB := newTestService(repoB, bstore)

	// PC-B が pull
	if _, err := svcB.Pull(context.Background(), gameB.ID, nil, false); err != nil {
		t.Fatalf("PC-B Pull: %v", err)
	}

	// Pull 後は LocalSyncHead が設定されているはず
	if repoB.localSyncHeadSet == "" {
		t.Fatal("PC-B Pull should have set LocalSyncHead")
	}

	// PC-B がセーブを追加
	if err := os.WriteFile(filepath.Join(saveDirB, "save2.dat"), []byte("pc-b new save"), 0o600); err != nil {
		t.Fatal(err)
	}

	// PC-B が push できる（エラーにならない）
	headBefore := bstore.heads[gameB.ID]
	if err := svcB.Push(context.Background(), gameB.ID, nil); err != nil {
		t.Fatalf("PC-B Push after Pull+add save: %v", err)
	}
	if bstore.heads[gameB.ID] == headBefore {
		t.Error("expected remote HEAD to be updated after PC-B push")
	}
}

// TestContentSyncServicePullRequiresConfirmationForUntracked は、base tree に無い
// ローカル固有ファイル（untracked）を削除する必要があるとき、Pull が変更を加えずに
// 確認待ち（Applied=false）を返すことを確認する。
func TestContentSyncServicePullRequiresConfirmationForUntracked(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("remote data"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, nil, saveDir)

	// 同期が知らないローカル固有ファイルを置く（base tree は空のまま）
	untrackedPath := filepath.Join(saveDir, "user_notes.txt")
	if err := os.WriteFile(untrackedPath, []byte("personal"), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	// deleteUntracked=false → 確認待ちを返し、何も変更しない
	res, err := svc.Pull(context.Background(), game.ID, nil, false)
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if res.Applied {
		t.Fatal("expected Applied=false (confirmation required)")
	}
	if len(res.UntrackedDeletes) != 1 || res.UntrackedDeletes[0] != "user_notes.txt" {
		t.Fatalf("UntrackedDeletes = %v, want [user_notes.txt]", res.UntrackedDeletes)
	}
	if _, statErr := os.Stat(untrackedPath); statErr != nil {
		t.Fatalf("untracked file must remain on confirmation-required path: %v", statErr)
	}
	if repo.upsertedGame != nil {
		t.Fatal("no DB changes should occur on confirmation-required path")
	}
	if repo.localSyncHeadSet != "" {
		t.Fatal("LocalSyncHead must not be set on confirmation-required path")
	}

	// deleteUntracked=true → 適用され、untracked も削除される
	res, err = svc.Pull(context.Background(), game.ID, nil, true)
	if err != nil {
		t.Fatalf("Pull(deleteUntracked=true): %v", err)
	}
	if !res.Applied {
		t.Fatal("expected Applied=true after confirmation")
	}
	if _, statErr := os.Stat(untrackedPath); !os.IsNotExist(statErr) {
		t.Fatalf("untracked file should be deleted after confirmation, stat err: %v", statErr)
	}
}
