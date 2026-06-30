package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/infrastructure/credentials"
	"CloudLaunch_Go/internal/infrastructure/storage"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ErrOffline はオフラインモード設定時に Push/Pull/DeleteFromCloud が返すセンチネル。
// 呼び出し側はこれを errors.Is で判定し、自動同期パスでは静かにスキップし、
// ユーザー操作パスでは UI に「オフラインモードです」を表示する。
var ErrOffline = errors.New("オフラインモードのため同期しません")

// ProgressFunc はセーブファイルの転送進捗を報告するコールバック。
type ProgressFunc func(current, total int)

// contentBlobStore はS3のブロブ操作を抽象化する（テスト差し替え用）。
type contentBlobStore interface {
	readHEAD(ctx context.Context, gameID string) (string, error)
	writeHEAD(ctx context.Context, gameID, hash string) error
	getBlob(ctx context.Context, gameID, kind, hash string) ([]byte, error)
	putBlob(ctx context.Context, gameID, kind, hash string, data []byte) error
	putBlobs(ctx context.Context, gameID string, blobs map[string][]byte, concurrency int, onProgress func(int, int)) error
	downloadBlobs(ctx context.Context, gameID, saveDir string, blobs map[string]string, concurrency int, onProgress func(int, int)) error
	deleteByPrefix(ctx context.Context, prefix string) error
	listGameIDs(ctx context.Context) ([]string, error)
}

type s3BlobStore struct {
	client *s3.Client
	bucket string
}

func (b *s3BlobStore) readHEAD(ctx context.Context, gameID string) (string, error) {
	return storage.ReadHEAD(ctx, b.client, b.bucket, gameID)
}
func (b *s3BlobStore) writeHEAD(ctx context.Context, gameID, hash string) error {
	return storage.WriteHEAD(ctx, b.client, b.bucket, gameID, hash)
}
func (b *s3BlobStore) getBlob(ctx context.Context, gameID, kind, hash string) ([]byte, error) {
	return storage.GetBlob(ctx, b.client, b.bucket, gameID, kind, hash)
}
func (b *s3BlobStore) putBlob(ctx context.Context, gameID, kind, hash string, data []byte) error {
	return storage.PutBlob(ctx, b.client, b.bucket, gameID, kind, hash, data)
}
func (b *s3BlobStore) putBlobs(ctx context.Context, gameID string, blobs map[string][]byte, concurrency int, onProgress func(int, int)) error {
	return storage.PutBlobs(ctx, b.client, b.bucket, gameID, blobs, concurrency, onProgress)
}
func (b *s3BlobStore) downloadBlobs(ctx context.Context, gameID, saveDir string, blobs map[string]string, concurrency int, onProgress func(int, int)) error {
	return storage.DownloadBlobs(ctx, b.client, b.bucket, gameID, saveDir, blobs, concurrency, onProgress)
}
func (b *s3BlobStore) deleteByPrefix(ctx context.Context, prefix string) error {
	return storage.DeleteObjectsByPrefix(ctx, b.client, b.bucket, prefix)
}
func (b *s3BlobStore) listGameIDs(ctx context.Context) ([]string, error) {
	objects, err := storage.ListObjects(ctx, b.client, b.bucket, "games/")
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	var ids []string
	for _, obj := range objects {
		parts := strings.Split(obj.Key, "/")
		if len(parts) == 3 && parts[0] == "games" && parts[2] == "HEAD" {
			if _, ok := seen[parts[1]]; !ok {
				seen[parts[1]] = struct{}{}
				ids = append(ids, parts[1])
			}
		}
	}
	return ids, nil
}

// ContentSyncService はコンテンツアドレッシングによるゲームデータ同期を提供する。
type ContentSyncService struct {
	config       config.Config
	store        credentials.Store
	repository   ContentSyncRepository
	logger       *slog.Logger
	newBlobStore func(ctx context.Context) (contentBlobStore, error)
	gameLocks    sync.Map // gameID → *sync.Mutex（同一ゲームの Push/Pull/ResolveConflict/DeleteFromCloud を直列化）
	offline      atomic.Bool
}

// SetOfflineMode はオフラインモードの ON/OFF を切り替える。
// ON の間は Push / Pull / DeleteFromCloud が ErrOffline を返し、
// 自動同期（process_monitor 経由）も静かにスキップされる。
func (s *ContentSyncService) SetOfflineMode(enabled bool) {
	s.offline.Store(enabled)
}

// IsOffline は現在のオフラインモード状態を返す。
func (s *ContentSyncService) IsOffline() bool {
	return s.offline.Load()
}

// NewContentSyncService は ContentSyncService を生成する。
func NewContentSyncService(cfg config.Config, store credentials.Store, repo ContentSyncRepository, logger *slog.Logger) *ContentSyncService {
	svc := &ContentSyncService{
		config:     cfg,
		store:      store,
		repository: repo,
		logger:     logger,
	}
	svc.newBlobStore = func(ctx context.Context) (contentBlobStore, error) {
		client, s3cfg, err := svc.newClient(ctx)
		if err != nil {
			return nil, err
		}
		return &s3BlobStore{client: client, bucket: s3cfg.Bucket}, nil
	}
	return svc
}

