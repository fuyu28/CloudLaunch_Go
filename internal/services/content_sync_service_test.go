package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/infrastructure/storage"
)

// ─── fakeContentSyncRepository ──────────────────────────────────────────────

type fakeContentSyncRepository struct {
	mu sync.Mutex

	game     *domain.Game
	sessions []domain.PlaySession
	routes   []domain.Route
	settings map[string]string
	saveTree string
	pending  map[string]domain.PendingPush
	pullOps  map[string]domain.PullOperation

	// 記録された呼び出し
	localSyncHeadSet string
	saveTreeSet      string
	upsertedGame     *domain.Game
	deletedSessions  bool
	upsertedSessions []domain.PlaySession
	upsertedRoutes   []domain.Route
	replacedRoutes   bool
	applyPullV2Err   error
	applyPullErr     error
	lastPullOpID     string

	// エラー注入
	getGameErr           error
	finalizePendingErr   error
	finalizePendingFails int // >0 のあいだ FinalizePendingPush を失敗させる
	beginPullOpErr       error
}

func newFakeRepo(game *domain.Game, sessions []domain.PlaySession) *fakeContentSyncRepository {
	return &fakeContentSyncRepository{
		game:     game,
		sessions: sessions,
		settings: make(map[string]string),
		pending:  make(map[string]domain.PendingPush),
		pullOps:  make(map[string]domain.PullOperation),
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
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.PlaySession{}, r.sessions...), nil
}

func (r *fakeContentSyncRepository) ListRoutesByGame(_ context.Context, _ string) ([]domain.Route, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.Route{}, r.routes...), nil
}

func (r *fakeContentSyncRepository) GetLocalSaveTree(_ context.Context, _ string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.saveTree, nil
}

func (r *fakeContentSyncRepository) SetLocalSyncState(_ context.Context, _ string, syncHead, saveTree string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.localSyncHeadSet = syncHead
	r.saveTree = saveTree
	r.saveTreeSet = saveTree
	if r.game != nil {
		r.game.LocalSyncHead = &syncHead
	}
	return nil
}

func (r *fakeContentSyncRepository) BeginPendingPush(_ context.Context, pending domain.PendingPush) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pending[pending.GameID] = pending
	return nil
}

func (r *fakeContentSyncRepository) FinalizePendingPush(_ context.Context, gameID, syncHead, saveTree string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.finalizePendingFails > 0 {
		r.finalizePendingFails--
		if r.finalizePendingErr != nil {
			return r.finalizePendingErr
		}
		return fmt.Errorf("injected finalize pending failure")
	}
	r.localSyncHeadSet = syncHead
	r.saveTree = saveTree
	r.saveTreeSet = saveTree
	if r.game != nil {
		r.game.LocalSyncHead = &syncHead
	}
	delete(r.pending, gameID)
	return nil
}

func (r *fakeContentSyncRepository) ClearPendingPush(_ context.Context, gameID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.pending, gameID)
	return nil
}

func (r *fakeContentSyncRepository) ListPendingPushes(_ context.Context) ([]domain.PendingPush, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.PendingPush, 0, len(r.pending))
	for _, item := range r.pending {
		out = append(out, item)
	}
	return out, nil
}

func (r *fakeContentSyncRepository) BeginPullOperation(_ context.Context, op domain.PullOperation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.beginPullOpErr != nil {
		return r.beginPullOpErr
	}
	if op.Status == "" {
		op.Status = domain.PullOperationPrepared
	}
	r.pullOps[op.OperationID] = op
	return nil
}

func (r *fakeContentSyncRepository) ClearPullOperation(_ context.Context, operationID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.pullOps, operationID)
	return nil
}

func (r *fakeContentSyncRepository) ListPullOperations(_ context.Context) ([]domain.PullOperation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.PullOperation, 0, len(r.pullOps))
	for _, item := range r.pullOps {
		out = append(out, item)
	}
	return out, nil
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
	syncHead, saveTree, pullOperationID string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.applyPullErr != nil {
		return r.applyPullErr
	}
	r.lastPullOpID = pullOperationID
	r.upsertedGame = &game
	r.deletedSessions = true
	r.replacedRoutes = false
	r.upsertedSessions = append([]domain.PlaySession{}, sessions...)
	r.localSyncHeadSet = syncHead
	if r.game != nil {
		r.game.LocalSyncHead = &syncHead
		r.game.CurrentRouteID = game.CurrentRouteID
	}
	r.sessions = append([]domain.PlaySession{}, sessions...)
	r.saveTree = saveTree
	r.saveTreeSet = saveTree
	delete(r.pending, game.ID)
	if pullOperationID != "" {
		if op, ok := r.pullOps[pullOperationID]; ok {
			op.Status = domain.PullOperationApplied
			r.pullOps[pullOperationID] = op
		}
	}
	return nil
}

