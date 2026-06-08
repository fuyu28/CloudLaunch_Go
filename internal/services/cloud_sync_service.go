// @fileoverview ゲーム基本情報とセッションのクラウド同期を提供する。
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/infrastructure/credentials"
	"CloudLaunch_Go/internal/infrastructure/storage"
	"CloudLaunch_Go/internal/domain"

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
	config       config.Config
	store        credentials.Store
	repository   CloudSyncRepository
	cloudStorage cloudSyncStorage
	imageFiles   cloudImageFileStore
	imageLoader  cloudImageLoader
	newClient    cloudSyncClientFactory
	logger       *slog.Logger
	offlineMu    sync.RWMutex
	offline      bool
}

// NewCloudSyncService は CloudSyncService を生成する。
func NewCloudSyncService(cfg config.Config, store credentials.Store, repository CloudSyncRepository, logger *slog.Logger) *CloudSyncService {
	service := &CloudSyncService{
		config:       cfg,
		store:        store,
		repository:   repository,
		cloudStorage: storageCloudSyncStorage{},
		imageFiles:   osCloudImageFileStore{},
		imageLoader:  defaultCloudImageLoader{},
		logger:       logger,
	}
	service.newClient = service.newStorageClient
	return service
}

type cloudSyncClientFactory func(ctx context.Context, credentialKey string) (*s3.Client, storage.S3Config, string, string, bool)

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
func (service *CloudSyncService) SyncAllGames(ctx context.Context, credentialKey string) (CloudSyncSummary, error) {
	return service.sync(ctx, credentialKey, "")
}

// SyncGame は指定ゲームのみ同期する。
func (service *CloudSyncService) SyncGame(ctx context.Context, credentialKey string, gameID string) (CloudSyncSummary, error) {
	if _, detail, ok := requireNonEmpty(gameID, "gameID"); !ok {
		service.logger.Warn("ゲームIDが不正です", "operation", "SyncGame", "detail", detail, "gameId", gameID)
		return CloudSyncSummary{}, newServiceError("ゲームIDが不正です", detail)
	}
	return service.sync(ctx, credentialKey, strings.TrimSpace(gameID))
}

// DeleteGameFromCloud は指定ゲームのクラウドデータを削除する。
func (service *CloudSyncService) DeleteGameFromCloud(ctx context.Context, credentialKey string, gameID string) error {
	if service.isOffline() {
		service.logger.Warn("オフラインモードのため削除できません", "operation", "DeleteGameFromCloud")
		return newServiceError("オフラインモードのため削除できません", "offline mode")
	}
	trimmedKey, detail, ok := requireNonEmpty(credentialKey, "credentialKey")
	if !ok {
		service.logger.Warn("認証情報が不正です", "operation", "DeleteGameFromCloud", "detail", detail)
		return newServiceError("認証情報が不正です", detail)
	}
	trimmedID, detail, ok := requireNonEmpty(gameID, "gameID")
	if !ok {
		service.logger.Warn("ゲームIDが不正です", "operation", "DeleteGameFromCloud", "detail", detail, "gameId", gameID)
		return newServiceError("ゲームIDが不正です", detail)
	}

	client, cfg, message, detail, ok := service.newClient(ctx, trimmedKey)
	if !ok {
		service.logger.Warn("S3クライアント初期化に失敗", "operation", "DeleteGameFromCloud", "message", message, "detail", detail)
		return newServiceError(message, detail)
	}

	prefix := fmt.Sprintf("games/%s/", trimmedID)
	if err := service.cloudStorage.DeleteObjectsByPrefix(ctx, client, cfg.Bucket, prefix); err != nil {
		service.logger.Error("クラウドデータ削除に失敗", "error", err, "gameId", trimmedID)
		return newServiceError("クラウドデータ削除に失敗しました", err.Error())
	}

	metadata, err := service.cloudStorage.LoadMetadata(ctx, client, cfg.Bucket, service.config.CloudMetadataKey)
	if err != nil {
		if isNotFoundError(err) {
			return nil
		}
		service.logger.Error("クラウドメタ情報取得に失敗", "error", err)
		return newServiceError("クラウドメタ情報取得に失敗しました", err.Error())
	}

	updatedGames := make([]storage.CloudGameMetadata, 0, len(metadata.Games))
	for _, game := range metadata.Games {
		if game.ID != trimmedID {
			updatedGames = append(updatedGames, game)
		}
	}

	if len(updatedGames) == len(metadata.Games) {
		return nil
	}

	metadata.Games = updatedGames
	metadata.UpdatedAt = time.Now()
	if err := service.cloudStorage.SaveMetadata(ctx, client, cfg.Bucket, service.config.CloudMetadataKey, *metadata); err != nil {
		service.logger.Error("クラウドメタ情報更新に失敗", "error", err)
		return newServiceError("クラウドメタ情報更新に失敗しました", err.Error())
	}

	return nil
}

