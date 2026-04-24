// @fileoverview メモのクラウド同期APIを提供する。
package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"CloudLaunch_Go/internal/memo"
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/services"
	"CloudLaunch_Go/internal/storage"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// CloudMemoInfo はクラウド上のメモ情報を表す。
type CloudMemoInfo struct {
	Key          string    `json:"key"`
	FileName     string    `json:"fileName"`
	GameID       string    `json:"gameId"`
	MemoTitle    string    `json:"memoTitle"`
	MemoID       string    `json:"memoId"`
	LastModified time.Time `json:"lastModified"`
	Size         int64     `json:"size"`
}

// MemoSyncResult はメモ同期の結果を表す。
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

// GetCloudMemos はクラウドメモ一覧を取得する。
func (app *App) GetCloudMemos() result.ApiResult[[]CloudMemoInfo] {
	ctx := app.context()
	client, bucket, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return errorResultWithLog[[]CloudMemoInfo](app, "クラウドメモ取得に失敗しました", error, "operation", "GetCloudMemos.getDefaultS3Client")
	}
	objects, error := storage.ListObjects(ctx, client, bucket, "games/")
	if error != nil {
		return errorResultWithLog[[]CloudMemoInfo](app, "クラウドメモ取得に失敗しました", error, "operation", "GetCloudMemos.listObjects", "bucket", bucket)
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

// DownloadMemoFromCloud はクラウドからメモ内容を取得する。
func (app *App) DownloadMemoFromCloud(gameID string, memoFileName string) result.ApiResult[string] {
	ctx := app.context()
	client, bucket, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return errorResultWithLog[string](app, "メモのダウンロードに失敗しました", error, "operation", "DownloadMemoFromCloud.getDefaultS3Client")
	}
	if strings.TrimSpace(gameID) == "" || strings.TrimSpace(memoFileName) == "" {
		app.Logger.Warn("メモのダウンロード入力が不正です", "operation", "DownloadMemoFromCloud", "gameId", gameID, "memoFileName", memoFileName)
		return result.ErrorResult[string]("メモのダウンロードに失敗しました", "入力が不正です")
	}
	key := fmt.Sprintf("games/%s/memo/%s", strings.TrimSpace(gameID), memoFileName)
	payload, error := storage.DownloadObject(ctx, client, bucket, key)
	if error != nil {
		return errorResultWithLog[string](app, "メモのダウンロードに失敗しました", error, "operation", "DownloadMemoFromCloud.downloadObject", "key", key)
	}
	return result.OkResult(string(payload))
}

// UploadMemoToCloud はメモをクラウドへ保存する。
func (app *App) UploadMemoToCloud(memoID string) result.ApiResult[bool] {
	ctx := app.context()
	client, bucket, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return errorResultWithLog[bool](app, "メモのアップロードに失敗しました", error, "operation", "UploadMemoToCloud.getDefaultS3Client")
	}
	memoResult := app.MemoService.GetMemoByID(ctx, strings.TrimSpace(memoID))
	if !memoResult.Success {
		return serviceErrorResult[bool](memoResult, "メモの取得に失敗しました")
	}
	memoData := memoResult.Data
	if memoData == nil {
		app.Logger.Warn("メモが見つかりません", "operation", "UploadMemoToCloud", "memoId", memoID)
		return result.ErrorResult[bool]("メモが見つかりません", "指定されたIDが存在しません")
	}
	gameResult := app.GameService.GetGameByID(ctx, memoData.GameID)
	if !gameResult.Success {
		return serviceErrorResult[bool](gameResult, "ゲームの取得に失敗しました")
	}
	game := gameResult.Data
	if game == nil {
		app.Logger.Warn("ゲームが見つかりません", "operation", "UploadMemoToCloud", "gameId", memoData.GameID)
		return result.ErrorResult[bool]("ゲームが見つかりません", "指定されたIDが存在しません")
	}

	key := memo.BuildMemoPath(game.ID, memoData.Title, memoData.ID)
	payload := memo.GenerateCloudMemoFileContent(memoData.Title, memoData.Content, game.Title)
	if error := storage.UploadBytes(ctx, client, bucket, key, []byte(payload), "text/markdown"); error != nil {
		return errorResultWithLog[bool](app, "メモのアップロードに失敗しました", error, "operation", "UploadMemoToCloud.uploadBytes", "key", key)
	}
	return result.OkResult(true)
}