// lockGame は gameID 単位の排他ロックを取得し、解放関数を返す。
// 同一ゲームに対する Push/Pull/ResolveConflict/DeleteFromCloud を直列化し、
// ローカルファイル操作とリモート HEAD 操作が交錯しないようにする。
// 使い方: defer s.lockGame(gameID)()
func (s *ContentSyncService) lockGame(gameID string) func() {
	m, _ := s.gameLocks.LoadOrStore(gameID, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func (s *ContentSyncService) newClient(ctx context.Context) (*s3.Client, storage.S3Config, error) {
	credential, err := s.store.Load(ctx, "default")
	if err != nil {
		return nil, storage.S3Config{}, fmt.Errorf("認証情報取得に失敗: %w", err)
	}
	if credential == nil {
		return nil, storage.S3Config{}, fmt.Errorf("認証情報が見つかりません")
	}
	cfg := resolveS3Config(s.config, credential)
	client, err := storage.NewClient(ctx, cfg, *credential)
	if err != nil {
		return nil, storage.S3Config{}, fmt.Errorf("S3クライアント作成に失敗: %w", err)
	}
	return client, cfg, nil
}

// contentFingerprint は MetaSnapshot のコンテンツ部分（タイムスタンプ・デバイス名を除く）からハッシュを生成する。
// ローカル変更検出の基準値として使用する。
func contentFingerprint(meta domain.MetaSnapshot) string {
	type fp struct {
		G string `json:"g"`
		S string `json:"s"`
		V string `json:"v"`
	}
	data, _ := json.Marshal(fp{G: meta.GameJSON, S: meta.SessionsJSON, V: meta.Saves})
	return hashBytes(data)
}

func (s *ContentSyncService) getOrInitDeviceName(ctx context.Context) (string, error) {
	name, err := s.repository.GetSetting(ctx, "device_name")
	if err != nil {
		return "", err
	}
	if name != "" {
		return name, nil
	}
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "Unknown Device"
	}
	// 保存に失敗しても致命的ではない（次回再取得・再保存される）が、
	// 書き込みエラーを完全に握り潰さないようログに残す。
	if err := s.repository.UpsertSetting(ctx, "device_name", hostname); err != nil {
		s.logger.Warn("device_name の保存に失敗", "error", err)
	}
	return hostname, nil
}

// buildLocalMeta はゲームの現在のローカル状態から MetaSnapshot を構築する。
func (s *ContentSyncService) buildLocalMeta(ctx context.Context, game domain.Game, saveFolderPath string) (metaBuildResult, error) {
	sessions, err := s.repository.ListPlaySessionsByGame(ctx, game.ID)
	if err != nil {
		return metaBuildResult{}, err
	}
	deviceName, err := s.getOrInitDeviceName(ctx)
	if err != nil {
		return metaBuildResult{}, err
	}
	imageHash := ""
	if game.ImagePath != nil && *game.ImagePath != "" {
		if h, herr := hashFileStream(*game.ImagePath); herr == nil {
			imageHash = h
		} else {
			// 画像が一時的に読めない場合 imageHash="" のままになり、fingerprint が
			// リモートとズレて偽の差分（PushNeeded）に見える。原因追跡のため記録する。
			s.logger.Warn("画像のハッシュ計算に失敗（imageHash を空として扱う）", "gameId", game.ID, "path", *game.ImagePath, "error", herr)
		}
	}
	saveSnap, err := buildSaveTree(saveFolderPath)
	if err != nil {
		return metaBuildResult{}, err
	}
	saveSnapJSON, err := json.Marshal(saveSnap)
	if err != nil {
		return metaBuildResult{}, err
	}
	savesHash := hashBytes(saveSnapJSON)
	return buildMetaSnapshot(game, sessions, imageHash, savesHash, deviceName)
}

// Status は現在の同期状態を返す。
func (s *ContentSyncService) Status(ctx context.Context, gameID string) (domain.SyncStatusDetail, error) {
	bstore, err := s.newBlobStore(ctx)
	if err != nil {
		return domain.SyncStatusDetail{}, err
	}

	remoteHead, err := bstore.readHEAD(ctx, gameID)
	if err != nil {
		return domain.SyncStatusDetail{}, err
	}
	if remoteHead == "" {
		return domain.SyncStatusDetail{Status: domain.SyncStatusNeverSynced}, nil
	}

	remoteMetaBytes, err := bstore.getBlob(ctx, gameID, storage.BlobKindCommit, remoteHead)
	if err != nil {
		return domain.SyncStatusDetail{}, err
	}
	var remoteMeta domain.MetaSnapshot
	if err := json.Unmarshal(remoteMetaBytes, &remoteMeta); err != nil {
		return domain.SyncStatusDetail{}, err
	}

	game, err := s.repository.GetGameByID(ctx, gameID)
	if err != nil {
		return domain.SyncStatusDetail{}, err
	}
	if game == nil {
		return domain.SyncStatusDetail{}, fmt.Errorf("ゲームが見つかりません: %s", gameID)
	}
	if game.SaveFolderPath == nil || *game.SaveFolderPath == "" {
		return domain.SyncStatusDetail{}, fmt.Errorf("セーブフォルダのパスが未設定です")
	}

	localMeta, err := s.buildLocalMeta(ctx, *game, *game.SaveFolderPath)
	if err != nil {
		return domain.SyncStatusDetail{}, err
	}

	currentLocalHash := contentFingerprint(localMeta.Snapshot)
	localSyncHead := ""
	if game.LocalSyncHead != nil {
		localSyncHead = *game.LocalSyncHead
	}
	remoteContentHash := contentFingerprint(remoteMeta)

	var status domain.SyncStatus
	switch {
	case localSyncHead == "":
		// このPCでまだ一度も同期していない（LocalSyncHead未設定）。
		// ベースラインがないので比較できず、リモートを正として pull_needed とする。
		status = domain.SyncStatusPullNeeded
	case currentLocalHash == localSyncHead && remoteContentHash == localSyncHead:
		status = domain.SyncStatusInSync
	case currentLocalHash != localSyncHead && remoteContentHash == localSyncHead:
		status = domain.SyncStatusPushNeeded
	case currentLocalHash == localSyncHead && remoteContentHash != localSyncHead:
		status = domain.SyncStatusPullNeeded
	default:
		status = domain.SyncStatusConflict
	}

	detail := domain.SyncStatusDetail{Status: status}
	if status == domain.SyncStatusConflict {
		snap := localMeta.Snapshot
		detail.LocalMeta = &snap
		detail.RemoteMeta = &remoteMeta
	}
	return detail, nil
}

// Push はローカルデータをリモートにアップロードする。同一ゲームの同期と直列化される。
// オフラインモード時は ErrOffline を返す（自動同期は process_monitor 側で握りつぶす）。
func (s *ContentSyncService) Push(ctx context.Context, gameID string, onProgress ProgressFunc) error {
	if s.offline.Load() {
		return ErrOffline
	}
	defer s.lockGame(gameID)()
	return s.push(ctx, gameID, onProgress, false)
}

func (s *ContentSyncService) push(ctx context.Context, gameID string, onProgress ProgressFunc, force bool) error {
	bstore, err := s.newBlobStore(ctx)
	if err != nil {
		return err
	}

	game, err := s.repository.GetGameByID(ctx, gameID)
	if err != nil {
		return err
	}
	if game == nil {
		return fmt.Errorf("ゲームが見つかりません: %s", gameID)
	}
	if game.SaveFolderPath == nil || *game.SaveFolderPath == "" {
		return fmt.Errorf("セーブフォルダのパスが未設定です")
	}
	// !force のとき、push 開始時点のリモート HEAD を控える（writeHEAD 直前の再確認に使う）。
	expectedHead, err := s.pushCheckRemoteHead(ctx, bstore, gameID, game, force)
	if err != nil {
		return err
	}

	meta, saveSnapJSON, savesHash, saveBlobs, imageHash, imageData, err := s.pushBuildLocalMeta(ctx, gameID, game)
	if err != nil {
		return err
	}
	metaHash := hashBytes(meta.SnapshotBytes)

	if err := s.pushUploadBlobs(ctx, bstore, gameID, onProgress, meta, saveSnapJSON, savesHash, saveBlobs, imageHash, imageData, metaHash); err != nil {
		return err
	}

	return s.pushFinalizeHead(ctx, bstore, gameID, force, expectedHead, metaHash, meta, saveSnapJSON)
}

// pushCheckRemoteHead は !force のとき push 開始時点のリモート HEAD を確認し、
// ローカルの同期基準とズレていればコンフリクトとして中断する。
// 戻り値の expectedHead は writeHEAD 直前の再確認（pushFinalizeHead）で使う。
func (s *ContentSyncService) pushCheckRemoteHead(ctx context.Context, bstore contentBlobStore, gameID string, game *domain.Game, force bool) (string, error) {
	if force {
		return "", nil
	}
	remoteHead, err := bstore.readHEAD(ctx, gameID)
	if err != nil {
		return "", err
	}
	if remoteHead == "" {
		return "", nil
	}
	remoteMetaBytes, err := bstore.getBlob(ctx, gameID, storage.BlobKindCommit, remoteHead)
	if err != nil {
		return "", err
	}
	var remoteMeta domain.MetaSnapshot
	if err := json.Unmarshal(remoteMetaBytes, &remoteMeta); err != nil {
		return "", err
	}
	localSyncHead := ""
	if game.LocalSyncHead != nil {
		localSyncHead = *game.LocalSyncHead
	}
	if contentFingerprint(remoteMeta) != localSyncHead {
		return "", fmt.Errorf("リモートが更新されています。同期状態を確認してコンフリクトを解決してください")
	}
	return remoteHead, nil
}

// pushBuildLocalMeta はゲームの現在のローカル状態から push 用の MetaSnapshot と
// アップロード対象（セーブスナップショット・差分ブロブ・画像）を構築する。
func (s *ContentSyncService) pushBuildLocalMeta(ctx context.Context, gameID string, game *domain.Game) (metaBuildResult, []byte, domain.BlobHash, map[string][]byte, domain.BlobHash, []byte, error) {
	sessions, err := s.repository.ListPlaySessionsByGame(ctx, gameID)
	if err != nil {
		return metaBuildResult{}, nil, "", nil, "", nil, err
	}
	deviceName, err := s.getOrInitDeviceName(ctx)
	if err != nil {
		return metaBuildResult{}, nil, "", nil, "", nil, err
	}

	saveSnap, saveBlobs, err := buildSaveSnapshot(*game.SaveFolderPath)
	if err != nil {
		return metaBuildResult{}, nil, "", nil, "", nil, err
	}
	saveSnapJSON, err := json.Marshal(saveSnap)
	if err != nil {
		return metaBuildResult{}, nil, "", nil, "", nil, err
	}
	savesHash := hashBytes(saveSnapJSON)

	imageHash := ""
	var imageData []byte
	if game.ImagePath != nil && *game.ImagePath != "" {
		h, data, herr := hashFile(*game.ImagePath)
		if herr == nil {
			imageHash = h
			imageData = data
		} else {
			// 画像欠損のまま push すると imageHash="" の game.json が確定し、
			// この状態が新しいリモート HEAD になる。意図せぬ画像消失を追跡できるよう記録する。
			s.logger.Warn("画像のハッシュ計算に失敗（imageHash を空として push）", "gameId", game.ID, "path", *game.ImagePath, "error", herr)
		}
	}

	meta, err := buildMetaSnapshot(*game, sessions, imageHash, savesHash, deviceName)
	if err != nil {
		return metaBuildResult{}, nil, "", nil, "", nil, err
	}
	return meta, saveSnapJSON, savesHash, saveBlobs, imageHash, imageData, nil
}

// pushUploadBlobs はセーブブロブ・セーブスナップショット・画像・game.json・sessions.json・
// コミットブロブを HEAD 書き換え前にアップロードする。
func (s *ContentSyncService) pushUploadBlobs(ctx context.Context, bstore contentBlobStore, gameID string, onProgress ProgressFunc, meta metaBuildResult, saveSnapJSON []byte, savesHash domain.BlobHash, saveBlobs map[string][]byte, imageHash domain.BlobHash, imageData []byte, metaHash domain.BlobHash) error {
	// セーブファイルをアップロード（既存ブロブ一括確認 + 差分並列アップロード）
	if err := bstore.putBlobs(ctx, gameID, saveBlobs, s.config.S3UploadConcurrency, onProgress); err != nil {
		return err
	}

	// セーブスナップショット・画像・game.json・sessions.json をアップロード
	if err := bstore.putBlob(ctx, gameID, storage.BlobKindTree, savesHash, saveSnapJSON); err != nil {
		return err
	}
	if imageHash != "" && imageData != nil {
		if err := bstore.putBlob(ctx, gameID, storage.BlobKindObject, imageHash, imageData); err != nil {
			return err
		}
	}
	if err := bstore.putBlob(ctx, gameID, storage.BlobKindMeta, meta.Snapshot.GameJSON, meta.GameJSON); err != nil {
		return err
	}
	if err := bstore.putBlob(ctx, gameID, storage.BlobKindMeta, meta.Snapshot.SessionsJSON, meta.SessionsJSON); err != nil {
		return err
	}
	if err := bstore.putBlob(ctx, gameID, storage.BlobKindCommit, metaHash, meta.SnapshotBytes); err != nil {
		return err
	}
	return nil
}

// pushFinalizeHead は HEAD 書き換え直前の再確認・HEAD 書き換え・ローカル同期基準の更新を行う。
func (s *ContentSyncService) pushFinalizeHead(ctx context.Context, bstore contentBlobStore, gameID string, force bool, expectedHead string, metaHash domain.BlobHash, meta metaBuildResult, saveSnapJSON []byte) error {
	// HEAD 書き換え直前に再度リモート HEAD を確認し、push 開始時から変化していれば中断する。
	// S3 に CAS が無いため完全な排他はできないが、アップロード中に別デバイスが push した場合の
	// ロストアップデートの窓を大幅に縮小する（force 時はユーザーが上書きを選択済みのため省略）。
	if !force {
		currentHead, err := bstore.readHEAD(ctx, gameID)
		if err != nil {
			return err
		}
		if currentHead != expectedHead {
			return fmt.Errorf("リモートが更新されています。同期状態を確認してコンフリクトを解決してください")
		}
	}
	if err := bstore.writeHEAD(ctx, gameID, metaHash); err != nil {
		return err
	}
	if err := s.repository.SetLocalSyncHead(ctx, gameID, contentFingerprint(meta.Snapshot)); err != nil {
		return err
	}
	// 次回 Pull の base tree として、今 push したローカルのスナップショットを保存する。
	return s.repository.SetLocalSaveTree(ctx, gameID, string(saveSnapJSON))
}

// Pull はリモートデータをローカルに適用する。同一ゲームの同期と直列化される。
// 詳細な挙動（deleteUntracked による未追跡ファイル削除確認）は内部の pull を参照。
// オフラインモード時は ErrOffline を返す。
func (s *ContentSyncService) Pull(ctx context.Context, gameID string, onProgress ProgressFunc, deleteUntracked bool) (domain.PullResult, error) {
	if s.offline.Load() {
		return domain.PullResult{}, ErrOffline
	}
	defer s.lockGame(gameID)()
	return s.pull(ctx, gameID, onProgress, deleteUntracked)
}

// pull はリモートデータをローカルに適用する。
//
// deleteUntracked が false の場合、同期が認識していないローカル固有ファイル
// （untracked）を削除する必要があると分かった時点で、ローカルに一切変更を加えずに
// PullResult{Applied:false, UntrackedDeletes:...} を返す。呼び出し側でユーザーに
// 確認を取り、承認後に deleteUntracked=true で再実行する。
func (s *ContentSyncService) pull(ctx context.Context, gameID string, onProgress ProgressFunc, deleteUntracked bool) (domain.PullResult, error) {
	bstore, err := s.newBlobStore(ctx)
	if err != nil {
		return domain.PullResult{}, err
	}

	remoteHead, err := bstore.readHEAD(ctx, gameID)
	if err != nil {
		return domain.PullResult{}, err
	}
	if remoteHead == "" {
		return domain.PullResult{}, fmt.Errorf("リモートにデータがありません")
	}

	metaBytes, err := bstore.getBlob(ctx, gameID, storage.BlobKindCommit, remoteHead)
	if err != nil {
		return domain.PullResult{}, err
	}
	var meta domain.MetaSnapshot
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return domain.PullResult{}, err
	}

	saveSnapBytes, err := bstore.getBlob(ctx, gameID, storage.BlobKindTree, meta.Saves)
	if err != nil {
		return domain.PullResult{}, err
	}
	var saveSnap domain.SaveSnapshot
	if err := json.Unmarshal(saveSnapBytes, &saveSnap); err != nil {
		return domain.PullResult{}, err
	}

	gameJSONBytes, err := bstore.getBlob(ctx, gameID, storage.BlobKindMeta, meta.GameJSON)
	if err != nil {
		return domain.PullResult{}, err
	}
	var cloudG cloudGame
	if err := json.Unmarshal(gameJSONBytes, &cloudG); err != nil {
		return domain.PullResult{}, err
	}
	if cloudG.ID != gameID {
		return domain.PullResult{}, fmt.Errorf("リモートのゲームIDが一致しません: %s", cloudG.ID)
	}

	sessionsJSONBytes, err := bstore.getBlob(ctx, gameID, storage.BlobKindMeta, meta.SessionsJSON)
	if err != nil {
		return domain.PullResult{}, err
	}
	var cloudSessions []cloudSession
	if err := json.Unmarshal(sessionsJSONBytes, &cloudSessions); err != nil {
		return domain.PullResult{}, err
	}

	// ローカルゲームのマシン固有フィールドを保持
	localGame, err := s.repository.GetGameByID(ctx, gameID)
	if err != nil {
		return domain.PullResult{}, err
	}
	exePath := UnconfiguredExePath
	saveFolderPath := (*string)(nil)
	imagePath := (*string)(nil)
	if localGame != nil {
		exePath = localGame.ExePath
		saveFolderPath = localGame.SaveFolderPath
		imagePath = localGame.ImagePath
	}

	// ── 削除計画を先に立て、未追跡ファイルの削除が必要なら変更前に確認へ回す ──
	// ローカルに副作用を与える前（画像・セーブのダウンロード前）に判定することで、
	// 「確認待ち」を返したときはディスクが一切変更されていないことを保証する。
	trackedDeletes, untrackedDeletes, err := s.pullPlanDeletions(ctx, gameID, saveFolderPath, saveSnap)
	if err != nil {
		return domain.PullResult{}, err
	}
	if len(untrackedDeletes) > 0 && !deleteUntracked {
		// 同期が知らないローカル固有ファイルを消そうとしている。確認を仰ぐ（ここまで無変更）。
		return domain.PullResult{Applied: false, UntrackedDeletes: untrackedDeletes}, nil
	}

	// 画像をダウンロード（ローカルと異なる場合のみ）
	imagePath, err = s.pullDownloadImage(ctx, bstore, gameID, cloudG, imagePath)
	if err != nil {
		return domain.PullResult{}, err
	}

	// セーブファイルをダウンロード（ローカルハッシュ比較で差分のみ並列ダウンロード）
	if err := s.pullDownloadSaves(ctx, bstore, gameID, onProgress, saveFolderPath, saveSnap, trackedDeletes, untrackedDeletes); err != nil {
		return domain.PullResult{}, err
	}

	return s.pullApplyToDB(ctx, gameID, cloudG, cloudSessions, imagePath, exePath, saveFolderPath, localGame, meta, saveSnapBytes)
}

