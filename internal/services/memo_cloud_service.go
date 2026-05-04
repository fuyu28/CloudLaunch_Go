package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/credentials"
	"CloudLaunch_Go/internal/memo"
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/storage"
)

type CloudMemoInfo struct {
	Key          string    `json:"key"`
	FileName     string    `json:"fileName"`
	GameID       string    `json:"gameId"`
	MemoTitle    string    `json:"memoTitle"`
	MemoID       string    `json:"memoId"`
	LastModified time.Time `json:"lastModified"`
	Size         int64     `json:"size"`
}

type MemoSyncResult struct {
	Success          bool     `json:"success"`
	Uploaded         int      `json:"uploaded"`
	LocalOverwritten int      `json:"localOverwritten"`
	CloudOverwritten int      `json:"cloudOverwritten"`
	Created          int      `json:"created"`
	Updated          int      `json:"updated"`
	Skipped          int      `json:"skipped"`
	Error            *string  `json:"error,omitempty"`
	Details          []string `json:"details"`
}

type MemoCloudService struct {
	config      config.Config
	store       credentials.Store
	objectStore cloudObjectStore
	gameService *GameService
	memoService *MemoService
	logger      *slog.Logger
}

func NewMemoCloudService(
	cfg config.Config,
	store credentials.Store,
	gameService *GameService,
	memoService *MemoService,
	logger *slog.Logger,
) *MemoCloudService {
	return &MemoCloudService{
		config:      cfg,
		store:       store,
		objectStore: storageCloudObjectStore{},
		gameService: gameService,
		memoService: memoService,
		logger:      logger,
	}
}

func (service *MemoCloudService) GetCloudMemos(ctx context.Context) ([]CloudMemoInfo, error) {
	cfg, credential, err := service.resolveDefaultS3Config(ctx)
	if err != nil {
		service.logger.Error("クラウドメモ取得に失敗しました", "error", err, "operation", "GetCloudMemos.getDefaultS3Client")
		return nil, newServiceError("クラウドメモ取得に失敗しました", err.Error())
	}
	objects, err := service.objectStore.ListObjects(ctx, cfg, credential, "games/")
	if err != nil {
		service.logger.Error("クラウドメモ取得に失敗しました", "error", err, "operation", "GetCloudMemos.listObjects", "bucket", cfg.Bucket)
		return nil, newServiceError("クラウドメモ取得に失敗しました", err.Error())
	}

	memos := make([]CloudMemoInfo, 0)
	for _, obj := range objects {
		if !memo.IsMemoPath(obj.Key) {
			continue
		}
		gameID, memoTitle, memoID, ok := memo.ExtractMemoInfo(obj.Key)
		if !ok {
			continue
		}
		fileName := obj.Key[strings.LastIndex(obj.Key, "/")+1:]
		memos = append(memos, CloudMemoInfo{
			Key:          obj.Key,
			FileName:     fileName,
			GameID:       gameID,
			MemoTitle:    memoTitle,
			MemoID:       memoID,
			LastModified: time.UnixMilli(obj.LastModified),
			Size:         obj.Size,
		})
	}
	return memos, nil
}

func (service *MemoCloudService) DownloadMemoFromCloud(ctx context.Context, gameID string, memoFileName string) (string, error) {
	cfg, credential, err := service.resolveDefaultS3Config(ctx)
	if err != nil {
		service.logger.Error("メモのダウンロードに失敗しました", "error", err, "operation", "DownloadMemoFromCloud.getDefaultS3Client")
		return "", newServiceError("メモのダウンロードに失敗しました", err.Error())
	}
	if strings.TrimSpace(gameID) == "" || strings.TrimSpace(memoFileName) == "" {
		service.logger.Warn("メモのダウンロード入力が不正です", "operation", "DownloadMemoFromCloud", "gameId", gameID, "memoFileName", memoFileName)
		return "", newServiceError("メモのダウンロードに失敗しました", "入力が不正です")
	}
	key := fmt.Sprintf("games/%s/memo/%s", strings.TrimSpace(gameID), memoFileName)
	payload, err := service.objectStore.DownloadObject(ctx, cfg, credential, key)
	if err != nil {
		service.logger.Error("メモのダウンロードに失敗しました", "error", err, "operation", "DownloadMemoFromCloud.downloadObject", "key", key)
		return "", newServiceError("メモのダウンロードに失敗しました", err.Error())
	}
	return string(payload), nil
}