func (service *CloudSyncService) sync(ctx context.Context, credentialKey string, gameID string) (CloudSyncSummary, error) {
	if service.isOffline() {
		service.logger.Warn("オフラインモードのため同期できません", "operation", "sync", "gameId", gameID)
		return CloudSyncSummary{}, newServiceError("オフラインモードのため同期できません", "offline mode")
	}
	trimmedKey, detail, ok := requireNonEmpty(credentialKey, "credentialKey")
	if !ok {
		service.logger.Warn("認証情報が不正です", "operation", "sync", "detail", detail)
		return CloudSyncSummary{}, newServiceError("認証情報が不正です", detail)
	}

	client, cfg, message, detail, ok := service.newClient(ctx, trimmedKey)
	if !ok {
		service.logger.Warn("S3クライアント初期化に失敗", "operation", "sync", "message", message, "detail", detail)
		return CloudSyncSummary{}, newServiceError(message, detail)
	}

	metadata, err := service.cloudStorage.LoadMetadata(ctx, client, cfg.Bucket, service.config.CloudMetadataKey)
	if err != nil {
		if !isNotFoundError(err) {
			service.logger.Error("クラウドメタ情報取得に失敗", "error", err)
			return CloudSyncSummary{}, newServiceError("クラウドメタ情報取得に失敗しました", err.Error())
		}
		metadata = &storage.CloudMetadata{
			Version: cloudMetadataVersion,
			Games:   []storage.CloudGameMetadata{},
		}
	}

	localGames, err := service.loadLocalGames(ctx, gameID)
	if err != nil {
		service.logger.Error("ローカルゲーム取得に失敗", "error", err)
		return CloudSyncSummary{}, newServiceError("ローカルゲーム取得に失敗しました", err.Error())
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
			return CloudSyncSummary{}, newServiceError(message, err.Error())
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
		if err := service.cloudStorage.SaveMetadata(ctx, client, cfg.Bucket, service.config.CloudMetadataKey, updated); err != nil {
			service.logger.Error("クラウドメタ情報更新に失敗", "error", err)
			return CloudSyncSummary{}, newServiceError("クラウド更新に失敗しました", err.Error())
		}
	}

	return summary, nil
}

