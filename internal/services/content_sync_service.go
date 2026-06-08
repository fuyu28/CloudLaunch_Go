package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/infrastructure/credentials"
	"CloudLaunch_Go/internal/infrastructure/storage"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ProgressFunc はセーブファイルの転送進捗を報告するコールバック。
type ProgressFunc func(current, total int)

// ContentSyncService はコンテンツアドレッシングによるゲームデータ同期を提供する。
type ContentSyncService struct {
	config     config.Config
	store      credentials.Store
	repository ContentSyncRepository
	logger     *slog.Logger
}

// NewContentSyncService は ContentSyncService を生成する。
func NewContentSyncService(cfg config.Config, store credentials.Store, repo ContentSyncRepository, logger *slog.Logger) *ContentSyncService {
	return &ContentSyncService{
		config:     cfg,
		store:      store,
		repository: repo,
		logger:     logger,
	}
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
	_ = s.repository.UpsertSetting(ctx, "device_name", hostname)
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
		if h, _, herr := hashFile(*game.ImagePath); herr == nil {
			imageHash = h
		}
	}
	saveSnap, _, err := buildSaveSnapshot(saveFolderPath)
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
	client, cfg, err := s.newClient(ctx)
	if err != nil {
		return domain.SyncStatusDetail{}, err
	}

	remoteHead, err := storage.ReadHEAD(ctx, client, cfg.Bucket, gameID)
	if err != nil {
		return domain.SyncStatusDetail{}, err
	}
	if remoteHead == "" {
		return domain.SyncStatusDetail{Status: domain.SyncStatusNeverSynced}, nil
	}

	remoteMetaBytes, err := storage.GetBlob(ctx, client, cfg.Bucket, gameID, remoteHead)
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

// Push はローカルデータをリモートにアップロードする。
func (s *ContentSyncService) Push(ctx context.Context, gameID string, onProgress ProgressFunc) error {
	client, cfg, err := s.newClient(ctx)
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

	sessions, err := s.repository.ListPlaySessionsByGame(ctx, gameID)
	if err != nil {
		return err
	}
	deviceName, err := s.getOrInitDeviceName(ctx)
	if err != nil {
		return err
	}

	saveSnap, saveBlobs, err := buildSaveSnapshot(*game.SaveFolderPath)
	if err != nil {
		return err
	}
	saveSnapJSON, err := json.Marshal(saveSnap)
	if err != nil {
		return err
	}
	savesHash := hashBytes(saveSnapJSON)

	imageHash := ""
	var imageData []byte
	if game.ImagePath != nil && *game.ImagePath != "" {
		h, data, herr := hashFile(*game.ImagePath)
		if herr == nil {
			imageHash = h
			imageData = data
		}
	}

	meta, err := buildMetaSnapshot(*game, sessions, imageHash, savesHash, deviceName)
	if err != nil {
		return err
	}
	metaHash := hashBytes(meta.SnapshotBytes)

	// セーブファイルをアップロード（進捗付き）
	total := len(saveBlobs)
	current := 0
	if onProgress != nil {
		onProgress(0, total)
	}
	for hash, data := range saveBlobs {
		if err := storage.PutBlob(ctx, client, cfg.Bucket, gameID, hash, data); err != nil {
			return err
		}
		current++
		if onProgress != nil {
			onProgress(current, total)
		}
	}

	// セーブスナップショット・画像・game.json・sessions.json をアップロード
	if err := storage.PutBlob(ctx, client, cfg.Bucket, gameID, savesHash, saveSnapJSON); err != nil {
		return err
	}
	if imageHash != "" && imageData != nil {
		if err := storage.PutBlob(ctx, client, cfg.Bucket, gameID, imageHash, imageData); err != nil {
			return err
		}
	}
	if err := storage.PutBlob(ctx, client, cfg.Bucket, gameID, meta.Snapshot.GameJSON, meta.GameJSON); err != nil {
		return err
	}
	if err := storage.PutBlob(ctx, client, cfg.Bucket, gameID, meta.Snapshot.SessionsJSON, meta.SessionsJSON); err != nil {
		return err
	}
	if err := storage.PutBlob(ctx, client, cfg.Bucket, gameID, metaHash, meta.SnapshotBytes); err != nil {
		return err
	}

	if err := storage.WriteHEAD(ctx, client, cfg.Bucket, gameID, metaHash); err != nil {
		return err
	}
	return s.repository.SetLocalSyncHead(ctx, gameID, contentFingerprint(meta.Snapshot))
}