func (service *MemoCloudService) UploadMemoToCloud(ctx context.Context, memoID string) error {
	cfg, credential, err := service.resolveDefaultS3Config(ctx)
	if err != nil {
		service.logger.Error("メモのアップロードに失敗しました", "error", err, "operation", "UploadMemoToCloud.getDefaultS3Client")
		return newServiceError("メモのアップロードに失敗しました", err.Error())
	}
	memoData, err := service.memoService.GetMemoByID(ctx, strings.TrimSpace(memoID))
	if err != nil {
		return wrapServiceError(err, "メモの取得に失敗しました")
	}
	if memoData == nil {
		service.logger.Warn("メモが見つかりません", "operation", "UploadMemoToCloud", "memoId", memoID)
		return newServiceError("メモが見つかりません", "指定されたIDが存在しません")
	}
	game, err := service.gameService.GetGameByID(ctx, memoData.GameID)
	if err != nil {
		return wrapServiceError(err, "ゲームの取得に失敗しました")
	}
	if game == nil {
		service.logger.Warn("ゲームが見つかりません", "operation", "UploadMemoToCloud", "gameId", memoData.GameID)
		return newServiceError("ゲームが見つかりません", "指定されたIDが存在しません")
	}

	key := memo.BuildMemoPath(game.ID, memoData.Title, memoData.ID)
	payload := memo.GenerateCloudMemoFileContent(memoData.Title, memoData.Content, game.Title)
	if err := service.objectStore.UploadBytes(ctx, cfg, credential, key, []byte(payload), "text/markdown"); err != nil {
		service.logger.Error("メモのアップロードに失敗しました", "error", err, "operation", "UploadMemoToCloud.uploadBytes", "key", key)
		return newServiceError("メモのアップロードに失敗しました", err.Error())
	}
	return nil
}