type localGameBundle struct {
	Game     domain.Game
	Sessions []domain.PlaySession
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

type mergedSessionsResult struct {
	Sessions        []storage.CloudSessionRecord
	UploadedCount   int
	DownloadedCount int
	Changed         bool
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
	if hasLocal && hasCloud {
		return service.syncExistingGamePair(ctx, client, bucket, gameID, local, cloud)
	}

	switch determineGameSyncAction(local, hasLocal, cloud, hasCloud) {
	case gameSyncActionUpload:
		existing := (*storage.CloudGameMetadata)(nil)
		operation := "sync.createCloudGame"
		if hasCloud {
			existing = &cloud
			operation = "sync.buildCloudGame"
		}
		cloudGame, uploadedImages, err := service.buildCloudGame(ctx, client, bucket, local.Game, composeCloudSessions(local.Sessions), existing)
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
		var currentLocal *domain.Game
		operation := "sync.createLocalGame"
		if hasLocal {
			currentLocal = &local.Game
			operation = "sync.applyCloudGame"
		}
		cloudSessions, err := service.loadCloudSessions(ctx, client, bucket, cloud.ID)
		if err != nil {
			service.logger.Error("クラウドセッション取得に失敗", "operation", operation+".loadCloudSessions", "gameId", gameID, "error", err)
			return gameSyncIterationResult{}, err
		}
		downloadedImages, err := service.applyCloudGame(ctx, client, bucket, cloud, currentLocal, cloudSessions)
		if err != nil {
			service.logger.Error("ローカル更新に失敗", "operation", operation, "gameId", gameID, "error", err)
			return gameSyncIterationResult{}, err
		}
		return gameSyncIterationResult{
			cloudGame: &cloud,
			summary: CloudSyncSummary{
				DownloadedGames:    1,
				DownloadedSessions: len(cloudSessions),
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

type gameSyncState struct {
	mergedSessions  mergedSessionsResult
	mergedGame      domain.Game
	mergedCloudGame storage.CloudGameMetadata
}

func (service *CloudSyncService) prepareGameSyncState(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	gameID string,
	local localGameBundle,
	cloud storage.CloudGameMetadata,
) (gameSyncState, error) {
	cloudSessions, err := service.loadCloudSessions(ctx, client, bucket, gameID)
	if err != nil {
		service.logger.Error("クラウドセッション取得に失敗", "operation", "sync.loadCloudSessions", "gameId", gameID, "error", err)
		return gameSyncState{}, err
	}

	mergedSessions := mergeSessions(local.Sessions, cloudSessions)
	if mergedSessions.Changed {
		if err := service.upsertMergedLocalSessions(ctx, gameID, mergedSessions.Sessions); err != nil {
			service.logger.Error("ローカルセッション統合に失敗", "operation", "sync.upsertMergedLocalSessions", "gameId", gameID, "error", err)
			return gameSyncState{}, err
		}
	}

	mergedGame := service.mergeCloudGameMetadata(cloud, &local.Game, mergedSessions.Sessions)
	return gameSyncState{
		mergedSessions:  mergedSessions,
		mergedGame:      mergedGame,
		mergedCloudGame: cloudMetadataFromGame(mergedGame, cloud.ImageKey),
	}, nil
}

func (service *CloudSyncService) syncUploadPath(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	gameID string,
	state gameSyncState,
	originalCloud *storage.CloudGameMetadata,
) (gameSyncIterationResult, error) {
	cloudGame, uploadedImages, err := service.buildCloudGame(ctx, client, bucket, state.mergedGame, state.mergedSessions.Sessions, originalCloud)
	if err != nil {
		service.logger.Error("クラウド更新に失敗", "operation", "sync.buildCloudGame", "gameId", gameID, "error", err)
		return gameSyncIterationResult{}, err
	}
	return gameSyncIterationResult{
		cloudGame: &cloudGame,
		summary: CloudSyncSummary{
			UploadedGames:      1,
			UploadedSessions:   state.mergedSessions.UploadedCount,
			DownloadedSessions: state.mergedSessions.DownloadedCount,
			UploadedImages:     uploadedImages,
		},
		shouldSaveMetadata: true,
	}, nil
}

func (service *CloudSyncService) syncDownloadPath(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	gameID string,
	state gameSyncState,
	localGame *domain.Game,
) (gameSyncIterationResult, error) {
	if state.mergedSessions.Changed {
		if err := service.cloudStorage.SaveSessions(ctx, client, bucket, cloudSessionsKey(gameID), state.mergedSessions.Sessions); err != nil {
			service.logger.Error("クラウドセッション更新に失敗", "operation", "sync.saveMergedSessions.cloudWins", "gameId", gameID, "error", err)
			return gameSyncIterationResult{}, err
		}
	}
	downloadedImages, err := service.applyCloudGame(ctx, client, bucket, state.mergedCloudGame, localGame, state.mergedSessions.Sessions)
	if err != nil {
		service.logger.Error("ローカル更新に失敗", "operation", "sync.applyCloudGame", "gameId", gameID, "error", err)
		return gameSyncIterationResult{}, err
	}
	return gameSyncIterationResult{
		cloudGame: &state.mergedCloudGame,
		summary: CloudSyncSummary{
			UploadedSessions:   state.mergedSessions.UploadedCount,
			DownloadedGames:    1,
			DownloadedSessions: state.mergedSessions.DownloadedCount,
			DownloadedImages:   downloadedImages,
		},
		shouldSaveMetadata: state.mergedSessions.Changed,
	}, nil
}

func (service *CloudSyncService) syncSkipPath(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	gameID string,
	state gameSyncState,
	localGame *domain.Game,
) (gameSyncIterationResult, error) {
	if state.mergedSessions.Changed {
		if err := service.cloudStorage.SaveSessions(ctx, client, bucket, cloudSessionsKey(gameID), state.mergedSessions.Sessions); err != nil {
			service.logger.Error("クラウドセッション更新に失敗", "operation", "sync.saveMergedSessions.sameTimestamp", "gameId", gameID, "error", err)
			return gameSyncIterationResult{}, err
		}
	}
	downloadedImages, err := service.applyCloudGame(ctx, client, bucket, state.mergedCloudGame, localGame, state.mergedSessions.Sessions)
	if err != nil {
		service.logger.Error("ローカル更新に失敗", "operation", "sync.applyMergedCloudGame", "gameId", gameID, "error", err)
		return gameSyncIterationResult{}, err
	}
	summary := CloudSyncSummary{
		UploadedSessions:   state.mergedSessions.UploadedCount,
		DownloadedSessions: state.mergedSessions.DownloadedCount,
		DownloadedImages:   downloadedImages,
	}
	if !state.mergedSessions.Changed {
		summary.SkippedGames = 1
	}
	return gameSyncIterationResult{
		cloudGame:          &state.mergedCloudGame,
		summary:            summary,
		shouldSaveMetadata: state.mergedSessions.Changed,
	}, nil
}

func (service *CloudSyncService) syncExistingGamePair(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	gameID string,
	local localGameBundle,
	cloud storage.CloudGameMetadata,
) (gameSyncIterationResult, error) {
	state, err := service.prepareGameSyncState(ctx, client, bucket, gameID, local, cloud)
	if err != nil {
		return gameSyncIterationResult{}, err
	}

	switch determineGameSyncAction(local, true, cloud, true) {
	case gameSyncActionUpload:
		return service.syncUploadPath(ctx, client, bucket, gameID, state, &cloud)
	case gameSyncActionDownload:
		return service.syncDownloadPath(ctx, client, bucket, gameID, state, &local.Game)
	default:
		return service.syncSkipPath(ctx, client, bucket, gameID, state, &local.Game)
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

	games, err := service.repository.ListGames(ctx, "", domain.PlayStatus(""), "title", "asc")
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
	game domain.Game,
	sessions []storage.CloudSessionRecord,
	existing *storage.CloudGameMetadata,
) (storage.CloudGameMetadata, int, error) {
	cloudGame := composeCloudGameMetadata(game)

	if err := service.cloudStorage.SaveSessions(ctx, client, bucket, cloudSessionsKey(game.ID), sessions); err != nil {
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
	local *domain.Game,
	cloudSessions []storage.CloudSessionRecord,
) (int, error) {
	imagePath, downloadedImages, err := service.downloadCloudImagePath(ctx, client, bucket, cloud)
	if err != nil {
		return 0, err
	}

	if err := service.upsertSyncedLocalGame(ctx, cloud, local, imagePath); err != nil {
		return 0, err
	}

	if err := service.upsertMergedLocalSessions(ctx, cloud.ID, cloudSessions); err != nil {
		return 0, err
	}

	return downloadedImages, nil
}

func (service *CloudSyncService) upsertSyncedLocalGame(
	ctx context.Context,
	cloud storage.CloudGameMetadata,
	local *domain.Game,
	imagePath *string,
) error {
	game := composeSyncedLocalGame(cloud, local, imagePath)
	return service.repository.UpsertGameSync(ctx, game)
}

func (service *CloudSyncService) loadCloudSessions(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	gameID string,
) ([]storage.CloudSessionRecord, error) {
	key := cloudSessionsKey(gameID)
	sessions, err := service.cloudStorage.LoadSessions(ctx, client, bucket, key)
	if err != nil {
		if isNotFoundError(err) {
			return []storage.CloudSessionRecord{}, nil
		}
		return nil, err
	}
	return sessions, nil
}

func (service *CloudSyncService) upsertMergedLocalSessions(
	ctx context.Context,
	gameID string,
	sessions []storage.CloudSessionRecord,
) error {
	var lastPlayed *time.Time
	var total int64
	for _, session := range sessions {
		playSession := domain.PlaySession{
			ID:          session.ID,
			GameID:      gameID,
			PlayedAt:    session.PlayedAt,
			Duration:    session.Duration,
			SessionName: session.SessionName,
			UpdatedAt:   session.UpdatedAt,
		}
		if err := service.repository.UpsertPlaySessionSync(ctx, playSession); err != nil {
			return err
		}
		total += session.Duration
		if lastPlayed == nil || session.PlayedAt.After(*lastPlayed) {
			playedAt := session.PlayedAt
			lastPlayed = &playedAt
		}
	}
	if lastPlayed != nil {
		return service.repository.UpdateGameTotalPlayTimeWithLastPlayed(ctx, gameID, total, *lastPlayed)
	}
	return service.repository.UpdateGameTotalPlayTime(ctx, gameID, total)
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

func (service *CloudSyncService) newStorageClient(
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