func (r *fakeContentSyncRepository) ApplyPullResultV2(
	_ context.Context,
	game domain.Game,
	routes []domain.Route,
	sessions []domain.PlaySession,
	syncHead, saveTree, pullOperationID string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.applyPullV2Err != nil {
		return r.applyPullV2Err
	}
	r.lastPullOpID = pullOperationID
	r.upsertedGame = &game
	r.deletedSessions = true
	r.replacedRoutes = true
	r.upsertedRoutes = append([]domain.Route{}, routes...)
	r.routes = append([]domain.Route{}, routes...)
	r.upsertedSessions = append([]domain.PlaySession{}, sessions...)
	r.sessions = append([]domain.PlaySession{}, sessions...)
	r.localSyncHeadSet = syncHead
	if r.game != nil {
		r.game.LocalSyncHead = &syncHead
		r.game.CurrentRouteID = game.CurrentRouteID
		r.game.Title = game.Title
	}
	r.saveTree = saveTree
	r.saveTreeSet = saveTree
	delete(r.pending, game.ID)
	if pullOperationID != "" {
		if op, ok := r.pullOps[pullOperationID]; ok {
			op.Status = domain.PullOperationApplied
			r.pullOps[pullOperationID] = op
		}
	}
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

	blobs   map[string][]byte // キー: "gameID/hash"
	heads   map[string]string // gameID → HEAD.v2 metaHash
	headsV1 map[string]string // gameID → レガシー HEAD（読取フォールバックのみ）

	// 記録された呼び出し
	downloadedBlobs []map[string]string // 各呼び出しの blobs 引数
	deletedPrefixes []string

	// onPutBlobs は putBlobs 呼び出し時に1回呼ばれるフック。
	// テストでアップロード中の HEAD 変更（別デバイスの並行 push）を模すのに使う。nil 可。
	onPutBlobs func()
	// onDownloadBlobs は downloadBlobs 呼び出し時に1回呼ばれるフック。
	// Push の onPutBlobs と同様に、Pull 中のロック保持を検証するのに使う。nil 可。
	onDownloadBlobs func()
	// downloadErr が非 nil なら downloadBlobs は書き込み前に失敗する（stage 失敗テスト用）。
	downloadErr error
}

func newFakeBlobStore() *fakeBlobStore {
	return &fakeBlobStore{
		blobs:   make(map[string][]byte),
		heads:   make(map[string]string),
		headsV1: make(map[string]string),
	}
}

func (f *fakeBlobStore) blobKey(gameID, kind, hash string) string {
	return gameID + "/" + kind + "/" + hash
}

func (f *fakeBlobStore) readHEAD(_ context.Context, gameID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if h := f.heads[gameID]; h != "" {
		return h, nil
	}
	return f.headsV1[gameID], nil
}

func (f *fakeBlobStore) writeHEAD(_ context.Context, gameID, hash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	// 新クライアントは HEAD.v2 のみ書き、レガシー HEAD は触らない。
	f.heads[gameID] = hash
	return nil
}