func (service *MemoCloudService) SyncMemosFromCloud(ctx context.Context, gameID string) (MemoSyncResult, error) {
	cfg, credential, err := service.resolveDefaultS3Config(ctx)
	if err != nil {
		service.logger.Error("メモ同期に失敗しました", "error", err, "operation", "SyncMemosFromCloud.getDefaultS3Client")
		return MemoSyncResult{}, newServiceError("メモ同期に失敗しました", err.Error())
	}

	resultData := MemoSyncResult{
		Success: true,
		Details: []string{},
	}

	cloudMemos, err := service.GetCloudMemos(ctx)
	if err != nil {
		service.logger.Warn("クラウドメモ取得に失敗しました", "operation", "SyncMemosFromCloud", "detail", err)
		return MemoSyncResult{}, wrapServiceError(err, "メモ同期に失敗しました")
	}
	if cloudMemos == nil {
		cloudMemos = []CloudMemoInfo{}
	}

	var targetGame *models.Game
	if strings.TrimSpace(gameID) != "" {
		game, err := service.gameService.GetGameByID(ctx, strings.TrimSpace(gameID))
		if err != nil {
			return MemoSyncResult{}, wrapServiceError(err, "メモ同期に失敗しました")
		}
		if game == nil {
			service.logger.Warn("指定されたゲームが見つかりません", "operation", "SyncMemosFromCloud", "gameId", gameID)
			return MemoSyncResult{}, newServiceError("メモ同期に失敗しました", "指定されたゲームが見つかりません")
		}
		targetGame = game
	}

	cloudMap := map[string]CloudMemoInfo{}
	for _, cloudMemo := range cloudMemos {
		cloudMap[fmt.Sprintf("%s:%s", cloudMemo.GameID, cloudMemo.MemoID)] = cloudMemo
	}

	games, err := service.gameService.ListGames(ctx, "", models.PlayStatus(""), "title", "asc")
	if err != nil {
		return MemoSyncResult{}, wrapServiceError(err, "メモ同期に失敗しました")
	}
	gameByID := map[string]models.Game{}
	for _, game := range games {
		gameByID[game.ID] = game
	}

	localMemos, err := service.fetchLocalMemos(ctx, gameID)
	if err != nil {
		return MemoSyncResult{}, wrapServiceError(err, "メモ同期に失敗しました")
	}

	processed := map[string]bool{}

	for _, localMemo := range localMemos {
		game, ok := gameByID[localMemo.GameID]
		if !ok {
			resultData.Skipped++
			resultData.Details = append(resultData.Details, fmt.Sprintf("ゲームが見つからないためスキップ: %s", localMemo.Title))
			continue
		}
		if targetGame != nil && game.ID != targetGame.ID {
			continue
		}
		key := fmt.Sprintf("%s:%s", game.ID, localMemo.ID)
		cloudMemo, exists := cloudMap[key]
		if !exists {
			if err := service.uploadMemoContent(ctx, cfg, credential, game, localMemo); err != nil {
				resultData.Details = append(resultData.Details, fmt.Sprintf("アップロード失敗: %s", localMemo.Title))
				continue
			}
			resultData.Uploaded++
			processed[key] = true
			continue
		}

		localUpdated := localMemo.UpdatedAt
		cloudUpdated := cloudMemo.LastModified
		if localUpdated.After(cloudUpdated) {
			if err := service.uploadMemoContent(ctx, cfg, credential, game, localMemo); err != nil {
				resultData.Details = append(resultData.Details, fmt.Sprintf("クラウド更新失敗: %s", localMemo.Title))
				continue
			}
			resultData.CloudOverwritten++
			processed[key] = true
			continue
		}
		if cloudUpdated.After(localUpdated) {
			continue
		}

		payload, err := service.objectStore.DownloadObject(ctx, cfg, credential, cloudMemo.Key)
		if err != nil {
			resultData.Skipped++
			resultData.Details = append(resultData.Details, fmt.Sprintf("クラウド取得失敗: %s", localMemo.Title))
			processed[key] = true
			continue
		}
		cloudContent := memo.ExtractMemoContent(string(payload))
		if memo.CalculateContentHash(localMemo.Content) == memo.CalculateContentHash(cloudContent) {
			resultData.Skipped++
			processed[key] = true
			continue
		}
		if err := service.uploadMemoContent(ctx, cfg, credential, game, localMemo); err != nil {
			resultData.Details = append(resultData.Details, fmt.Sprintf("クラウド更新失敗: %s", localMemo.Title))
			continue
		}
		resultData.CloudOverwritten++
		processed[key] = true
	}

	for _, cloudMemo := range cloudMemos {
		if targetGame != nil && cloudMemo.GameID != targetGame.ID {
			continue
		}
		key := fmt.Sprintf("%s:%s", cloudMemo.GameID, cloudMemo.MemoID)
		if processed[key] {
			continue
		}
		game, ok := gameByID[cloudMemo.GameID]
		if !ok {
			resultData.Skipped++
			resultData.Details = append(resultData.Details, fmt.Sprintf("ゲームが見つからないためスキップ: %s", cloudMemo.MemoTitle))
			continue
		}
		payload, err := service.objectStore.DownloadObject(ctx, cfg, credential, cloudMemo.Key)
		if err != nil {
			resultData.Skipped++
			resultData.Details = append(resultData.Details, fmt.Sprintf("ダウンロード失敗: %s", cloudMemo.MemoTitle))
			continue
		}
		content := memo.ExtractMemoContent(string(payload))

		existingMemo, err := service.memoService.GetMemoByID(ctx, cloudMemo.MemoID)
		if err != nil {
			resultData.Skipped++
			resultData.Details = append(resultData.Details, fmt.Sprintf("メモ取得失敗: %s", cloudMemo.MemoTitle))
			continue
		}
		if existingMemo == nil || existingMemo.GameID != game.ID {
			existingMemo, err = service.memoService.FindMemoByTitle(ctx, game.ID, cloudMemo.MemoTitle)
			if err != nil {
				resultData.Skipped++
				resultData.Details = append(resultData.Details, fmt.Sprintf("メモ検索失敗: %s", cloudMemo.MemoTitle))
				continue
			}
		}

		if existingMemo == nil {
			createdMemo, err := service.memoService.CreateMemo(ctx, MemoInput{
				Title:   cloudMemo.MemoTitle,
				Content: content,
				GameID:  game.ID,
			})
			if err != nil || createdMemo == nil {
				resultData.Skipped++
				resultData.Details = append(resultData.Details, fmt.Sprintf("作成失敗: %s", cloudMemo.MemoTitle))
				continue
			}
			resultData.Created++
			processed[key] = true
			continue
		}

		if memo.CalculateContentHash(existingMemo.Content) == memo.CalculateContentHash(content) {
			resultData.Skipped++
			processed[key] = true
			continue
		}
		if cloudMemo.LastModified.After(existingMemo.UpdatedAt) || cloudMemo.LastModified.Equal(existingMemo.UpdatedAt) {
			_, err := service.memoService.UpdateMemo(ctx, existingMemo.ID, MemoUpdateInput{
				Title:   existingMemo.Title,
				Content: content,
			})
			if err != nil {
				resultData.Skipped++
				resultData.Details = append(resultData.Details, fmt.Sprintf("更新失敗: %s", cloudMemo.MemoTitle))
				continue
			}
			resultData.LocalOverwritten++
			processed[key] = true
			continue
		}

		if err := service.uploadMemoContent(ctx, cfg, credential, game, *existingMemo); err != nil {
			resultData.Details = append(resultData.Details, fmt.Sprintf("クラウド更新失敗: %s", existingMemo.Title))
			continue
		}
		resultData.CloudOverwritten++
		processed[key] = true
	}

	return resultData, nil
}

