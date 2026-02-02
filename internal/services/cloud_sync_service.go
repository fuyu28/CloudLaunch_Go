// @fileoverview ゲーム基本情報とセッションのクラウド同期を提供する。
package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/credentials"
	"CloudLaunch_Go/internal/db"
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/storage"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

const cloudMetadataVersion = 2
const cloudSessionsFileName = "sessions.json"

// CloudSyncSummary は同期結果の要約を表す。
type CloudSyncSummary struct {
	UploadedGames      int `json:"uploadedGames"`
	DownloadedGames    int `json:"downloadedGames"`
	UploadedSessions   int `json:"uploadedSessions"`
	DownloadedSessions int `json:"downloadedSessions"`
	UploadedImages     int `json:"uploadedImages"`
	DownloadedImages   int `json:"downloadedImages"`
	SkippedGames       int `json:"skippedGames"`
}

// CloudSyncService はゲーム情報のクラウド同期を提供する。
type CloudSyncService struct {
	config     config.Config
	store      credentials.Store
	repository *db.Repository
	logger     *slog.Logger
	offlineMu  sync.RWMutex
	offline    bool
}

// NewCloudSyncService は CloudSyncService を生成する。
func NewCloudSyncService(cfg config.Config, store credentials.Store, repository *db.Repository, logger *slog.Logger) *CloudSyncService {
	return &CloudSyncService{
		config:     cfg,
		store:      store,
		repository: repository,
		logger:     logger,
	}
}

// SetOfflineMode は同期可否を切り替える。
func (service *CloudSyncService) SetOfflineMode(enabled bool) {
	service.offlineMu.Lock()
	defer service.offlineMu.Unlock()
	service.offline = enabled
}

func (service *CloudSyncService) isOffline() bool {
	service.offlineMu.RLock()
	defer service.offlineMu.RUnlock()
	return service.offline
}

// SyncAllGames は全ゲームの同期を行う。
func (service *CloudSyncService) SyncAllGames(ctx context.Context, credentialKey string) result.ApiResult[CloudSyncSummary] {
	return service.sync(ctx, credentialKey, "")
}

// SyncGame は指定ゲームのみ同期する。
func (service *CloudSyncService) SyncGame(ctx context.Context, credentialKey string, gameID string) result.ApiResult[CloudSyncSummary] {
	if _, detail, ok := requireNonEmpty(gameID, "gameID"); !ok {
		return result.ErrorResult[CloudSyncSummary]("ゲームIDが不正です", detail)
	}
	return service.sync(ctx, credentialKey, strings.TrimSpace(gameID))
}