// pullPlanDeletions はリモートのセーブスナップショットとローカルの base tree を突き合わせ、
// 削除すべき tracked / untracked ファイルの一覧を返す。ディスクには一切変更を加えない。
func (s *ContentSyncService) pullPlanDeletions(ctx context.Context, gameID string, saveFolderPath *string, saveSnap domain.SaveSnapshot) ([]string, []string, error) {
	var trackedDeletes, untrackedDeletes []string
	if saveFolderPath != nil && *saveFolderPath != "" {
		if _, statErr := os.Stat(*saveFolderPath); statErr == nil {
			baseTreeJSON, terr := s.repository.GetLocalSaveTree(ctx, gameID)
			if terr != nil {
				return nil, nil, terr
			}
			baseTree, terr := parseSaveTree(baseTreeJSON)
			if terr != nil {
				return nil, nil, terr
			}
			var err error
			trackedDeletes, untrackedDeletes, err = planDeletions(*saveFolderPath, saveSnap, baseTree)
			if err != nil {
				return nil, nil, err
			}
		} else if !os.IsNotExist(statErr) {
			return nil, nil, statErr
		}
	}
	return trackedDeletes, untrackedDeletes, nil
}

// pullDownloadImage はリモート画像がローカルと異なる場合のみダウンロードし、
// 更新後の imagePath を返す（変更が不要なら受け取った imagePath をそのまま返す）。
func (s *ContentSyncService) pullDownloadImage(ctx context.Context, bstore contentBlobStore, gameID string, cloudG cloudGame, imagePath *string) (*string, error) {
	if cloudG.ImageHash != "" {
		localImageHash := ""
		if imagePath != nil && *imagePath != "" {
			if h, herr := hashFileStream(*imagePath); herr == nil {
				localImageHash = h
			}
		}
		if localImageHash != cloudG.ImageHash {
			imageData, berr := bstore.getBlob(ctx, gameID, storage.BlobKindObject, cloudG.ImageHash)
			if berr != nil {
				return nil, berr
			}
			contentType := http.DetectContentType(imageData)
			ext := normalizeImageExt("", contentType)
			imgPath := filepath.Join(s.config.AppDataDir, "thumbnails",
				fmt.Sprintf("%s_%s%s", cloudG.ImageHash, gameID, ext))
			if err := os.MkdirAll(filepath.Dir(imgPath), 0o700); err != nil {
				return nil, err
			}
			if err := os.WriteFile(imgPath, imageData, 0o600); err != nil {
				return nil, err
			}
			imagePath = &imgPath
		}
	}
	return imagePath, nil
}