func (f *fakeBlobStore) writeHEADv1(_ context.Context, gameID, hash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.headsV1[gameID] = hash
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
	if f.onPutBlobs != nil {
		f.onPutBlobs()
	}
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
	if f.onDownloadBlobs != nil {
		f.onDownloadBlobs()
	}
	// 呼び出しを記録する
	snapshot := make(map[string]string, len(blobs))
	for k, v := range blobs {
		snapshot[k] = v
	}
	f.mu.Lock()
	f.downloadedBlobs = append(f.downloadedBlobs, snapshot)
	downloadErr := f.downloadErr
	f.mu.Unlock()
	if downloadErr != nil {
		return downloadErr
	}

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
	seen := make(map[string]struct{})
	var ids []string
	for gameID := range f.heads {
		seen[gameID] = struct{}{}
		ids = append(ids, gameID)
	}
	for gameID := range f.headsV1 {
		if _, ok := seen[gameID]; !ok {
			ids = append(ids, gameID)
		}
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
// テストで「remote = 現在のローカル」という基準点を作るために使う（v2 / HEAD.v2）。
func setupRemoteState(
	t *testing.T,
	bstore *fakeBlobStore,
	gameID string,
	game domain.Game,
	sessions []domain.PlaySession,
	saveDir string,
) domain.MetaSnapshot {
	t.Helper()
	return setupRemoteStateWithRoutes(t, bstore, gameID, game, sessions, nil, saveDir, true)
}

// setupRemoteStateWithRoutes は routes と HEAD 書き込み先（v2/v1）を指定してリモート状態を作る。
func setupRemoteStateWithRoutes(
	t *testing.T,
	bstore *fakeBlobStore,
	gameID string,
	game domain.Game,
	sessions []domain.PlaySession,
	routes []domain.Route,
	saveDir string,
	writeV2 bool,
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

	meta, err := buildMetaSnapshot(game, sessions, routes, "", savesHash, "testdevice", 0, 0)
	if err != nil {
		t.Fatalf("buildMetaSnapshot: %v", err)
	}
	if !writeV2 {
		// レガシー v1 commit: SchemaVersion / RoutesJSON を落とす
		meta.Snapshot.SchemaVersion = domain.SyncSchemaVersionV1
		meta.Snapshot.RoutesJSON = ""
		meta.RoutesJSON = nil
		meta.SnapshotBytes, err = json.Marshal(meta.Snapshot)
		if err != nil {
			t.Fatalf("marshal v1 meta: %v", err)
		}
	}
	metaHash := hashBytes(meta.SnapshotBytes)

	if err := bstore.putBlob(ctx, gameID, storage.BlobKindMeta, meta.Snapshot.GameJSON, meta.GameJSON); err != nil {
		t.Fatalf("putBlob gameJSON: %v", err)
	}
	if err := bstore.putBlob(ctx, gameID, storage.BlobKindMeta, meta.Snapshot.SessionsJSON, meta.SessionsJSON); err != nil {
		t.Fatalf("putBlob sessionsJSON: %v", err)
	}
	if meta.Snapshot.RoutesJSON != "" && meta.RoutesJSON != nil {
		if err := bstore.putBlob(ctx, gameID, storage.BlobKindMeta, meta.Snapshot.RoutesJSON, meta.RoutesJSON); err != nil {
			t.Fatalf("putBlob routesJSON: %v", err)
		}
	}
	if err := bstore.putBlob(ctx, gameID, storage.BlobKindCommit, metaHash, meta.SnapshotBytes); err != nil {
		t.Fatalf("putBlob meta: %v", err)
	}
	if writeV2 {
		if err := bstore.writeHEAD(ctx, gameID, metaHash); err != nil {
			t.Fatalf("writeHEAD: %v", err)
		}
	} else if err := bstore.writeHEADv1(ctx, gameID, metaHash); err != nil {
		t.Fatalf("writeHEADv1: %v", err)
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

// putCommitBlobs は MetaSnapshot と依存 meta ブロブを HEAD.v2 に載せる。
func putCommitBlobs(t *testing.T, bstore *fakeBlobStore, gameID string, meta metaBuildResult) {
	t.Helper()
	ctx := context.Background()
	metaHash := hashBytes(meta.SnapshotBytes)
	if err := bstore.putBlob(ctx, gameID, storage.BlobKindMeta, meta.Snapshot.GameJSON, meta.GameJSON); err != nil {
		t.Fatal(err)
	}
	if err := bstore.putBlob(ctx, gameID, storage.BlobKindMeta, meta.Snapshot.SessionsJSON, meta.SessionsJSON); err != nil {
		t.Fatal(err)
	}
	if meta.Snapshot.RoutesJSON != "" && meta.RoutesJSON != nil {
		if err := bstore.putBlob(ctx, gameID, storage.BlobKindMeta, meta.Snapshot.RoutesJSON, meta.RoutesJSON); err != nil {
			t.Fatal(err)
		}
	}
	if err := bstore.putBlob(ctx, gameID, storage.BlobKindCommit, metaHash, meta.SnapshotBytes); err != nil {
		t.Fatal(err)
	}
	if err := bstore.writeHEAD(ctx, gameID, metaHash); err != nil {
		t.Fatal(err)
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
		t.Error("expected local sync baseline to be finalized")
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

// TestContentSyncServicePushAbortsWhenRemoteHeadChangesMidUpload は、アップロード中に
// 別デバイスがリモート HEAD を進めた場合、push が writeHEAD せず中断することを確認する。
func TestContentSyncServicePushAbortsWhenRemoteHeadChangesMidUpload(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir) // localSyncHead 未設定、remote HEAD なしから開始
	repo := newFakeRepo(&game, nil)
	bstore := newFakeBlobStore()
	// アップロード中（putBlobs）に別デバイスが HEAD を進めた状況を模す
	bstore.onPutBlobs = func() {
		_ = bstore.writeHEAD(context.Background(), game.ID, "concurrent-head")
	}
	svc := newTestService(repo, bstore)

	err := svc.Push(context.Background(), game.ID, nil)
	if err == nil {
		t.Fatal("Push should abort when remote HEAD changed during upload")
	}

	// 別デバイスの HEAD が上書きされていないこと
	head, _ := bstore.readHEAD(context.Background(), game.ID)
	if head != "concurrent-head" {
		t.Fatalf("remote HEAD should not be overwritten, got %q", head)
	}
	// 中断したので localSyncHead は更新されていないこと
	if repo.localSyncHeadSet != "" {
		t.Fatalf("localSyncHead should not be set on aborted push, got %q", repo.localSyncHeadSet)
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
		t.Error("expected local sync baseline to be finalized")
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

	meta, err := buildMetaSnapshot(game, nil, nil, "", savesHash, "testdevice", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	putCommitBlobs(t, bstore, game.ID, meta)

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
	meta, err := buildMetaSnapshot(remoteGame, nil, nil, "", savesHash, "testdevice", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	putCommitBlobs(t, bstore, game.ID, meta)

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
	localMeta, err := buildMetaSnapshot(game, sessions, nil, "", localSavesHash, "testdevice", 0, 0)
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
	baseMeta, err := buildMetaSnapshot(game, nil, nil, "", baseSavesHash, "testdevice", 0, 0)
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

// TestContentSyncServiceDeleteFromCloudClearsLocalSyncState は、リモート削除後に
// ローカルの localSyncHead / localSaveTree がクリアされることを確認する。
func TestContentSyncServiceDeleteFromCloudClearsLocalSyncState(t *testing.T) {
	t.Parallel()

	game := baseGame(t.TempDir())
	game.LocalSyncHead = strPtr("old-head")
	repo := newFakeRepo(&game, nil)
	repo.saveTree = "{\"files\":{\"a.sav\":\"h\"}}"
	bstore := newFakeBlobStore()
	svc := newTestService(repo, bstore)

	if err := svc.DeleteFromCloud(context.Background(), game.ID); err != nil {
		t.Fatalf("DeleteFromCloud: %v", err)
	}

	if repo.localSyncHeadSet != "" {
		t.Errorf("localSyncHead should be cleared, got %q", repo.localSyncHeadSet)
	}
	if repo.saveTreeSet != "" {
		t.Errorf("localSaveTree should be cleared, got %q", repo.saveTreeSet)
	}
}

// TestContentSyncServiceStatusReportsSavesDifferFalseWhenOnlyMetadataChanged は、
// リモート push 済みの状態からセッションだけ増やしたケース（セーブファイルは byte 同一）で
// PushNeeded にはなるが SavesDiffer=false になることを確認する。
// 「セッション終了後にセーブ不変でもアップロード確認プロンプトが出る」バグの再現テスト。
func TestContentSyncServiceStatusReportsSavesDifferFalseWhenOnlyMetadataChanged(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()

	// リモートはセッション 0 件で push 済み
	remoteMeta := setupRemoteState(t, bstore, game.ID, game, nil, saveDir)
	fp := contentFingerprint(remoteMeta)
	game.LocalSyncHead = &fp

	// ローカルにだけ新しいセッションを追加（セーブファイルは byte 同一）
	sessions := []domain.PlaySession{
		{
			ID:        "s1",
			GameID:    game.ID,
			PlayedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Duration:  600,
			UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	repo := newFakeRepo(&game, sessions)
	svc := newTestService(repo, bstore)

	detail, err := svc.Status(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if detail.Status != domain.SyncStatusPushNeeded {
		t.Fatalf("Status = %q, want %q", detail.Status, domain.SyncStatusPushNeeded)
	}
	if detail.SavesDiffer {
		t.Fatal("SavesDiffer should be false when only sessions.json changed (bug repro)")
	}
}

// TestContentSyncServiceStatusReportsSavesDifferTrueWhenSaveContentChanged は、
// セーブファイル内容が実際に変化したときに SavesDiffer=true が返ることを確認する。
func TestContentSyncServiceStatusReportsSavesDifferTrueWhenSaveContentChanged(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	remoteMeta := setupRemoteState(t, bstore, game.ID, game, nil, saveDir)
	fp := contentFingerprint(remoteMeta)
	game.LocalSyncHead = &fp

	// ローカルのセーブ内容を変更（新しいファイル追加）
	if err := os.WriteFile(filepath.Join(saveDir, "save2.dat"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	detail, err := svc.Status(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if detail.Status != domain.SyncStatusPushNeeded {
		t.Fatalf("Status = %q, want %q", detail.Status, domain.SyncStatusPushNeeded)
	}
	if !detail.SavesDiffer {
		t.Fatal("SavesDiffer should be true when save file content changed")
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

// ─── LoadCloudMetadata tests ─────────────────────────────────────────────────

// TestLoadCloudMetadataReturnsAllGamesAndSkipsBroken は、複数ゲームのメタ情報が
// 並列取得で全件返り、HEAD はあるが commit ブロブが壊れているゲームはスキップされることを確認する。
func TestLoadCloudMetadataReturnsAllGamesAndSkipsBroken(t *testing.T) {
	t.Parallel()

	bstore := newFakeBlobStore()
	ctx := context.Background()

	// 正常な2ゲームをリモートに用意する
	for _, id := range []string{"game-a", "game-b"} {
		saveDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(saveDir, "s.dat"), []byte(id), 0o600); err != nil {
			t.Fatal(err)
		}
		g := baseGame(saveDir)
		g.ID = id
		setupRemoteState(t, bstore, id, g, nil, saveDir)
	}

	// HEAD はあるが commit ブロブが存在しない壊れたゲーム
	if err := bstore.writeHEAD(ctx, "game-broken", "missing-head-hash"); err != nil {
		t.Fatal(err)
	}

	repo := newFakeRepo(nil, nil)
	svc := newTestService(repo, bstore)

	infos, err := svc.LoadCloudMetadata(ctx)
	if err != nil {
		t.Fatalf("LoadCloudMetadata: %v", err)
	}

	got := map[string]bool{}
	for _, info := range infos {
		got[info.ID] = true
	}
	if !got["game-a"] || !got["game-b"] {
		t.Fatalf("expected game-a and game-b, got %v", got)
	}
	if got["game-broken"] {
		t.Fatalf("broken game should be skipped, got %v", got)
	}
	if len(infos) != 2 {
		t.Fatalf("expected 2 games, got %d", len(infos))
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

// TestContentSyncServicePushSerializesSameGame は、同一ゲームに対する複数の Push が
// 同時並行に putBlobs（=セーブ走査・アップロード本体）へ突入しないことを確認する。
func TestContentSyncServicePushSerializesSameGame(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "s.dat"), []byte("d"), 0o600); err != nil {
		t.Fatal(err)
	}
	game := baseGame(saveDir)
	repo := newFakeRepo(&game, nil)
	bstore := newFakeBlobStore()

	var concurrent, overlap int32
	bstore.onPutBlobs = func() {
		if atomic.AddInt32(&concurrent, 1) > 1 {
			atomic.StoreInt32(&overlap, 1)
		}
		time.Sleep(20 * time.Millisecond) // 重なりがあれば検出できるよう保持
		atomic.AddInt32(&concurrent, -1)
	}
	svc := newTestService(repo, bstore)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = svc.Push(context.Background(), game.ID, nil)
		}()
	}
	wg.Wait()

	if atomic.LoadInt32(&overlap) != 0 {
		t.Fatal("same-game Push calls must be serialized (no concurrent putBlobs)")
	}
}

// TestContentSyncServiceLockGameAllowsDifferentIDs は、異なる gameID のロックが
// 互いをブロックしないことを確認する。
func TestContentSyncServiceLockGameAllowsDifferentIDs(t *testing.T) {
	t.Parallel()

	svc := newTestService(newFakeRepo(nil, nil), newFakeBlobStore())
	unlockA := svc.lockGame("game-a")
	defer unlockA()

	done := make(chan struct{})
	go func() {
		unlockB := svc.lockGame("game-b")
		unlockB()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("locks for different gameIDs must not block each other")
	}
}

// TestContentSyncServiceStatusWaitsDuringSameGamePush は、同一 gameID の Push 中は
// Status が一貫スナップショット用ロックで待機することを確認する。
func TestContentSyncServiceStatusWaitsDuringSameGamePush(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("local"), 0o600); err != nil {
		t.Fatal(err)
	}
	game := baseGame(saveDir)
	repo := newFakeRepo(&game, nil)
	bstore := newFakeBlobStore()
	svc := newTestService(repo, bstore)

	hold := make(chan struct{})
	entered := make(chan struct{})
	bstore.onPutBlobs = func() {
		close(entered)
		<-hold
	}

	pushDone := make(chan error, 1)
	go func() {
		pushDone <- svc.Push(context.Background(), game.ID, nil)
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("Push did not reach putBlobs")
	}

	statusDone := make(chan struct{})
	go func() {
		_, _ = svc.Status(context.Background(), game.ID)
		close(statusDone)
	}()

	select {
	case <-statusDone:
		t.Fatal("Status must wait while same-game Push holds the lock")
	case <-time.After(100 * time.Millisecond):
	}

	close(hold)
	select {
	case err := <-pushDone:
		if err != nil {
			t.Fatalf("Push: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Push did not finish")
	}
	select {
	case <-statusDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Status did not finish after Push released the lock")
	}
}

// TestContentSyncServiceStatusWaitsDuringSameGamePull は、同一 gameID の Pull 中は
// Status が待機することを確認する。
func TestContentSyncServiceStatusWaitsDuringSameGamePull(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("remote data"), 0o600); err != nil {
		t.Fatal(err)
	}
	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	setupRemoteState(t, bstore, game.ID, game, nil, saveDir)
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("old local data"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)

	hold := make(chan struct{})
	entered := make(chan struct{})
	bstore.onDownloadBlobs = func() {
		close(entered)
		<-hold
	}

	pullDone := make(chan error, 1)
	go func() {
		_, err := svc.Pull(context.Background(), game.ID, nil, false)
		pullDone <- err
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("Pull did not reach downloadBlobs")
	}

	statusDone := make(chan struct{})
	go func() {
		_, _ = svc.Status(context.Background(), game.ID)
		close(statusDone)
	}()

	select {
	case <-statusDone:
		t.Fatal("Status must wait while same-game Pull holds the lock")
	case <-time.After(100 * time.Millisecond):
	}

	close(hold)
	select {
	case err := <-pullDone:
		if err != nil {
			t.Fatalf("Pull: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Pull did not finish")
	}
	select {
	case <-statusDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Status did not finish after Pull released the lock")
	}
}

// TestContentSyncServiceStatusDoesNotBlockDifferentGame は、別 gameID の Push 保持中でも
// Status がブロックされないことを確認する。
func TestContentSyncServiceStatusDoesNotBlockDifferentGame(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("local"), 0o600); err != nil {
		t.Fatal(err)
	}
	gameA := baseGame(saveDir)
	repo := newFakeRepo(&gameA, nil)
	bstore := newFakeBlobStore()
	svc := newTestService(repo, bstore)

	hold := make(chan struct{})
	entered := make(chan struct{})
	bstore.onPutBlobs = func() {
		close(entered)
		<-hold
	}

	pushDone := make(chan error, 1)
	go func() {
		pushDone <- svc.Push(context.Background(), gameA.ID, nil)
	}()

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("Push did not reach putBlobs")
	}

	statusDone := make(chan error, 1)
	go func() {
		_, err := svc.Status(context.Background(), "game-other")
		statusDone <- err
	}()

	select {
	case err := <-statusDone:
		if err != nil {
			t.Fatalf("Status for different game: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Status for a different gameID must not wait on another game's Push")
	}

	close(hold)
	select {
	case err := <-pushDone:
		if err != nil {
			t.Fatalf("Push: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Push did not finish")
	}
}

// TestContentSyncServiceRecoversBaselineAfterRemoteSuccessDBFailure は、
// writeHEAD 成功後に FinalizePendingPush が失敗しても、Recover で baseline が確定することを確認する。
func TestContentSyncServiceRecoversBaselineAfterRemoteSuccessDBFailure(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("recover-me"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	repo := newFakeRepo(&game, nil)
	repo.finalizePendingFails = 1
	bstore := newFakeBlobStore()
	svc := newTestService(repo, bstore)

	if err := svc.Push(context.Background(), game.ID, nil); err == nil {
		t.Fatal("Push should fail when FinalizePendingPush fails after HEAD write")
	}
	head := bstore.heads[game.ID]
	if head == "" {
		t.Fatal("remote HEAD should already be written")
	}
	if repo.localSyncHeadSet != "" {
		t.Fatalf("baseline must not be set before recovery, got %q", repo.localSyncHeadSet)
	}
	if len(repo.pending) != 1 {
		t.Fatalf("pending should remain after DB failure, got %#v", repo.pending)
	}

	if err := svc.RecoverPendingPushes(context.Background()); err != nil {
		t.Fatalf("RecoverPendingPushes: %v", err)
	}
	if repo.localSyncHeadSet == "" {
		t.Fatal("expected baseline to be finalized by recovery")
	}
	if repo.saveTreeSet == "" {
		t.Fatal("expected saveTree to be finalized by recovery")
	}
	if len(repo.pending) != 0 {
		t.Fatalf("pending should be cleared after recovery, got %#v", repo.pending)
	}
	if bstore.heads[game.ID] != head {
		t.Fatalf("remote HEAD must not change during recovery")
	}
}

// TestContentSyncServiceStatusFinalizesPendingPushBeforeJudging は、
// HEAD 成功・baseline 未確定のまま Status を呼んでも偽 Conflict にならないことを確認する。
func TestContentSyncServiceStatusFinalizesPendingPushBeforeJudging(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("pending-status"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	repo := newFakeRepo(&game, nil)
	repo.finalizePendingFails = 1
	bstore := newFakeBlobStore()
	svc := newTestService(repo, bstore)

	if err := svc.Push(context.Background(), game.ID, nil); err == nil {
		t.Fatal("Push should fail when FinalizePendingPush fails")
	}
	if len(repo.pending) != 1 {
		t.Fatalf("pending should remain, got %#v", repo.pending)
	}

	detail, err := svc.Status(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if detail.Status != domain.SyncStatusInSync {
		t.Fatalf("Status = %s, want in_sync after pending finalize", detail.Status)
	}
	if len(repo.pending) != 0 {
		t.Fatalf("pending should be cleared by Status, got %#v", repo.pending)
	}
	if repo.localSyncHeadSet == "" {
		t.Fatal("baseline should be finalized by Status")
	}
}

// TestContentSyncServiceDoesNotFinalizePendingWhenRemoteHeadDiffers は、
// pending 新コミットと remote HEAD が不一致なら自動確定せず pending を破棄することを確認する。
func TestContentSyncServiceDoesNotFinalizePendingWhenRemoteHeadDiffers(t *testing.T) {
	t.Parallel()

	game := baseGame(t.TempDir())
	repo := newFakeRepo(&game, nil)
	bstore := newFakeBlobStore()
	svc := newTestService(repo, bstore)

	repo.pending[game.ID] = domain.PendingPush{
		GameID:             game.ID,
		ExpectedRemoteHead: "old",
		NewCommitHash:      "our-commit",
		ContentFingerprint: "fp-ours",
		SaveTree:           `{"files":{"a":"b"}}`,
	}
	bstore.heads[game.ID] = "someone-elses-commit"

	if err := svc.RecoverPendingPushes(context.Background()); err != nil {
		t.Fatalf("RecoverPendingPushes: %v", err)
	}
	if repo.localSyncHeadSet != "" {
		t.Fatalf("must not auto-finalize baseline, got %q", repo.localSyncHeadSet)
	}
	if len(repo.pending) != 0 {
		t.Fatalf("stale pending should be cleared, got %#v", repo.pending)
	}
}

// TestContentSyncServiceForcePushRecoversPendingBaseline は force Push（ResolveConflict useLocal）でも
// HEAD 成功後の DB 失敗を Recover で確定できることを確認する。
func TestContentSyncServiceForcePushRecoversPendingBaseline(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("local-force"), 0o600); err != nil {
		t.Fatal(err)
	}

	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	baseMeta := setupRemoteState(t, bstore, game.ID, game, nil, saveDir)
	baseFP := contentFingerprint(baseMeta)
	game.LocalSyncHead = &baseFP

	remoteDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(remoteDir, "save.dat"), []byte("remote-other"), 0o600); err != nil {
		t.Fatal(err)
	}
	setupRemoteState(t, bstore, game.ID, game, nil, remoteDir)

	repo := newFakeRepo(&game, nil)
	repo.finalizePendingFails = 1
	svc := newTestService(repo, bstore)

	if _, err := svc.ResolveConflict(context.Background(), game.ID, true, false); err == nil {
		t.Fatal("force push should fail when FinalizePendingPush fails")
	}
	head := bstore.heads[game.ID]
	if head == "" {
		t.Fatal("force push should have written remote HEAD")
	}
	if repo.localSyncHeadSet != "" {
		t.Fatalf("baseline must not be set before recovery, got %q", repo.localSyncHeadSet)
	}

	if err := svc.RecoverPendingPushes(context.Background()); err != nil {
		t.Fatalf("RecoverPendingPushes: %v", err)
	}
	if repo.localSyncHeadSet == "" || repo.saveTreeSet == "" {
		t.Fatal("force-push pending should finalize baseline on recovery")
	}
	if len(repo.pending) != 0 {
		t.Fatalf("pending should be cleared, got %#v", repo.pending)
	}
	if bstore.heads[game.ID] != head {
		t.Fatal("remote HEAD must remain the force-pushed commit")
	}
}

// TestContentSyncServiceRecoverPendingSurvivesBlobStoreInitFailure は、
// ネットワーク不通などで blob store 初期化に失敗しても Recover が fatal 扱いのエラーを返しつつ
// pending を消さないことを確認する（起動側は Warn して継続する）。
func TestContentSyncServiceRecoverPendingSurvivesBlobStoreInitFailure(t *testing.T) {
	t.Parallel()

	game := baseGame(t.TempDir())
	repo := newFakeRepo(&game, nil)
	repo.pending[game.ID] = domain.PendingPush{
		GameID:             game.ID,
		NewCommitHash:      "commit",
		ContentFingerprint: "fp",
		SaveTree:           "{}",
	}
	svc := newTestService(repo, newFakeBlobStore())
	svc.newBlobStore = func(context.Context) (contentBlobStore, error) {
		return nil, fmt.Errorf("network unreachable")
	}

	err := svc.RecoverPendingPushes(context.Background())
	if err == nil {
		t.Fatal("expected network error to surface")
	}
	if len(repo.pending) != 1 {
		t.Fatalf("pending must remain when remote inspect fails, got %#v", repo.pending)
	}
}

// ─── H8: Route sync protocol v2 ──────────────────────────────────────────────

func TestContentSyncServicePushPullPreservesRoutesAcrossDevices(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("shared"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	routeA := domain.Route{ID: "route-a", Name: "本編", Order: 0, GameID: "game-1", CreatedAt: now}
	routeB := domain.Route{ID: "route-b", Name: "後日談", Order: 1, GameID: "game-1", CreatedAt: now}
	routeID := routeA.ID
	game := baseGame(saveDir)
	game.CurrentRouteID = &routeID
	sessions := []domain.PlaySession{
		{ID: "sess-1", GameID: game.ID, PlayedAt: now, Duration: 120, RouteID: &routeID, UpdatedAt: now},
	}

	repoA := newFakeRepo(&game, sessions)
	repoA.routes = []domain.Route{routeA, routeB}
	bstore := newFakeBlobStore()
	svcA := newTestService(repoA, bstore)

	if err := svcA.Push(context.Background(), game.ID, nil); err != nil {
		t.Fatalf("Push: %v", err)
	}
	if bstore.heads[game.ID] == "" {
		t.Fatal("expected HEAD.v2 after Push")
	}
	if bstore.headsV1[game.ID] != "" {
		t.Fatal("Push must not write legacy HEAD")
	}

	gameB := baseGame(saveDir)
	repoB := newFakeRepo(&gameB, nil)
	repoB.routes = []domain.Route{{ID: "local-only", Name: "ローカル", Order: 0, GameID: game.ID, CreatedAt: now}}
	svcB := newTestService(repoB, bstore)

	result, err := svcB.Pull(context.Background(), game.ID, nil, false)
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if !result.Applied {
		t.Fatalf("expected Applied, got %#v", result)
	}
	if !repoB.replacedRoutes {
		t.Fatal("v2 Pull must replace routes")
	}
	if len(repoB.upsertedRoutes) != 2 {
		t.Fatalf("routes = %d, want 2", len(repoB.upsertedRoutes))
	}
	if repoB.upsertedRoutes[0].ID != "route-a" || repoB.upsertedRoutes[1].Name != "後日談" {
		t.Fatalf("routes not preserved: %#v", repoB.upsertedRoutes)
	}
	if repoB.game.CurrentRouteID == nil || *repoB.game.CurrentRouteID != "route-a" {
		t.Fatalf("currentRouteId not preserved: %v", repoB.game.CurrentRouteID)
	}
	if len(repoB.upsertedSessions) != 1 || repoB.upsertedSessions[0].RouteID == nil || *repoB.upsertedSessions[0].RouteID != "route-a" {
		t.Fatalf("session routeId not preserved: %#v", repoB.upsertedSessions)
	}
}

func TestContentSyncServicePullPropagatesRouteDeletion(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	kept := domain.Route{ID: "keep", Name: "残す", Order: 0, GameID: "game-1", CreatedAt: now}
	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	setupRemoteStateWithRoutes(t, bstore, game.ID, game, nil, []domain.Route{kept}, saveDir, true)

	repo := newFakeRepo(&game, nil)
	repo.routes = []domain.Route{
		kept,
		{ID: "gone", Name: "消える", Order: 1, GameID: game.ID, CreatedAt: now},
	}
	svc := newTestService(repo, bstore)

	if _, err := svc.Pull(context.Background(), game.ID, nil, false); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if len(repo.upsertedRoutes) != 1 || repo.upsertedRoutes[0].ID != "keep" {
		t.Fatalf("expected only kept route, got %#v", repo.upsertedRoutes)
	}
}

func TestContentSyncServiceRouteOnlyChangeNeedsPush(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("same"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	routes := []domain.Route{{ID: "r1", Name: "旧名", Order: 0, GameID: "game-1", CreatedAt: now}}
	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	remoteMeta := setupRemoteStateWithRoutes(t, bstore, game.ID, game, nil, routes, saveDir, true)
	fp := contentFingerprint(remoteMeta)
	game.LocalSyncHead = &fp

	repo := newFakeRepo(&game, nil)
	repo.routes = []domain.Route{{ID: "r1", Name: "新名", Order: 0, GameID: game.ID, CreatedAt: now}}
	svc := newTestService(repo, bstore)

	detail, err := svc.Status(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if detail.Status != domain.SyncStatusPushNeeded {
		t.Fatalf("Status = %q, want push_needed for route-only change", detail.Status)
	}
	if detail.SavesDiffer {
		t.Fatal("route-only change must not set SavesDiffer")
	}
}

func TestContentSyncServiceV1PullKeepsLocalRoutesAndPushCreatesV2(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("v1"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	setupRemoteStateWithRoutes(t, bstore, game.ID, game, nil, nil, saveDir, false)
	if bstore.heads[game.ID] != "" || bstore.headsV1[game.ID] == "" {
		t.Fatal("fixture must use legacy HEAD only")
	}

	localRoute := domain.Route{ID: "local-r", Name: "手元ルート", Order: 0, GameID: game.ID, CreatedAt: now}
	repo := newFakeRepo(&game, nil)
	repo.routes = []domain.Route{localRoute}
	svc := newTestService(repo, bstore)

	if _, err := svc.Pull(context.Background(), game.ID, nil, false); err != nil {
		t.Fatalf("v1 Pull: %v", err)
	}
	if repo.replacedRoutes {
		t.Fatal("v1 Pull must not replace local routes")
	}
	if len(repo.routes) != 1 || repo.routes[0].ID != "local-r" {
		t.Fatalf("local routes should remain, got %#v", repo.routes)
	}

	if err := svc.Push(context.Background(), game.ID, nil); err != nil {
		t.Fatalf("migration Push: %v", err)
	}
	if bstore.heads[game.ID] == "" {
		t.Fatal("next Push must create HEAD.v2")
	}
	if bstore.headsV1[game.ID] == "" {
		t.Fatal("legacy HEAD must remain (no overwrite/delete)")
	}
	// HEAD.v2 の commit が v2 であること
	metaBytes, err := bstore.getBlob(context.Background(), game.ID, storage.BlobKindCommit, bstore.heads[game.ID])
	if err != nil {
		t.Fatal(err)
	}
	var meta domain.MetaSnapshot
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatal(err)
	}
	if meta.SchemaVersion != domain.SyncSchemaVersionV2 || meta.RoutesJSON == "" {
		t.Fatalf("pushed commit must be v2, got %#v", meta)
	}
}

func TestContentSyncServicePrefersHEADv2AndDoesNotDowngradeLegacy(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("v2"), 0o600); err != nil {
		t.Fatal(err)
	}
	v1Dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(v1Dir, "save.dat"), []byte("v1-old"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	game := baseGame(saveDir)
	bstore := newFakeBlobStore()
	v1Meta := setupRemoteStateWithRoutes(t, bstore, game.ID, game, nil, nil, v1Dir, false)
	v1Head := bstore.headsV1[game.ID]
	routes := []domain.Route{{ID: "r-v2", Name: "v2ルート", Order: 0, GameID: game.ID, CreatedAt: now}}
	v2Meta := setupRemoteStateWithRoutes(t, bstore, game.ID, game, nil, routes, saveDir, true)
	if bstore.headsV1[game.ID] != v1Head {
		t.Fatal("writing HEAD.v2 must not change legacy HEAD")
	}

	repo := newFakeRepo(&game, nil)
	svc := newTestService(repo, bstore)
	if _, err := svc.Pull(context.Background(), game.ID, nil, false); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if !repo.replacedRoutes {
		t.Fatal("preferred HEAD.v2 must use v2 apply path")
	}
	if len(repo.upsertedRoutes) != 1 || repo.upsertedRoutes[0].ID != "r-v2" {
		t.Fatalf("expected v2 routes, got %#v", repo.upsertedRoutes)
	}
	// Status も v2 を見る
	fp := contentFingerprint(v2Meta)
	game.LocalSyncHead = &fp
	repo.routes = routes
	detail, err := svc.Status(context.Background(), game.ID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if detail.Status != domain.SyncStatusInSync {
		t.Fatalf("Status = %q, want in_sync against HEAD.v2 (not legacy %s)", detail.Status, contentFingerprint(v1Meta))
	}
}

func TestContentSyncServiceV2PullPropagatesApplyPullResultV2Error(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	game := baseGame(saveDir)
	routes := []domain.Route{{ID: "exists", Name: "ある", Order: 0, GameID: game.ID, CreatedAt: now}}
	bstore := newFakeBlobStore()
	setupRemoteStateWithRoutes(t, bstore, game.ID, game, nil, routes, saveDir, true)

	repo := newFakeRepo(&game, nil)
	repo.applyPullV2Err = fmt.Errorf("missing route reference: currentRouteId=x")
	svc := newTestService(repo, bstore)

	if _, err := svc.Pull(context.Background(), game.ID, nil, false); err == nil {
		t.Fatal("expected Pull to fail when ApplyPullResultV2 rejects")
	}
	if repo.localSyncHeadSet != "" {
		t.Fatal("failed v2 Pull must not update baseline")
	}
}

func TestContentSyncServicePendingPushRecoversAgainstHEADv2(t *testing.T) {
	t.Parallel()

	saveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(saveDir, "save.dat"), []byte("pending-v2"), 0o600); err != nil {
		t.Fatal(err)
	}
	game := baseGame(saveDir)
	repo := newFakeRepo(&game, nil)
	repo.finalizePendingFails = 1
	bstore := newFakeBlobStore()
	svc := newTestService(repo, bstore)

	if err := svc.Push(context.Background(), game.ID, nil); err == nil {
		t.Fatal("Push should fail after HEAD.v2 write when Finalize fails")
	}
	v2Head := bstore.heads[game.ID]
	if v2Head == "" {
		t.Fatal("expected HEAD.v2 written")
	}
	// 旧クライアント残骸があっても回復は preferred HEAD（v2）を見る
	bstore.headsV1[game.ID] = "legacy-stale"

	if err := svc.RecoverPendingPushes(context.Background()); err != nil {
		t.Fatalf("RecoverPendingPushes: %v", err)
	}
	if repo.localSyncHeadSet == "" || len(repo.pending) != 0 {
		t.Fatalf("pending should finalize against HEAD.v2, baseline=%q pending=%#v", repo.localSyncHeadSet, repo.pending)
	}
	if bstore.heads[game.ID] != v2Head {
		t.Fatal("recovery must not change HEAD.v2")
	}
	if bstore.headsV1[game.ID] != "legacy-stale" {
		t.Fatal("recovery must not touch legacy HEAD")
	}
}