func (service *CloudSyncService) sync(ctx context.Context, credentialKey string, gameID string) result.ApiResult[CloudSyncSummary] {
	if service.isOffline() {
		return result.ErrorResult[CloudSyncSummary]("オフラインモードのため同期できません", "offline mode")
	}
	trimmedKey, detail, ok := requireNonEmpty(credentialKey, "credentialKey")
	if !ok {
		return result.ErrorResult[CloudSyncSummary]("認証情報が不正です", detail)
	}

	client, cfg, message, detail, ok := service.newClient(ctx, trimmedKey)
	if !ok {
		return result.ErrorResult[CloudSyncSummary](message, detail)
	}

	metadata, err := storage.LoadMetadata(ctx, client, cfg.Bucket, service.config.CloudMetadataKey)
	if err != nil {
		if !isNotFoundError(err) {
			service.logger.Error("クラウドメタ情報取得に失敗", "error", err)
			return result.ErrorResult[CloudSyncSummary]("クラウドメタ情報取得に失敗しました", err.Error())
		}
		metadata = &storage.CloudMetadata{
			Version: cloudMetadataVersion,
			Games:   []storage.CloudGameMetadata{},
		}
	}

	localGames, err := service.loadLocalGames(ctx, gameID)
	if err != nil {
		service.logger.Error("ローカルゲーム取得に失敗", "error", err)
		return result.ErrorResult[CloudSyncSummary]("ローカルゲーム取得に失敗しました", err.Error())
	}

	cloudMap := make(map[string]storage.CloudGameMetadata, len(metadata.Games))
	for _, game := range metadata.Games {
		cloudMap[game.ID] = game
	}

	merged := make(map[string]storage.CloudGameMetadata, len(cloudMap))
	for id, cloudGame := range cloudMap {
		merged[id] = cloudGame
	}

	var summary CloudSyncSummary
	shouldSave := false

	unionIDs := map[string]struct{}{}
	for id := range localGames {
		unionIDs[id] = struct{}{}
	}
	for id := range cloudMap {
		unionIDs[id] = struct{}{}
	}

	for id := range unionIDs {
		if gameID != "" && id != gameID {
			continue
		}
		local, hasLocal := localGames[id]
		cloud, hasCloud := cloudMap[id]

		switch {
		case hasLocal && hasCloud:
			switch {
			case local.Game.UpdatedAt.After(cloud.UpdatedAt):
				cloudGame, uploadedImages, err := service.buildCloudGame(ctx, client, cfg.Bucket, local, &cloud)
				if err != nil {
					return result.ErrorResult[CloudSyncSummary]("クラウド更新に失敗しました", err.Error())
				}
				merged[id] = cloudGame
				shouldSave = true
				summary.UploadedGames++
				summary.UploadedSessions += len(local.Sessions)
				summary.UploadedImages += uploadedImages
			case cloud.UpdatedAt.After(local.Game.UpdatedAt):
				downloadedImages, downloadedSessions, err := service.applyCloudGame(ctx, client, cfg.Bucket, cloud, &local.Game)
				if err != nil {
					return result.ErrorResult[CloudSyncSummary]("ローカル更新に失敗しました", err.Error())
				}
				merged[id] = cloud
				summary.DownloadedGames++
				summary.DownloadedSessions += downloadedSessions
				summary.DownloadedImages += downloadedImages
			default:
				merged[id] = cloud
				summary.SkippedGames++
			}
		case hasLocal && !hasCloud:
			cloudGame, uploadedImages, err := service.buildCloudGame(ctx, client, cfg.Bucket, local, nil)
			if err != nil {
				return result.ErrorResult[CloudSyncSummary]("クラウド更新に失敗しました", err.Error())
			}
			merged[id] = cloudGame
			shouldSave = true
			summary.UploadedGames++
			summary.UploadedSessions += len(local.Sessions)
			summary.UploadedImages += uploadedImages
		case !hasLocal && hasCloud:
			downloadedImages, downloadedSessions, err := service.applyCloudGame(ctx, client, cfg.Bucket, cloud, nil)
			if err != nil {
				return result.ErrorResult[CloudSyncSummary]("ローカル更新に失敗しました", err.Error())
			}
			merged[id] = cloud
			summary.DownloadedGames++
			summary.DownloadedSessions += downloadedSessions
			summary.DownloadedImages += downloadedImages
		}
	}

	if shouldSave {
		updatedGames := mapToSortedGames(merged)
		updated := storage.CloudMetadata{
			Version:   cloudMetadataVersion,
			UpdatedAt: time.Now(),
			Games:     updatedGames,
		}
		if err := storage.SaveMetadata(ctx, client, cfg.Bucket, service.config.CloudMetadataKey, updated); err != nil {
			service.logger.Error("クラウドメタ情報更新に失敗", "error", err)
			return result.ErrorResult[CloudSyncSummary]("クラウド更新に失敗しました", err.Error())
		}
	}

	return result.OkResult(summary)
}

type localGameBundle struct {
	Game     models.Game
	Sessions []models.PlaySession
}

func cloudSessionsKey(gameID string) string {
	return fmt.Sprintf("games/%s/%s", gameID, cloudSessionsFileName)
}

func (service *CloudSyncService) loadLocalGames(ctx context.Context, gameID string) (map[string]localGameBundle, error) {
	result := make(map[string]localGameBundle)
	if gameID != "" {
		game, err := service.repository.GetGameByID(ctx, gameID)
		if err != nil {
			return nil, err
		}
		if game == nil {
			return result, nil
		}
		sessions, err := service.repository.ListPlaySessionsByGame(ctx, gameID)
		if err != nil {
			return nil, err
		}
		result[game.ID] = localGameBundle{Game: *game, Sessions: sessions}
		return result, nil
	}

	games, err := service.repository.ListGames(ctx, "", models.PlayStatus(""), "title", "asc")
	if err != nil {
		return nil, err
	}
	for _, game := range games {
		sessions, err := service.repository.ListPlaySessionsByGame(ctx, game.ID)
		if err != nil {
			return nil, err
		}
		result[game.ID] = localGameBundle{Game: game, Sessions: sessions}
	}
	return result, nil
}