// pullDownloadSaves はセーブファイルの差分を並列ダウンロードし、計画済みの削除を適用する。
func (s *ContentSyncService) pullDownloadSaves(ctx context.Context, bstore contentBlobStore, gameID string, onProgress ProgressFunc, saveFolderPath *string, saveSnap domain.SaveSnapshot, trackedDeletes, untrackedDeletes []string) error {
	if saveFolderPath != nil && *saveFolderPath != "" {
		saveDir := *saveFolderPath
		total := len(saveSnap.Files)

		if err := os.MkdirAll(saveDir, 0o700); err != nil {
			return err
		}

		needsDownload := make(map[string]string, total)
		for relPath, hash := range saveSnap.Files {
			targetPath, err := storage.ResolveSafeRelativePath(saveDir, relPath)
			if err != nil {
				return err
			}
			localHash, err := hashFileStream(targetPath)
			if err != nil || localHash != hash {
				needsDownload[relPath] = hash
			}
		}

		var wrappedProgress func(int, int)
		if onProgress != nil {
			alreadyDone := total - len(needsDownload)
			onProgress(alreadyDone, total)
			wrappedProgress = func(downloaded, _ int) {
				onProgress(alreadyDone+downloaded, total)
			}
		}
		if err := bstore.downloadBlobs(ctx, gameID, saveDir, needsDownload, s.config.S3UploadConcurrency, wrappedProgress); err != nil {
			return err
		}

		// tracked は常に削除、untracked はここに来た時点で deleteUntracked=true のときのみ含む。
		toDelete := append(append([]string{}, trackedDeletes...), untrackedDeletes...)
		if err := applyDeletions(saveDir, toDelete); err != nil {
			return err
		}
		if len(toDelete) > 0 {
			s.logger.Warn("Pull によりローカルのセーブファイルを削除しました",
				"gameId", gameID, "saveDir", saveDir,
				"tracked", len(trackedDeletes), "untracked", len(untrackedDeletes),
				"files", logSamplePaths(toDelete, 20))
		}
	} else {
		s.logger.Warn("セーブフォルダ未設定のためセーブデータをスキップします", "gameId", gameID)
	}
	return nil
}