// SyncMemosFromCloud はメモをクラウドと同期する。
func (app *App) SyncMemosFromCloud(gameID string) result.ApiResult[MemoSyncResult] {
	ctx := app.context()
	client, bucket, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return errorResultWithLog[MemoSyncResult](app, "メモ同期に失敗しました", error, "operation", "SyncMemosFromCloud.getDefaultS3Client")
	}

	resultData := MemoSyncResult{
		Success: true,
		Details: []string{},
	}

	cloudMemosResult := app.GetCloudMemos()
	if !cloudMemosResult.Success {
		app.Logger.Warn("クラウドメモ取得に失敗しました", "operation", "SyncMemosFromCloud", "detail", cloudMemosResult.Error)
		if cloudMemosResult.Error == nil {
			return result.ErrorResult[MemoSyncResult]("メモ同期に失敗しました", "クラウドメモ取得に失敗しました")
		}
		return result.ErrorResult[MemoSyncResult]("メモ同期に失敗しました", cloudMemosResult.Error.Detail)
	}
	cloudMemos := cloudMemosResult.Data
	if cloudMemos == nil {
		cloudMemos = []CloudMemoInfo{}
	}

	var targetGame *models.Game
	if strings.TrimSpace(gameID) != "" {
		gameResult := app.GameService.GetGameByID(ctx, strings.TrimSpace(gameID))
		if !gameResult.Success {
			return serviceErrorResult[MemoSyncResult](gameResult, "メモ同期に失敗しました")
		}
		game := gameResult.Data
		if game == nil {
			app.Logger.Warn("指定されたゲームが見つかりません", "operation", "SyncMemosFromCloud", "gameId", gameID)
			return result.ErrorResult[MemoSyncResult]("メモ同期に失敗しました", "指定されたゲームが見つかりません")
		}
		targetGame = game
	}

	cloudMap := map[string]CloudMemoInfo{}
	for _, cloudMemo := range cloudMemos {
		cloudMap[fmt.Sprintf("%s:%s", cloudMemo.GameID, cloudMemo.MemoID)] = cloudMemo
	}

	gamesResult := app.GameService.ListGames(ctx, "", models.PlayStatus(""), "title", "asc")
	if !gamesResult.Success {
		return serviceErrorResult[MemoSyncResult](gamesResult, "メモ同期に失敗しました")
	}
	games := gamesResult.Data
	gameByID := map[string]models.Game{}
	for _, game := range games {
		gameByID[game.ID] = game
	}

	localMemosResult := app.fetchLocalMemos(ctx, gameID)
	if !localMemosResult.Success {
		return serviceErrorResult[MemoSyncResult](localMemosResult, "メモ同期に失敗しました")
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
			if error := uploadMemoContent(ctx, client, bucket, game, localMemo); error != nil {
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
			if error := uploadMemoContent(ctx, client, bucket, game, localMemo); error != nil {
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

		payload, error := storage.DownloadObject(ctx, client, bucket, cloudMemo.Key)
		if error != nil {
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
		if error := uploadMemoContent(ctx, client, bucket, game, localMemo); error != nil {
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
		payload, error := storage.DownloadObject(ctx, client, bucket, cloudMemo.Key)
		if error != nil {
			resultData.Skipped++
			resultData.Details = append(resultData.Details, fmt.Sprintf("ダウンロード失敗: %s", cloudMemo.MemoTitle))
			continue
		}
		content := memo.ExtractMemoContent(string(payload))

		existingMemoResult := app.MemoService.GetMemoByID(ctx, cloudMemo.MemoID)
		if !existingMemoResult.Success {
			resultData.Skipped++
			resultData.Details = append(resultData.Details, fmt.Sprintf("メモ取得失敗: %s", cloudMemo.MemoTitle))
			continue
		}
		existingMemo := existingMemoResult.Data
		if existingMemo == nil || existingMemo.GameID != game.ID {
			findMemoResult := app.MemoService.FindMemoByTitle(ctx, game.ID, cloudMemo.MemoTitle)
			if !findMemoResult.Success {
				resultData.Skipped++
				resultData.Details = append(resultData.Details, fmt.Sprintf("メモ検索失敗: %s", cloudMemo.MemoTitle))
				continue
			}
			existingMemo = findMemoResult.Data
		}

		if existingMemo == nil {
			createdResult := app.MemoService.CreateMemo(ctx, servicesMemoInputFromCloud(cloudMemo.MemoTitle, content, game.ID))
			if !createdResult.Success {
				resultData.Skipped++
				resultData.Details = append(resultData.Details, fmt.Sprintf("作成失敗: %s", cloudMemo.MemoTitle))
				continue
			}
			created := createdResult.Data
			if created == nil {
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
			updatedResult := app.MemoService.UpdateMemo(ctx, existingMemo.ID, servicesMemoUpdateInputFromCloud(existingMemo.Title, content))
			if !updatedResult.Success {
				resultData.Skipped++
				resultData.Details = append(resultData.Details, fmt.Sprintf("更新失敗: %s", cloudMemo.MemoTitle))
				continue
			}
			resultData.LocalOverwritten++
			processed[key] = true
			continue
		}

		if error := uploadMemoContent(ctx, client, bucket, game, *existingMemo); error != nil {
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

func (app *App) fetchLocalMemos(ctx context.Context, gameID string) result.ApiResult[[]models.Memo] {
	if strings.TrimSpace(gameID) == "" {
		return app.MemoService.ListAllMemos(ctx)
	}
	return app.MemoService.ListMemosByGame(ctx, strings.TrimSpace(gameID))
}

func serviceErrorResult[T any, U any](serviceResult result.ApiResult[U], fallbackMessage string) result.ApiResult[T] {
	if serviceResult.Error == nil {
		return result.ErrorResult[T](fallbackMessage, "不明なエラーです")
	}
	message := serviceResult.Error.Message
	if strings.TrimSpace(message) == "" {
		message = fallbackMessage
	}
	return result.ErrorResult[T](message, serviceResult.Error.Detail)
}

func servicesMemoInputFromCloud(title string, content string, gameID string) services.MemoInput {
	return services.MemoInput{
		Title:   title,
		Content: content,
		GameID:  gameID,
	}
}

func servicesMemoUpdateInputFromCloud(title string, content string) services.MemoUpdateInput {
	return services.MemoUpdateInput{
		Title:   title,
		Content: content,
	}
}
