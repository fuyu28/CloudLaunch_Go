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
	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/storage"

	"github.com/aws/aws-sdk-go-v2/service/s3"
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
		gameService: gameService,
		memoService: memoService,
		logger:      logger,
	}
}

func (service *MemoCloudService) GetCloudMemos(ctx context.Context) result.ApiResult[[]CloudMemoInfo] {
	client, bucket, err := service.getDefaultS3Client(ctx)
	if err != nil {
		service.logger.Error("クラウドメモ取得に失敗しました", "error", err, "operation", "GetCloudMemos.getDefaultS3Client")
		return result.ErrorResult[[]CloudMemoInfo]("クラウドメモ取得に失敗しました", err.Error())
	}
	objects, err := storage.ListObjects(ctx, client, bucket, "games/")
	if err != nil {
		service.logger.Error("クラウドメモ取得に失敗しました", "error", err, "operation", "GetCloudMemos.listObjects", "bucket", bucket)
		return result.ErrorResult[[]CloudMemoInfo]("クラウドメモ取得に失敗しました", err.Error())
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
	return result.OkResult(memos)
}

func (service *MemoCloudService) DownloadMemoFromCloud(ctx context.Context, gameID string, memoFileName string) result.ApiResult[string] {
	client, bucket, err := service.getDefaultS3Client(ctx)
	if err != nil {
		service.logger.Error("メモのダウンロードに失敗しました", "error", err, "operation", "DownloadMemoFromCloud.getDefaultS3Client")
		return result.ErrorResult[string]("メモのダウンロードに失敗しました", err.Error())
	}
	if strings.TrimSpace(gameID) == "" || strings.TrimSpace(memoFileName) == "" {
		service.logger.Warn("メモのダウンロード入力が不正です", "operation", "DownloadMemoFromCloud", "gameId", gameID, "memoFileName", memoFileName)
		return result.ErrorResult[string]("メモのダウンロードに失敗しました", "入力が不正です")
	}
	key := fmt.Sprintf("games/%s/memo/%s", strings.TrimSpace(gameID), memoFileName)
	payload, err := storage.DownloadObject(ctx, client, bucket, key)
	if err != nil {
		service.logger.Error("メモのダウンロードに失敗しました", "error", err, "operation", "DownloadMemoFromCloud.downloadObject", "key", key)
		return result.ErrorResult[string]("メモのダウンロードに失敗しました", err.Error())
	}
	return result.OkResult(string(payload))
}

func (service *MemoCloudService) UploadMemoToCloud(ctx context.Context, memoID string) result.ApiResult[bool] {
	client, bucket, err := service.getDefaultS3Client(ctx)
	if err != nil {
		service.logger.Error("メモのアップロードに失敗しました", "error", err, "operation", "UploadMemoToCloud.getDefaultS3Client")
		return result.ErrorResult[bool]("メモのアップロードに失敗しました", err.Error())
	}
	memoResult := service.memoService.GetMemoByID(ctx, strings.TrimSpace(memoID))
	if !memoResult.Success {
		return convertServiceError[bool](memoResult, "メモの取得に失敗しました")
	}
	memoData := memoResult.Data
	if memoData == nil {
		service.logger.Warn("メモが見つかりません", "operation", "UploadMemoToCloud", "memoId", memoID)
		return result.ErrorResult[bool]("メモが見つかりません", "指定されたIDが存在しません")
	}
	gameResult := service.gameService.GetGameByID(ctx, memoData.GameID)
	if !gameResult.Success {
		return convertServiceError[bool](gameResult, "ゲームの取得に失敗しました")
	}
	game := gameResult.Data
	if game == nil {
		service.logger.Warn("ゲームが見つかりません", "operation", "UploadMemoToCloud", "gameId", memoData.GameID)
		return result.ErrorResult[bool]("ゲームが見つかりません", "指定されたIDが存在しません")
	}

	key := memo.BuildMemoPath(game.ID, memoData.Title, memoData.ID)
	payload := memo.GenerateCloudMemoFileContent(memoData.Title, memoData.Content, game.Title)
	if err := storage.UploadBytes(ctx, client, bucket, key, []byte(payload), "text/markdown"); err != nil {
		service.logger.Error("メモのアップロードに失敗しました", "error", err, "operation", "UploadMemoToCloud.uploadBytes", "key", key)
		return result.ErrorResult[bool]("メモのアップロードに失敗しました", err.Error())
	}
	return result.OkResult(true)
}

func (service *MemoCloudService) SyncMemosFromCloud(ctx context.Context, gameID string) result.ApiResult[MemoSyncResult] {
	client, bucket, err := service.getDefaultS3Client(ctx)
	if err != nil {
		service.logger.Error("メモ同期に失敗しました", "error", err, "operation", "SyncMemosFromCloud.getDefaultS3Client")
		return result.ErrorResult[MemoSyncResult]("メモ同期に失敗しました", err.Error())
	}

	resultData := MemoSyncResult{
		Success: true,
		Details: []string{},
	}

	cloudMemosResult := service.GetCloudMemos(ctx)
	if !cloudMemosResult.Success {
		service.logger.Warn("クラウドメモ取得に失敗しました", "operation", "SyncMemosFromCloud", "detail", cloudMemosResult.Error)
		return convertServiceError[MemoSyncResult](cloudMemosResult, "メモ同期に失敗しました")
	}
	cloudMemos := cloudMemosResult.Data
	if cloudMemos == nil {
		cloudMemos = []CloudMemoInfo{}
	}

	var targetGame *models.Game
	if strings.TrimSpace(gameID) != "" {
		gameResult := service.gameService.GetGameByID(ctx, strings.TrimSpace(gameID))
		if !gameResult.Success {
			return convertServiceError[MemoSyncResult](gameResult, "メモ同期に失敗しました")
		}
		game := gameResult.Data
		if game == nil {
			service.logger.Warn("指定されたゲームが見つかりません", "operation", "SyncMemosFromCloud", "gameId", gameID)
			return result.ErrorResult[MemoSyncResult]("メモ同期に失敗しました", "指定されたゲームが見つかりません")
		}
		targetGame = game
	}

	cloudMap := map[string]CloudMemoInfo{}
	for _, cloudMemo := range cloudMemos {
		cloudMap[fmt.Sprintf("%s:%s", cloudMemo.GameID, cloudMemo.MemoID)] = cloudMemo
	}

	gamesResult := service.gameService.ListGames(ctx, "", models.PlayStatus(""), "title", "asc")
	if !gamesResult.Success {
		return convertServiceError[MemoSyncResult](gamesResult, "メモ同期に失敗しました")
	}
	games := gamesResult.Data
	gameByID := map[string]models.Game{}
	for _, game := range games {
		gameByID[game.ID] = game
	}

	localMemosResult := service.fetchLocalMemos(ctx, gameID)
	if !localMemosResult.Success {
		return convertServiceError[MemoSyncResult](localMemosResult, "メモ同期に失敗しました")
	}
	localMemos := localMemosResult.Data

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
			if err := uploadMemoContent(ctx, client, bucket, game, localMemo); err != nil {
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
			if err := uploadMemoContent(ctx, client, bucket, game, localMemo); err != nil {
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

		payload, err := storage.DownloadObject(ctx, client, bucket, cloudMemo.Key)
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
		if err := uploadMemoContent(ctx, client, bucket, game, localMemo); err != nil {
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
		payload, err := storage.DownloadObject(ctx, client, bucket, cloudMemo.Key)
		if err != nil {
			resultData.Skipped++
			resultData.Details = append(resultData.Details, fmt.Sprintf("ダウンロード失敗: %s", cloudMemo.MemoTitle))
			continue
		}
		content := memo.ExtractMemoContent(string(payload))

		existingMemoResult := service.memoService.GetMemoByID(ctx, cloudMemo.MemoID)
		if !existingMemoResult.Success {
			resultData.Skipped++
			resultData.Details = append(resultData.Details, fmt.Sprintf("メモ取得失敗: %s", cloudMemo.MemoTitle))
			continue
		}
		existingMemo := existingMemoResult.Data
		if existingMemo == nil || existingMemo.GameID != game.ID {
			findMemoResult := service.memoService.FindMemoByTitle(ctx, game.ID, cloudMemo.MemoTitle)
			if !findMemoResult.Success {
				resultData.Skipped++
				resultData.Details = append(resultData.Details, fmt.Sprintf("メモ検索失敗: %s", cloudMemo.MemoTitle))
				continue
			}
			existingMemo = findMemoResult.Data
		}

		if existingMemo == nil {
			createdResult := service.memoService.CreateMemo(ctx, MemoInput{
				Title:   cloudMemo.MemoTitle,
				Content: content,
				GameID:  game.ID,
			})
			if !createdResult.Success || createdResult.Data == nil {
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
			updatedResult := service.memoService.UpdateMemo(ctx, existingMemo.ID, MemoUpdateInput{
				Title:   existingMemo.Title,
				Content: content,
			})
			if !updatedResult.Success {
				resultData.Skipped++
				resultData.Details = append(resultData.Details, fmt.Sprintf("更新失敗: %s", cloudMemo.MemoTitle))
				continue
			}
			resultData.LocalOverwritten++
			processed[key] = true
			continue
		}

		if err := uploadMemoContent(ctx, client, bucket, game, *existingMemo); err != nil {
			resultData.Details = append(resultData.Details, fmt.Sprintf("クラウド更新失敗: %s", existingMemo.Title))
			continue
		}
		resultData.CloudOverwritten++
		processed[key] = true
	}

	return result.OkResult(resultData)
}

func uploadMemoContent(ctx context.Context, client *s3.Client, bucket string, game models.Game, memoData models.Memo) error {
	key := memo.BuildMemoPath(game.ID, memoData.Title, memoData.ID)
	payload := memo.GenerateCloudMemoFileContent(memoData.Title, memoData.Content, game.Title)
	return storage.UploadBytes(ctx, client, bucket, key, []byte(payload), "text/markdown")
}

func (service *MemoCloudService) fetchLocalMemos(ctx context.Context, gameID string) result.ApiResult[[]models.Memo] {
	if strings.TrimSpace(gameID) == "" {
		return service.memoService.ListAllMemos(ctx)
	}
	return service.memoService.ListMemosByGame(ctx, strings.TrimSpace(gameID))
}

func (service *MemoCloudService) getDefaultS3Client(ctx context.Context) (*s3.Client, string, error) {
	cfg, credential, err := service.resolveDefaultS3Config(ctx)
	if err != nil {
		return nil, "", err
	}
	client, err := storage.NewClient(ctx, cfg, credential)
	if err != nil {
		return nil, "", err
	}
	return client, cfg.Bucket, nil
}

func (service *MemoCloudService) resolveDefaultS3Config(ctx context.Context) (storage.S3Config, credentials.Credential, error) {
	credential, err := service.store.Load(ctx, "default")
	if err != nil || credential == nil {
		return storage.S3Config{}, credentials.Credential{}, errors.New("認証情報がありません")
	}
	return resolveS3Config(service.config, credential), *credential, nil
}

func convertServiceError[T any, U any](serviceResult result.ApiResult[U], fallbackMessage string) result.ApiResult[T] {
	if serviceResult.Error == nil {
		return result.ErrorResult[T](fallbackMessage, "不明なエラーです")
	}
	message := serviceResult.Error.Message
	if strings.TrimSpace(message) == "" {
		message = fallbackMessage
	}
	return result.ErrorResult[T](message, serviceResult.Error.Detail)
}