// pullApplyToDB はリモートのゲーム情報・セッション・同期基準・base tree を単一トランザクションで反映する。
// localGame はマシン固有フィールド（LocalSaveHash / LocalSaveHashUpdatedAt 等）の引き継ぎに使う。
func (s *ContentSyncService) pullApplyToDB(ctx context.Context, gameID string, cloudG cloudGame, cloudSessions []cloudSession, imagePath *string, exePath string, saveFolderPath *string, localGame *domain.Game, meta domain.MetaSnapshot, saveSnapBytes []byte) (domain.PullResult, error) {
	// ゲーム情報・セッション・localSyncHead・localSaveTree を単一トランザクションで反映する。
	// 部分失敗による DB 不整合と、同期対象外 Route 参照による FK 違反を防ぐ。
	updatedGame := domain.Game{
		ID:             cloudG.ID,
		Title:          cloudG.Title,
		Publisher:      cloudG.Publisher,
		ImagePath:      imagePath,
		ExePath:        exePath,
		SaveFolderPath: saveFolderPath,
		PlayStatus:     cloudG.PlayStatus,
		TotalPlayTime:  cloudG.TotalPlayTime,
		LastPlayed:     cloudG.LastPlayed,
		ClearedAt:      cloudG.ClearedAt,
		CurrentRouteID: cloudG.CurrentRouteID,
		CreatedAt:      cloudG.CreatedAt,
		UpdatedAt:      cloudG.UpdatedAt,
	}
	// マシン固有フィールド（process_monitor.saveSession が書き込む LocalSaveHash 等）は
	// ApplyPullResult が ON CONFLICT DO UPDATE で excluded.* を書くため、ここで明示的に
	// 引き継がないと毎回 NULL に潰される。UI の「最終同期」表示が壊れる原因になっていた。
	if localGame != nil {
		updatedGame.LocalSaveHash = localGame.LocalSaveHash
		updatedGame.LocalSaveHashUpdatedAt = localGame.LocalSaveHashUpdatedAt
	}
	sessions := make([]domain.PlaySession, 0, len(cloudSessions))
	for _, cs := range cloudSessions {
		sessions = append(sessions, domain.PlaySession{
			ID:          cs.ID,
			GameID:      gameID,
			PlayedAt:    cs.PlayedAt,
			Duration:    cs.Duration,
			SessionName: cs.SessionName,
			RouteID:     cs.RouteID,
			UpdatedAt:   cs.UpdatedAt,
		})
	}
	// 次回 Pull の base tree として、今適用した（リモートの）スナップショットを保存する。
	if err := s.repository.ApplyPullResult(ctx, updatedGame, sessions, contentFingerprint(meta), string(saveSnapBytes)); err != nil {
		return domain.PullResult{}, err
	}
	return domain.PullResult{Applied: true}, nil
}