// Pull はリモートデータをローカルに適用する。
func (s *ContentSyncService) Pull(ctx context.Context, gameID string, onProgress ProgressFunc) error {
	client, cfg, err := s.newClient(ctx)
	if err != nil {
		return err
	}

	remoteHead, err := storage.ReadHEAD(ctx, client, cfg.Bucket, gameID)
	if err != nil {
		return err
	}
	if remoteHead == "" {
		return fmt.Errorf("リモートにデータがありません")
	}

	metaBytes, err := storage.GetBlob(ctx, client, cfg.Bucket, gameID, remoteHead)
	if err != nil {
		return err
	}
	var meta domain.MetaSnapshot
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return err
	}

	saveSnapBytes, err := storage.GetBlob(ctx, client, cfg.Bucket, gameID, meta.Saves)
	if err != nil {
		return err
	}
	var saveSnap domain.SaveSnapshot
	if err := json.Unmarshal(saveSnapBytes, &saveSnap); err != nil {
		return err
	}

	gameJSONBytes, err := storage.GetBlob(ctx, client, cfg.Bucket, gameID, meta.GameJSON)
	if err != nil {
		return err
	}
	var cloudG cloudGame
	if err := json.Unmarshal(gameJSONBytes, &cloudG); err != nil {
		return err
	}

	sessionsJSONBytes, err := storage.GetBlob(ctx, client, cfg.Bucket, gameID, meta.SessionsJSON)
	if err != nil {
		return err
	}
	var cloudSessions []cloudSession
	if err := json.Unmarshal(sessionsJSONBytes, &cloudSessions); err != nil {
		return err
	}

	// ローカルゲームのマシン固有フィールドを保持
	localGame, err := s.repository.GetGameByID(ctx, gameID)
	if err != nil {
		return err
	}
	exePath := UnconfiguredExePath
	saveFolderPath := (*string)(nil)
	imagePath := (*string)(nil)
	if localGame != nil {
		exePath = localGame.ExePath
		saveFolderPath = localGame.SaveFolderPath
		imagePath = localGame.ImagePath
	}

	// 画像をダウンロード（ローカルと異なる場合のみ）
	if cloudG.ImageHash != "" {
		localImageHash := ""
		if imagePath != nil && *imagePath != "" {
			if h, _, herr := hashFile(*imagePath); herr == nil {
				localImageHash = h
			}
		}
		if localImageHash != cloudG.ImageHash {
			imageData, berr := storage.GetBlob(ctx, client, cfg.Bucket, gameID, cloudG.ImageHash)
			if berr != nil {
				return berr
			}
			contentType := http.DetectContentType(imageData)
			ext := normalizeImageExt("", contentType)
			imgPath := filepath.Join(s.config.AppDataDir, "thumbnails",
				fmt.Sprintf("%s_%s%s", cloudG.ImageHash, gameID, ext))
			if err := os.MkdirAll(filepath.Dir(imgPath), 0o700); err != nil {
				return err
			}
			if err := os.WriteFile(imgPath, imageData, 0o600); err != nil {
				return err
			}
			imagePath = &imgPath
		}
	}

	// セーブファイルをダウンロード（進捗付き）
	if saveFolderPath != nil && *saveFolderPath != "" {
		saveDir := *saveFolderPath
		total := len(saveSnap.Files)
		current := 0
		if onProgress != nil {
			onProgress(0, total)
		}
		if err := os.MkdirAll(saveDir, 0o700); err != nil {
			return err
		}
		for relPath, hash := range saveSnap.Files {
			data, berr := storage.GetBlob(ctx, client, cfg.Bucket, gameID, hash)
			if berr != nil {
				return berr
			}
			targetPath := filepath.Join(saveDir, filepath.FromSlash(relPath))
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
				return err
			}
			if err := os.WriteFile(targetPath, data, 0o600); err != nil {
				return err
			}
			current++
			if onProgress != nil {
				onProgress(current, total)
			}
		}
	} else {
		s.logger.Warn("セーブフォルダ未設定のためセーブデータをスキップします", "gameId", gameID)
	}

	// ゲーム情報を更新
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
	if err := s.repository.UpsertGameSync(ctx, updatedGame); err != nil {
		return err
	}

	// セッションを差し替え
	if err := s.repository.DeletePlaySessionsByGame(ctx, gameID); err != nil {
		return err
	}
	for _, cs := range cloudSessions {
		session := domain.PlaySession{
			ID:          cs.ID,
			GameID:      gameID,
			PlayedAt:    cs.PlayedAt,
			Duration:    cs.Duration,
			SessionName: cs.SessionName,
			RouteID:     cs.RouteID,
			UpdatedAt:   cs.UpdatedAt,
		}
		if err := s.repository.UpsertPlaySessionSync(ctx, session); err != nil {
			return err
		}
	}

	return s.repository.SetLocalSyncHead(ctx, gameID, contentFingerprint(meta))
}

// ResolveConflict はコンフリクトを手動解決する。
func (s *ContentSyncService) ResolveConflict(ctx context.Context, gameID string, useLocal bool) error {
	if useLocal {
		return s.Push(ctx, gameID, nil)
	}
	return s.Pull(ctx, gameID, nil)
}

// DeleteFromCloud はゲームのリモートデータを削除する。
func (s *ContentSyncService) DeleteFromCloud(ctx context.Context, gameID string) error {
	client, cfg, err := s.newClient(ctx)
	if err != nil {
		return err
	}
	prefix := fmt.Sprintf("games/%s/", gameID)
	return storage.DeleteObjectsByPrefix(ctx, client, cfg.Bucket, prefix)
}