func (service *CloudSyncService) buildCloudGame(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	local localGameBundle,
	existing *storage.CloudGameMetadata,
) (storage.CloudGameMetadata, int, error) {
	game := local.Game
	cloudGame := storage.CloudGameMetadata{
		ID:             game.ID,
		Title:          game.Title,
		Publisher:      game.Publisher,
		PlayStatus:     string(game.PlayStatus),
		TotalPlayTime:  game.TotalPlayTime,
		LastPlayed:     game.LastPlayed,
		ClearedAt:      game.ClearedAt,
		CurrentChapter: game.CurrentChapter,
		CreatedAt:      game.CreatedAt,
		UpdatedAt:      game.UpdatedAt,
	}

	sessions := make([]storage.CloudSessionRecord, 0, len(local.Sessions))
	for _, session := range local.Sessions {
		sessions = append(sessions, storage.CloudSessionRecord{
			ID:          session.ID,
			PlayedAt:    session.PlayedAt,
			Duration:    session.Duration,
			SessionName: session.SessionName,
			UpdatedAt:   session.UpdatedAt,
		})
	}
	if err := storage.SaveSessions(ctx, client, bucket, cloudSessionsKey(game.ID), sessions); err != nil {
		return cloudGame, 0, err
	}

	if game.ImagePath != nil && strings.TrimSpace(*game.ImagePath) != "" {
		imageKey, uploaded, err := service.uploadImageIfNeeded(ctx, client, bucket, game.ID, *game.ImagePath, existing)
		if err != nil {
			service.logger.Warn("サムネイルのアップロードに失敗", "error", err, "gameId", game.ID)
		} else if imageKey != "" {
			cloudGame.ImageKey = &imageKey
			if uploaded {
				return cloudGame, 1, nil
			}
		}
	}

	if existing != nil && existing.ImageKey != nil {
		cloudGame.ImageKey = existing.ImageKey
	}

	return cloudGame, 0, nil
}

func (service *CloudSyncService) applyCloudGame(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	cloud storage.CloudGameMetadata,
	local *models.Game,
) (int, int, error) {
	exePath := UnconfiguredExePath
	saveFolder := (*string)(nil)
	if local != nil {
		if strings.TrimSpace(local.ExePath) != "" {
			exePath = local.ExePath
		}
		saveFolder = local.SaveFolderPath
	}

	imagePath := (*string)(nil)
	downloadedImages := 0
	if cloud.ImageKey != nil && strings.TrimSpace(*cloud.ImageKey) != "" {
		path, downloaded, err := service.downloadImageIfNeeded(ctx, client, bucket, cloud.ID, *cloud.ImageKey)
		if err != nil {
			return 0, 0, err
		}
		imagePath = &path
		if downloaded {
			downloadedImages++
		}
	}

	game := models.Game{
		ID:             cloud.ID,
		Title:          cloud.Title,
		Publisher:      cloud.Publisher,
		ImagePath:      imagePath,
		ExePath:        exePath,
		SaveFolderPath: saveFolder,
		CreatedAt:      cloud.CreatedAt,
		UpdatedAt:      cloud.UpdatedAt,
		PlayStatus:     models.PlayStatus(cloud.PlayStatus),
		TotalPlayTime:  cloud.TotalPlayTime,
		LastPlayed:     cloud.LastPlayed,
		ClearedAt:      cloud.ClearedAt,
		CurrentChapter: cloud.CurrentChapter,
	}

	if err := service.repository.UpsertGameSync(ctx, game); err != nil {
		return 0, 0, err
	}

	if err := service.repository.DeletePlaySessionsByGame(ctx, cloud.ID); err != nil {
		return 0, 0, err
	}

	cloudSessions, err := service.loadCloudSessions(ctx, client, bucket, cloud.ID)
	if err != nil {
		return 0, 0, err
	}
	for _, session := range cloudSessions {
		playSession := models.PlaySession{
			ID:          session.ID,
			GameID:      cloud.ID,
			PlayedAt:    session.PlayedAt,
			Duration:    session.Duration,
			SessionName: session.SessionName,
			UpdatedAt:   session.UpdatedAt,
		}
		if err := service.repository.UpsertPlaySessionSync(ctx, playSession); err != nil {
			return 0, 0, err
		}
	}

	if total, err := service.repository.SumPlaySessionDurationsByGame(ctx, cloud.ID); err == nil {
		if updateErr := service.repository.UpdateGameTotalPlayTime(ctx, cloud.ID, total); updateErr != nil {
			return 0, 0, updateErr
		}
	}

	return downloadedImages, len(cloudSessions), nil
}