// ResolveConflict はコンフリクトを手動解決する。同一ゲームの同期と直列化される。
// useLocal=false（リモート採用）は Pull と同様に未追跡ファイルの削除確認を経由する。
// オフラインモード時は ErrOffline を返す。
func (s *ContentSyncService) ResolveConflict(ctx context.Context, gameID string, useLocal, deleteUntracked bool) (domain.PullResult, error) {
	if s.offline.Load() {
		return domain.PullResult{}, ErrOffline
	}
	defer s.lockGame(gameID)()
	if useLocal {
		if err := s.push(ctx, gameID, nil, true); err != nil {
			return domain.PullResult{}, err
		}
		return domain.PullResult{Applied: true}, nil
	}
	return s.pull(ctx, gameID, nil, deleteUntracked)
}

// DeleteFromCloud はゲームのリモートデータを削除し、ローカルの同期基準もクリアする。同一ゲームの同期と直列化される。
// オフラインモード時は ErrOffline を返す。
func (s *ContentSyncService) DeleteFromCloud(ctx context.Context, gameID string) error {
	if s.offline.Load() {
		return ErrOffline
	}
	defer s.lockGame(gameID)()
	bstore, err := s.newBlobStore(ctx)
	if err != nil {
		return err
	}
	prefix := fmt.Sprintf("games/%s/", gameID)
	if err := bstore.deleteByPrefix(ctx, prefix); err != nil {
		return err
	}
	// リモート削除後はローカルの同期基準（localSyncHead/localSaveTree）を無効化して状態を
	// 一貫させる。残しても Status は remoteHead=="" を先に判定するため誤判定はしないが、
	// 古い基準が残るのを避ける。削除は成功済みなのでクリア失敗は致命扱いせずログのみ。
	if err := s.repository.SetLocalSyncHead(ctx, gameID, ""); err != nil {
		s.logger.Warn("localSyncHead のクリアに失敗", "gameId", gameID, "error", err)
	}
	if err := s.repository.SetLocalSaveTree(ctx, gameID, ""); err != nil {
		s.logger.Warn("localSaveTree のクリアに失敗", "gameId", gameID, "error", err)
	}
	return nil
}