func (service *MemoCloudService) uploadMemoContent(
	ctx context.Context,
	cfg storage.S3Config,
	credential credentials.Credential,
	game models.Game,
	memoData models.Memo,
) error {
	key := memo.BuildMemoPath(game.ID, memoData.Title, memoData.ID)
	payload := memo.GenerateCloudMemoFileContent(memoData.Title, memoData.Content, game.Title)
	return service.objectStore.UploadBytes(ctx, cfg, credential, key, []byte(payload), "text/markdown")
}

func (service *MemoCloudService) fetchLocalMemos(ctx context.Context, gameID string) ([]models.Memo, error) {
	if strings.TrimSpace(gameID) == "" {
		return service.memoService.ListAllMemos(ctx)
	}
	return service.memoService.ListMemosByGame(ctx, strings.TrimSpace(gameID))
}

func (service *MemoCloudService) resolveDefaultS3Config(ctx context.Context) (storage.S3Config, credentials.Credential, error) {
	credential, err := service.store.Load(ctx, "default")
	if err != nil || credential == nil {
		return storage.S3Config{}, credentials.Credential{}, errors.New("認証情報がありません")
	}
	return resolveS3Config(service.config, credential), *credential, nil
}

func wrapServiceError(err error, fallbackMessage string) error {
	if err == nil {
		return newServiceError(fallbackMessage, "不明なエラーです")
	}
	serviceErr := &ServiceError{}
	if errors.As(err, &serviceErr) {
		message := serviceErr.Message
		if strings.TrimSpace(message) == "" {
			message = fallbackMessage
		}
		return newServiceError(message, serviceErr.Detail)
	}
	return newServiceError(fallbackMessage, err.Error())
}