func (service *CloudSyncService) loadCloudSessions(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	gameID string,
) ([]storage.CloudSessionRecord, error) {
	key := cloudSessionsKey(gameID)
	sessions, err := storage.LoadSessions(ctx, client, bucket, key)
	if err != nil {
		if isNotFoundError(err) {
			return []storage.CloudSessionRecord{}, nil
		}
		return nil, err
	}
	return sessions, nil
}

func (service *CloudSyncService) uploadImageIfNeeded(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	gameID string,
	imagePath string,
	existing *storage.CloudGameMetadata,
) (string, bool, error) {
	payload, ext, contentType, err := loadImagePayload(imagePath)
	if err != nil {
		return "", false, err
	}
	hash := sha256.Sum256(payload)
	hashHex := hex.EncodeToString(hash[:])
	normalizedExt := normalizeImageExt(ext, contentType)
	key := fmt.Sprintf("games/%s/thumbnail/%s%s", gameID, hashHex, normalizedExt)

	if existing != nil && existing.ImageKey != nil && *existing.ImageKey == key {
		return key, false, nil
	}

	if err := storage.UploadBytes(ctx, client, bucket, key, payload, contentType); err != nil {
		return "", false, err
	}
	return key, true, nil
}

func (service *CloudSyncService) downloadImageIfNeeded(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	gameID string,
	key string,
) (string, bool, error) {
	baseName := filepath.Base(key)
	if baseName == "" {
		return "", false, errors.New("image key is empty")
	}
	targetDir := filepath.Join(service.config.AppDataDir, "images", gameID)
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return "", false, err
	}
	targetPath := filepath.Join(targetDir, baseName)
	if _, err := os.Stat(targetPath); err == nil {
		return targetPath, false, nil
	}

	payload, err := storage.DownloadObject(ctx, client, bucket, key)
	if err != nil {
		return "", false, err
	}
	if err := os.WriteFile(targetPath, payload, 0o600); err != nil {
		return "", false, err
	}
	return targetPath, true, nil
}

func loadImagePayload(path string) ([]byte, string, string, error) {
	if isURL(path) {
		response, err := http.Get(path)
		if err != nil {
			return nil, "", "", err
		}
		defer func() {
			if closeErr := response.Body.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
		}()
		payload, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, "", "", err
		}
		ext := filepath.Ext(response.Request.URL.Path)
		contentType := strings.TrimSpace(response.Header.Get("Content-Type"))
		if contentType == "" {
			contentType = http.DetectContentType(payload)
		}
		return payload, ext, contentType, nil
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, "", "", err
	}
	ext := filepath.Ext(path)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = http.DetectContentType(payload)
	}
	return payload, ext, contentType, nil
}

func normalizeImageExt(ext string, contentType string) string {
	trimmed := strings.ToLower(strings.TrimSpace(ext))
	if trimmed != "" {
		if strings.HasPrefix(trimmed, ".") {
			return trimmed
		}
		return "." + trimmed
	}
	if strings.Contains(contentType, "png") {
		return ".png"
	}
	if strings.Contains(contentType, "gif") {
		return ".gif"
	}
	if strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg") {
		return ".jpg"
	}
	return ".png"
}

func isURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		if code == "NoSuchKey" || code == "NotFound" {
			return true
		}
	}
	var noSuchKey *types.NoSuchKey
	return errors.As(err, &noSuchKey)
}

func mapToSortedGames(source map[string]storage.CloudGameMetadata) []storage.CloudGameMetadata {
	games := make([]storage.CloudGameMetadata, 0, len(source))
	for _, game := range source {
		games = append(games, game)
	}
	sort.Slice(games, func(i, j int) bool {
		if games[i].Title == games[j].Title {
			return games[i].ID < games[j].ID
		}
		return games[i].Title < games[j].Title
	})
	return games
}

func (service *CloudSyncService) newClient(
	ctx context.Context,
	credentialKey string,
) (*s3.Client, storage.S3Config, string, string, bool) {
	credential, err := service.store.Load(ctx, strings.TrimSpace(credentialKey))
	if err != nil {
		service.logger.Error("認証情報取得に失敗", "error", err)
		return nil, storage.S3Config{}, "認証情報取得に失敗しました", err.Error(), false
	}
	if credential == nil {
		return nil, storage.S3Config{}, "認証情報が見つかりません", "credentialが空です", false
	}

	cfg := resolveS3Config(service.config, credential)
	client, err := storage.NewClient(ctx, cfg, *credential)
	if err != nil {
		service.logger.Error("S3クライアント作成に失敗", "error", err)
		return nil, cfg, "S3クライアント作成に失敗しました", err.Error(), false
	}
	return client, cfg, "", "", true
}