// fanOutGames は gameIDs をまたいで fn を最大 concurrency 並列で実行し、
// nil でない結果を gameIDs の順序で集めて返す。concurrency<=0 のときは既定値 6 を使う。
func fanOutGames[T any](gameIDs []string, concurrency int, fn func(gameID string) *T) []T {
	if concurrency <= 0 {
		concurrency = 6
	}
	if concurrency > len(gameIDs) {
		concurrency = len(gameIDs)
	}
	results := make([]*T, len(gameIDs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	for i, gameID := range gameIDs {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, id string) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = fn(id)
		}(i, gameID)
	}
	wg.Wait()

	out := make([]T, 0, len(gameIDs))
	for _, r := range results {
		if r != nil {
			out = append(out, *r)
		}
	}
	return out
}

// LoadCloudMetadata はクラウド上の全ゲームのメタ情報を返す。
// ゲームごとの取得（readHEAD + commit + game.json）を S3UploadConcurrency 並列で実行する。
// 取得・解析に失敗したゲームは警告ログを出してスキップする。結果は gameIDs の順序を保つ。
func (s *ContentSyncService) LoadCloudMetadata(ctx context.Context) ([]CloudGameInfo, error) {
	bstore, err := s.newBlobStore(ctx)
	if err != nil {
		return nil, err
	}
	gameIDs, err := bstore.listGameIDs(ctx)
	if err != nil {
		return nil, err
	}
	if len(gameIDs) == 0 {
		return nil, nil
	}
	return fanOutGames(gameIDs, s.config.S3UploadConcurrency, func(id string) *CloudGameInfo {
		return s.loadCloudGameInfo(ctx, bstore, id)
	}), nil
}

// CloudLogicalFile はクラウド上の論理セーブファイル1件を表す。
type CloudLogicalFile struct {
	RelPath string `json:"relPath"`
	Size    int64  `json:"size"`
}

// CloudGameView は1ゲームのクラウド論理セーブビュー（最新コミットから復元したファイル一覧）を表す。
type CloudGameView struct {
	GameID       string             `json:"gameId"`
	Title        string             `json:"title"`
	Files        []CloudLogicalFile `json:"files"`
	TotalSize    int64              `json:"totalSize"`
	LastModified time.Time          `json:"lastModified"`
}

// GetCloudGameView は1ゲームの最新コミットから論理セーブファイル一覧を復元する。
// HEAD 未設定や解析失敗時は (nil, nil)（=クラウドデータ無し扱い）を返す。エラーは取得失敗時のみ返す。
func (s *ContentSyncService) GetCloudGameView(ctx context.Context, gameID string) (*CloudGameView, error) {
	client, cfg, err := s.newClient(ctx)
	if err != nil {
		return nil, err
	}
	bstore := &s3BlobStore{client: client, bucket: cfg.Bucket}
	return s.buildCloudGameView(ctx, bstore, gameID)
}

// loadCloudCommit は gameID の HEAD からコミット（MetaSnapshot）と game.json 由来のタイトルを読む。
// HEAD 未設定やコミット解析失敗時は (nil, "", nil) を返す（クラウドデータ無し扱い）。
// タイトル取得に失敗した場合は gameID にフォールバックする。
func (s *ContentSyncService) loadCloudCommit(ctx context.Context, bstore contentBlobStore, gameID string) (*domain.MetaSnapshot, string, error) {
	head, err := bstore.readHEAD(ctx, gameID)
	if err != nil {
		return nil, "", err
	}
	if head == "" {
		return nil, "", nil
	}

	metaBytes, err := bstore.getBlob(ctx, gameID, storage.BlobKindCommit, head)
	if err != nil {
		return nil, "", err
	}
	var meta domain.MetaSnapshot
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		s.logger.Warn("コミットブロブ解析失敗", "gameId", gameID, "error", err)
		return nil, "", nil
	}

	// Title: game.json から取得。失敗時は gameID にフォールバックする。
	title := gameID
	if meta.GameJSON != "" {
		if gameJSONBytes, gerr := bstore.getBlob(ctx, gameID, storage.BlobKindMeta, meta.GameJSON); gerr == nil {
			var cg cloudGame
			if json.Unmarshal(gameJSONBytes, &cg) == nil && cg.Title != "" {
				title = cg.Title
			}
		} else {
			s.logger.Warn("game.json取得失敗（Titleをフォールバック）", "gameId", gameID, "error", gerr)
		}
	}
	return &meta, title, nil
}

// buildCloudGameView は与えられた blob store を使って1ゲームの論理ビューを復元する。
// サイズ解決のため bstore は内部に S3 クライアントとバケットを保持している必要がある。
func (s *ContentSyncService) buildCloudGameView(ctx context.Context, bstore *s3BlobStore, gameID string) (*CloudGameView, error) {
	meta, title, err := s.loadCloudCommit(ctx, bstore, gameID)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, nil
	}

	// saves tree から relPath→hash を取得する。
	var saveSnap domain.SaveSnapshot
	if meta.Saves != "" {
		saveSnapBytes, terr := bstore.getBlob(ctx, gameID, storage.BlobKindTree, meta.Saves)
		if terr != nil {
			return nil, terr
		}
		if err := json.Unmarshal(saveSnapBytes, &saveSnap); err != nil {
			s.logger.Warn("セーブツリー解析失敗", "gameId", gameID, "error", err)
			return nil, nil
		}
	}

	// サイズ解決: objects プレフィックスを1回列挙して hash→size マップを作る。
	sizeMap := make(map[string]int64)
	if len(saveSnap.Files) > 0 {
		prefix := fmt.Sprintf("games/%s/%s/", gameID, storage.BlobKindObject)
		objects, oerr := storage.ListObjects(ctx, bstore.client, bstore.bucket, prefix)
		if oerr != nil {
			return nil, oerr
		}
		for _, obj := range objects {
			hash := obj.Key
			if idx := strings.LastIndex(hash, "/"); idx >= 0 {
				hash = hash[idx+1:]
			}
			sizeMap[hash] = obj.Size
		}
	}

	files := make([]CloudLogicalFile, 0, len(saveSnap.Files))
	var totalSize int64
	for relPath, hash := range saveSnap.Files {
		size := sizeMap[hash]
		totalSize += size
		files = append(files, CloudLogicalFile{RelPath: relPath, Size: size})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].RelPath < files[j].RelPath })

	return &CloudGameView{
		GameID:       gameID,
		Title:        title,
		Files:        files,
		TotalSize:    totalSize,
		LastModified: meta.CreatedAt,
	}, nil
}

