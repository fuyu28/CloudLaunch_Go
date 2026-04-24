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
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/credentials"
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/storage"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

const cloudMetadataVersion = 2
const cloudSessionsFileName = "sessions.json"
const maxRemoteImageBytes = 10 << 20

var remoteImageHTTPClient = &http.Client{Timeout: 15 * time.Second}

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
	repository CloudSyncRepository
	logger     *slog.Logger
	offlineMu  sync.RWMutex
	offline    bool
}

// NewCloudSyncService は CloudSyncService を生成する。
func NewCloudSyncService(cfg config.Config, store credentials.Store, repository CloudSyncRepository, logger *slog.Logger) *CloudSyncService {
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
		service.logger.Warn("ゲームIDが不正です", "operation", "SyncGame", "detail", detail, "gameId", gameID)
		return result.ErrorResult[CloudSyncSummary]("ゲームIDが不正です", detail)
	}
	return service.sync(ctx, credentialKey, strings.TrimSpace(gameID))
}

// DeleteGameFromCloud は指定ゲームのクラウドデータを削除する。
func (service *CloudSyncService) DeleteGameFromCloud(ctx context.Context, credentialKey string, gameID string) result.ApiResult[bool] {
	if service.isOffline() {
		service.logger.Warn("オフラインモードのため削除できません", "operation", "DeleteGameFromCloud")
		return result.ErrorResult[bool]("オフラインモードのため削除できません", "offline mode")
	}
	trimmedKey, detail, ok := requireNonEmpty(credentialKey, "credentialKey")
	if !ok {
		service.logger.Warn("認証情報が不正です", "operation", "DeleteGameFromCloud", "detail", detail)
		return result.ErrorResult[bool]("認証情報が不正です", detail)
	}
	trimmedID, detail, ok := requireNonEmpty(gameID, "gameID")
	if !ok {
		service.logger.Warn("ゲームIDが不正です", "operation", "DeleteGameFromCloud", "detail", detail, "gameId", gameID)
		return result.ErrorResult[bool]("ゲームIDが不正です", detail)
	}

	client, cfg, message, detail, ok := service.newClient(ctx, trimmedKey)
	if !ok {
		service.logger.Warn("S3クライアント初期化に失敗", "operation", "DeleteGameFromCloud", "message", message, "detail", detail)
		return result.ErrorResult[bool](message, detail)
	}

	prefix := fmt.Sprintf("games/%s/", trimmedID)
	if err := storage.DeleteObjectsByPrefix(ctx, client, cfg.Bucket, prefix); err != nil {
		service.logger.Error("クラウドデータ削除に失敗", "error", err, "gameId", trimmedID)
		return result.ErrorResult[bool]("クラウドデータ削除に失敗しました", err.Error())
	}

	metadata, err := storage.LoadMetadata(ctx, client, cfg.Bucket, service.config.CloudMetadataKey)
	if err != nil {
		if isNotFoundError(err) {
			return result.OkResult(true)
		}
		service.logger.Error("クラウドメタ情報取得に失敗", "error", err)
		return result.ErrorResult[bool]("クラウドメタ情報取得に失敗しました", err.Error())
	}

	updatedGames := make([]storage.CloudGameMetadata, 0, len(metadata.Games))
	for _, game := range metadata.Games {
		if game.ID != trimmedID {
			updatedGames = append(updatedGames, game)
		}
	}

	if len(updatedGames) == len(metadata.Games) {
		return result.OkResult(true)
	}

	metadata.Games = updatedGames
	metadata.UpdatedAt = time.Now()
	if err := storage.SaveMetadata(ctx, client, cfg.Bucket, service.config.CloudMetadataKey, *metadata); err != nil {
		service.logger.Error("クラウドメタ情報更新に失敗", "error", err)
		return result.ErrorResult[bool]("クラウドメタ情報更新に失敗しました", err.Error())
	}

	return result.OkResult(true)
}

func (service *CloudSyncService) sync(ctx context.Context, credentialKey string, gameID string) result.ApiResult[CloudSyncSummary] {
	if service.isOffline() {
		service.logger.Warn("オフラインモードのため同期できません", "operation", "sync", "gameId", gameID)
		return result.ErrorResult[CloudSyncSummary]("オフラインモードのため同期できません", "offline mode")
	}
	trimmedKey, detail, ok := requireNonEmpty(credentialKey, "credentialKey")
	if !ok {
		service.logger.Warn("認証情報が不正です", "operation", "sync", "detail", detail)
		return result.ErrorResult[CloudSyncSummary]("認証情報が不正です", detail)
	}

	client, cfg, message, detail, ok := service.newClient(ctx, trimmedKey)
	if !ok {
		service.logger.Warn("S3クライアント初期化に失敗", "operation", "sync", "message", message, "detail", detail)
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

	cloudMap := cloudMetadataToMap(metadata)
	merged := copyCloudGameMap(cloudMap)

	var summary CloudSyncSummary
	shouldSave := false
	targetIDs := collectUnionGameIDs(localGames, cloudMap, gameID)

	for _, id := range targetIDs {
		local, hasLocal := localGames[id]
		cloud, hasCloud := cloudMap[id]
		iteration, err := service.syncSingleGame(ctx, client, cfg.Bucket, id, local, hasLocal, cloud, hasCloud)
		if err != nil {
			message := "クラウド更新に失敗しました"
			if determineGameSyncAction(local, hasLocal, cloud, hasCloud) == gameSyncActionDownload {
				message = "ローカル更新に失敗しました"
			}
			return result.ErrorResult[CloudSyncSummary](message, err.Error())
		}
		if iteration.cloudGame != nil {
			merged[id] = *iteration.cloudGame
		}
		shouldSave = shouldSave || iteration.shouldSaveMetadata
		summary.add(iteration.summary)
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

type gameSyncAction int

const (
	gameSyncActionSkip gameSyncAction = iota
	gameSyncActionUpload
	gameSyncActionDownload
)

type gameSyncIterationResult struct {
	cloudGame          *storage.CloudGameMetadata
	summary            CloudSyncSummary
	shouldSaveMetadata bool
}

func cloudSessionsKey(gameID string) string {
	return fmt.Sprintf("games/%s/%s", gameID, cloudSessionsFileName)
}

func determineGameSyncAction(local localGameBundle, hasLocal bool, cloud storage.CloudGameMetadata, hasCloud bool) gameSyncAction {
	switch {
	case hasLocal && hasCloud:
		switch {
		case local.Game.UpdatedAt.After(cloud.UpdatedAt):
			return gameSyncActionUpload
		case cloud.UpdatedAt.After(local.Game.UpdatedAt):
			return gameSyncActionDownload
		default:
			return gameSyncActionSkip
		}
	case hasLocal && !hasCloud:
		return gameSyncActionUpload
	case !hasLocal && hasCloud:
		return gameSyncActionDownload
	default:
		return gameSyncActionSkip
	}
}

func cloudMetadataToMap(metadata *storage.CloudMetadata) map[string]storage.CloudGameMetadata {
	if metadata == nil {
		return map[string]storage.CloudGameMetadata{}
	}
	cloudMap := make(map[string]storage.CloudGameMetadata, len(metadata.Games))
	for _, game := range metadata.Games {
		cloudMap[game.ID] = game
	}
	return cloudMap
}

func copyCloudGameMap(source map[string]storage.CloudGameMetadata) map[string]storage.CloudGameMetadata {
	result := make(map[string]storage.CloudGameMetadata, len(source))
	for id, game := range source {
		result[id] = game
	}
	return result
}

func collectUnionGameIDs(localGames map[string]localGameBundle, cloudGames map[string]storage.CloudGameMetadata, gameID string) []string {
	unionIDs := map[string]struct{}{}
	for id := range localGames {
		unionIDs[id] = struct{}{}
	}
	for id := range cloudGames {
		unionIDs[id] = struct{}{}
	}

	collected := make([]string, 0, len(unionIDs))
	for id := range unionIDs {
		if gameID != "" && id != gameID {
			continue
		}
		collected = append(collected, id)
	}
	sort.Strings(collected)
	return collected
}

func (service *CloudSyncService) syncSingleGame(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	gameID string,
	local localGameBundle,
	hasLocal bool,
	cloud storage.CloudGameMetadata,
	hasCloud bool,
) (gameSyncIterationResult, error) {
	switch determineGameSyncAction(local, hasLocal, cloud, hasCloud) {
	case gameSyncActionUpload:
		existing := (*storage.CloudGameMetadata)(nil)
		operation := "sync.createCloudGame"
		if hasCloud {
			existing = &cloud
			operation = "sync.buildCloudGame"
		}
		cloudGame, uploadedImages, err := service.buildCloudGame(ctx, client, bucket, local, existing)
		if err != nil {
			service.logger.Error("クラウド更新に失敗", "operation", operation, "gameId", gameID, "error", err)
			return gameSyncIterationResult{}, err
		}
		return gameSyncIterationResult{
			cloudGame: &cloudGame,
			summary: CloudSyncSummary{
				UploadedGames:    1,
				UploadedSessions: len(local.Sessions),
				UploadedImages:   uploadedImages,
			},
			shouldSaveMetadata: true,
		}, nil
	case gameSyncActionDownload:
		var currentLocal *models.Game
		operation := "sync.createLocalGame"
		if hasLocal {
			currentLocal = &local.Game
			operation = "sync.applyCloudGame"
		}
		downloadedImages, downloadedSessions, err := service.applyCloudGame(ctx, client, bucket, cloud, currentLocal)
		if err != nil {
			service.logger.Error("ローカル更新に失敗", "operation", operation, "gameId", gameID, "error", err)
			return gameSyncIterationResult{}, err
		}
		return gameSyncIterationResult{
			cloudGame: &cloud,
			summary: CloudSyncSummary{
				DownloadedGames:    1,
				DownloadedSessions: downloadedSessions,
				DownloadedImages:   downloadedImages,
			},
		}, nil
	default:
		result := gameSyncIterationResult{
			summary: CloudSyncSummary{
				SkippedGames: 1,
			},
		}
		if hasCloud {
			result.cloudGame = &cloud
		}
		return result, nil
	}
}

func (summary *CloudSyncSummary) add(other CloudSyncSummary) {
	summary.UploadedGames += other.UploadedGames
	summary.DownloadedGames += other.DownloadedGames
	summary.UploadedSessions += other.UploadedSessions
	summary.DownloadedSessions += other.DownloadedSessions
	summary.UploadedImages += other.UploadedImages
	summary.DownloadedImages += other.DownloadedImages
	summary.SkippedGames += other.SkippedGames
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
	localSaveHash := (*string)(nil)
	localSaveHashUpdatedAt := (*time.Time)(nil)
	if local != nil {
		if strings.TrimSpace(local.ExePath) != "" {
			exePath = local.ExePath
		}
		saveFolder = local.SaveFolderPath
		localSaveHash = local.LocalSaveHash
		localSaveHashUpdatedAt = local.LocalSaveHashUpdatedAt
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
		ID:                     cloud.ID,
		Title:                  cloud.Title,
		Publisher:              cloud.Publisher,
		ImagePath:              imagePath,
		ExePath:                exePath,
		SaveFolderPath:         saveFolder,
		CreatedAt:              cloud.CreatedAt,
		UpdatedAt:              cloud.UpdatedAt,
		LocalSaveHash:          localSaveHash,
		LocalSaveHashUpdatedAt: localSaveHashUpdatedAt,
		PlayStatus:             models.PlayStatus(cloud.PlayStatus),
		TotalPlayTime:          cloud.TotalPlayTime,
		LastPlayed:             cloud.LastPlayed,
		ClearedAt:              cloud.ClearedAt,
		CurrentChapter:         cloud.CurrentChapter,
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
	ext := filepath.Ext(baseName)
	hash := strings.TrimSuffix(baseName, ext)
	if hash == "" {
		return "", false, errors.New("image hash is empty")
	}
	targetDir := filepath.Join(service.config.AppDataDir, "thumbnails")
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return "", false, err
	}
	targetPath := filepath.Join(targetDir, fmt.Sprintf("%s_%s%s", hash, gameID, ext))
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
		parsed, err := url.Parse(path)
		if err != nil {
			return nil, "", "", err
		}
		if err := validateRemoteImageURL(parsed); err != nil {
			return nil, "", "", err
		}

		request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, parsed.String(), nil)
		if err != nil {
			return nil, "", "", err
		}
		response, err := remoteImageHTTPClient.Do(request)
		if err != nil {
			return nil, "", "", err
		}
		defer func() {
			if closeErr := response.Body.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
		}()
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			return nil, "", "", fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}
		if response.ContentLength > maxRemoteImageBytes {
			return nil, "", "", fmt.Errorf("image is too large: %d", response.ContentLength)
		}

		payload, err := io.ReadAll(io.LimitReader(response.Body, maxRemoteImageBytes+1))
		if err != nil {
			return nil, "", "", err
		}
		if len(payload) > maxRemoteImageBytes {
			return nil, "", "", fmt.Errorf("image is too large: %d", len(payload))
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

func validateRemoteImageURL(parsed *url.URL) error {
	if parsed == nil {
		return errors.New("url is nil")
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return errors.New("url hostname is empty")
	}
	if strings.EqualFold(host, "localhost") {
		return errors.New("localhost is not allowed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		if isPrivateOrLocalAddr(addr) {
			return fmt.Errorf("private or local address is not allowed: %s", addr.String())
		}
	}
	return nil
}

func isPrivateOrLocalAddr(addr netip.Addr) bool {
	return addr.IsLoopback() ||
		addr.IsPrivate() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsMulticast() ||
		addr.IsUnspecified()
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
		service.logger.Warn("認証情報が見つかりません", "credentialKey", credentialKey)
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