// ListCloudGameViews は全ゲームの論理ビューを返す（Title 昇順）。
// 個別ゲームの取得に失敗した場合は警告ログを出してスキップする。
func (s *ContentSyncService) ListCloudGameViews(ctx context.Context) ([]CloudGameView, error) {
	client, cfg, err := s.newClient(ctx)
	if err != nil {
		return nil, err
	}
	bstore := &s3BlobStore{client: client, bucket: cfg.Bucket}

	gameIDs, err := bstore.listGameIDs(ctx)
	if err != nil {
		return nil, err
	}

	views := fanOutGames(gameIDs, s.config.S3UploadConcurrency, func(id string) *CloudGameView {
		view, verr := s.buildCloudGameView(ctx, bstore, id)
		if verr != nil {
			s.logger.Warn("クラウド論理ビュー復元失敗（スキップ）", "gameId", id, "error", verr)
			return nil
		}
		return view
	})
	sort.Slice(views, func(i, j int) bool { return views[i].Title < views[j].Title })
	return views, nil
}

// CloudGameSummary は1ゲームの軽量サマリ（ファイル一覧・サイズを含まない）を表す。
// クラウドデータ管理の初期表示ではタイトル一覧のみが必要なため、
// セーブツリーの解析や objects の列挙（サイズ解決）は行わない。
type CloudGameSummary struct {
	GameID       string    `json:"gameId"`
	Title        string    `json:"title"`
	LastModified time.Time `json:"lastModified"`
}

// buildCloudGameSummary は HEAD→commit→game.json のみを読み取り、軽量サマリを復元する。
// buildCloudGameView と異なりセーブツリーの解析・objects の列挙は行わないため高速。
func (s *ContentSyncService) buildCloudGameSummary(ctx context.Context, bstore *s3BlobStore, gameID string) (*CloudGameSummary, error) {
	meta, title, err := s.loadCloudCommit(ctx, bstore, gameID)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, nil
	}
	return &CloudGameSummary{GameID: gameID, Title: title, LastModified: meta.CreatedAt}, nil
}

// ListCloudGameSummaries は全ゲームの軽量サマリ（Title 昇順）を返す。
// ファイル数・サイズは含まず、各ゲームの詳細は GetCloudGameView で個別に遅延取得する。
func (s *ContentSyncService) ListCloudGameSummaries(ctx context.Context) ([]CloudGameSummary, error) {
	client, cfg, err := s.newClient(ctx)
	if err != nil {
		return nil, err
	}
	bstore := &s3BlobStore{client: client, bucket: cfg.Bucket}

	gameIDs, err := bstore.listGameIDs(ctx)
	if err != nil {
		return nil, err
	}

	// 各ゲームは小さな GET を3回行うのみ。並列度を絞り全タイトルをまとめて取得する。
	const maxConcurrency = 8
	summaries := fanOutGames(gameIDs, maxConcurrency, func(id string) *CloudGameSummary {
		summary, verr := s.buildCloudGameSummary(ctx, bstore, id)
		if verr != nil {
			s.logger.Warn("クラウドサマリ復元失敗（スキップ）", "gameId", id, "error", verr)
			return nil
		}
		return summary
	})
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].Title < summaries[j].Title })
	return summaries, nil
}

// loadCloudGameInfo は1ゲームのクラウドメタ情報を取得する。
// HEAD 未設定や取得・解析失敗時は（必要なら警告ログを出して）nil を返す。
func (s *ContentSyncService) loadCloudGameInfo(ctx context.Context, bstore contentBlobStore, gameID string) *CloudGameInfo {
	head, err := bstore.readHEAD(ctx, gameID)
	if err != nil || head == "" {
		return nil
	}
	metaBytes, err := bstore.getBlob(ctx, gameID, storage.BlobKindCommit, head)
	if err != nil {
		s.logger.Warn("コミットブロブ取得失敗", "gameId", gameID, "error", err)
		return nil
	}
	var meta domain.MetaSnapshot
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		s.logger.Warn("コミットブロブ解析失敗", "gameId", gameID, "error", err)
		return nil
	}
	gameJSONBytes, err := bstore.getBlob(ctx, gameID, storage.BlobKindMeta, meta.GameJSON)
	if err != nil {
		s.logger.Warn("game.json取得失敗", "gameId", gameID, "error", err)
		return nil
	}
	var cg cloudGame
	if err := json.Unmarshal(gameJSONBytes, &cg); err != nil {
		s.logger.Warn("game.json解析失敗", "gameId", gameID, "error", err)
		return nil
	}
	return &CloudGameInfo{
		ID:             cg.ID,
		Title:          cg.Title,
		Publisher:      cg.Publisher,
		ImageHash:      cg.ImageHash,
		PlayStatus:     cg.PlayStatus,
		TotalPlayTime:  cg.TotalPlayTime,
		LastPlayed:     cg.LastPlayed,
		ClearedAt:      cg.ClearedAt,
		CurrentRouteID: cg.CurrentRouteID,
		CreatedAt:      cg.CreatedAt,
		UpdatedAt:      cg.UpdatedAt,
	}
}
