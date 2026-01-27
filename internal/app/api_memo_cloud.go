// @fileoverview メモのクラウド同期APIを提供する。
package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"CloudLaunch_Go/internal/db"
	"CloudLaunch_Go/internal/memo"
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/storage"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// CloudMemoInfo はクラウド上のメモ情報を表す。
type CloudMemoInfo struct {
	Key          string    `json:"key"`
	FileName     string    `json:"fileName"`
	GameTitle    string    `json:"gameTitle"`
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
	client, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return result.ErrorResult[[]CloudMemoInfo]("クラウドメモ取得に失敗しました", error.Error())
	}
	objects, error := storage.ListObjects(ctx, client, app.Config.S3Bucket, "games/")
	if error != nil {
		return result.ErrorResult[[]CloudMemoInfo]("クラウドメモ取得に失敗しました", error.Error())
	}

	memos := make([]CloudMemoInfo, 0)
	for _, obj := range objects {
		if !memo.IsMemoPath(obj.Key) {
			continue
		}
		gameTitle, memoTitle, memoID, ok := memo.ExtractMemoInfo(obj.Key)
		if !ok {
			continue
		}
		fileName := obj.Key[strings.LastIndex(obj.Key, "/")+1:]
		memos = append(memos, CloudMemoInfo{
			Key:          obj.Key,
			FileName:     fileName,
			GameTitle:    gameTitle,
			MemoTitle:    memoTitle,
			MemoID:       memoID,
			LastModified: time.UnixMilli(obj.LastModified),
			Size:         obj.Size,
		})
	}

	return result.OkResult(memos)
}

// DownloadMemoFromCloud はクラウドからメモ内容を取得する。
func (app *App) DownloadMemoFromCloud(gameTitle string, memoFileName string) result.ApiResult[string] {
	ctx := app.context()
	client, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return result.ErrorResult[string]("メモのダウンロードに失敗しました", error.Error())
	}
	if strings.TrimSpace(gameTitle) == "" || strings.TrimSpace(memoFileName) == "" {
		return result.ErrorResult[string]("メモのダウンロードに失敗しました", "入力が不正です")
	}
	key := fmt.Sprintf("games/%s/memo/%s", memo.SanitizeForCloudPath(gameTitle), memoFileName)
	payload, error := storage.DownloadObject(ctx, client, app.Config.S3Bucket, key)
	if error != nil {
		return result.ErrorResult[string]("メモのダウンロードに失敗しました", error.Error())
	}
	return result.OkResult(string(payload))
}

// UploadMemoToCloud はメモをクラウドへ保存する。
func (app *App) UploadMemoToCloud(memoID string) result.ApiResult[bool] {
	ctx := app.context()
	client, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return result.ErrorResult[bool]("メモのアップロードに失敗しました", error.Error())
	}
	memoData, error := app.Database.GetMemoByID(ctx, strings.TrimSpace(memoID))
	if error != nil {
		return result.ErrorResult[bool]("メモの取得に失敗しました", error.Error())
	}
	if memoData == nil {
		return result.ErrorResult[bool]("メモが見つかりません", "指定されたIDが存在しません")
	}
	game, error := app.Database.GetGameByID(ctx, memoData.GameID)
	if error != nil {
		return result.ErrorResult[bool]("ゲームの取得に失敗しました", error.Error())
	}
	if game == nil {
		return result.ErrorResult[bool]("ゲームが見つかりません", "指定されたIDが存在しません")
	}

	key := memo.BuildMemoPath(game.Title, memoData.Title, memoData.ID)
	payload := memo.GenerateCloudMemoFileContent(memoData.Title, memoData.Content, game.Title)
	if error := storage.UploadBytes(ctx, client, app.Config.S3Bucket, key, []byte(payload), "text/markdown"); error != nil {
		return result.ErrorResult[bool]("メモのアップロードに失敗しました", error.Error())
	}
	return result.OkResult(true)
}

// SyncMemosFromCloud はメモをクラウドと同期する。
func (app *App) SyncMemosFromCloud(gameID string) result.ApiResult[MemoSyncResult] {
	ctx := app.context()
	client, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return result.ErrorResult[MemoSyncResult]("メモ同期に失敗しました", error.Error())
	}

	resultData := MemoSyncResult{
		Success: true,
		Details: []string{},
	}

	cloudMemosResult := app.GetCloudMemos()
	if !cloudMemosResult.Success {
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
		game, error := app.Database.GetGameByID(ctx, strings.TrimSpace(gameID))
		if error != nil {
			return result.ErrorResult[MemoSyncResult]("メモ同期に失敗しました", error.Error())
		}
		if game == nil {
			return result.ErrorResult[MemoSyncResult]("メモ同期に失敗しました", "指定されたゲームが見つかりません")
		}
		targetGame = game
	}

	cloudMap := map[string]CloudMemoInfo{}
	for _, cloudMemo := range cloudMemos {
		cloudMap[fmt.Sprintf("%s:%s", cloudMemo.GameTitle, cloudMemo.MemoID)] = cloudMemo
	}

	games, error := app.Database.ListGames(ctx, "", models.PlayStatus(""), "title", "asc")
	if error != nil {
		return result.ErrorResult[MemoSyncResult]("メモ同期に失敗しました", error.Error())
	}
	gameByID := map[string]models.Game{}
	gameBySanitizedTitle := map[string]models.Game{}
	for _, game := range games {
		gameByID[game.ID] = game
		sanitized := memo.SanitizeForCloudPath(game.Title)
		if _, exists := gameBySanitizedTitle[sanitized]; !exists {
			gameBySanitizedTitle[sanitized] = game
		}
	}

	localMemos, error := fetchLocalMemos(ctx, app.Database, gameID)
	if error != nil {
		return result.ErrorResult[MemoSyncResult]("メモ同期に失敗しました", error.Error())
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
		sanitizedGameTitle := memo.SanitizeForCloudPath(game.Title)
		key := fmt.Sprintf("%s:%s", sanitizedGameTitle, localMemo.ID)
		cloudMemo, exists := cloudMap[key]
		if !exists {
			if error := uploadMemoContent(ctx, client, app.Config.S3Bucket, game, localMemo); error != nil {
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
			if error := uploadMemoContent(ctx, client, app.Config.S3Bucket, game, localMemo); error != nil {
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

		payload, error := storage.DownloadObject(ctx, client, app.Config.S3Bucket, cloudMemo.Key)
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
		if error := uploadMemoContent(ctx, client, app.Config.S3Bucket, game, localMemo); error != nil {
			resultData.Details = append(resultData.Details, fmt.Sprintf("クラウド更新失敗: %s", localMemo.Title))
			continue
		}
		resultData.CloudOverwritten++
		processed[key] = true
	}

	targetSanitized := ""
	if targetGame != nil {
		targetSanitized = memo.SanitizeForCloudPath(targetGame.Title)
	}

	for _, cloudMemo := range cloudMemos {
		if targetSanitized != "" && cloudMemo.GameTitle != targetSanitized {
			continue
		}
		key := fmt.Sprintf("%s:%s", cloudMemo.GameTitle, cloudMemo.MemoID)
		if processed[key] {
			continue
		}
		game, ok := gameBySanitizedTitle[cloudMemo.GameTitle]
		if !ok {
			resultData.Skipped++
			resultData.Details = append(resultData.Details, fmt.Sprintf("ゲームが見つからないためスキップ: %s", cloudMemo.MemoTitle))
			continue
		}
		payload, error := storage.DownloadObject(ctx, client, app.Config.S3Bucket, cloudMemo.Key)
		if error != nil {
			resultData.Skipped++
			resultData.Details = append(resultData.Details, fmt.Sprintf("ダウンロード失敗: %s", cloudMemo.MemoTitle))
			continue
		}
		content := memo.ExtractMemoContent(string(payload))

		existingMemo, error := app.Database.GetMemoByID(ctx, cloudMemo.MemoID)
		if error != nil {
			resultData.Skipped++
			resultData.Details = append(resultData.Details, fmt.Sprintf("メモ取得失敗: %s", cloudMemo.MemoTitle))
			continue
		}
		if existingMemo == nil || existingMemo.GameID != game.ID {
			existingMemo, error = app.Database.FindMemoByTitle(ctx, game.ID, cloudMemo.MemoTitle)
			if error != nil {
				resultData.Skipped++
				resultData.Details = append(resultData.Details, fmt.Sprintf("メモ検索失敗: %s", cloudMemo.MemoTitle))
				continue
			}
		}

		if existingMemo == nil {
			created, error := app.Database.CreateMemo(ctx, models.Memo{
				Title:   cloudMemo.MemoTitle,
				Content: content,
				GameID:  game.ID,
			})
			if error != nil {
				resultData.Skipped++
				resultData.Details = append(resultData.Details, fmt.Sprintf("作成失敗: %s", cloudMemo.MemoTitle))
				continue
			}
			if app.MemoFiles != nil {
				_, _ = app.MemoFiles.CreateMemoFile(created.GameID, created.ID, created.Title, created.Content)
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
			existingMemo.Content = content
			updated, error := app.Database.UpdateMemo(ctx, *existingMemo)
			if error != nil {
				resultData.Skipped++
				resultData.Details = append(resultData.Details, fmt.Sprintf("更新失敗: %s", cloudMemo.MemoTitle))
				continue
			}
			if app.MemoFiles != nil {
				_, _ = app.MemoFiles.UpdateMemoFile(updated.GameID, updated.ID, updated.Title, updated.Title, updated.Content)
			}
			resultData.LocalOverwritten++
			processed[key] = true
			continue
		}

		if error := uploadMemoContent(ctx, client, app.Config.S3Bucket, game, *existingMemo); error != nil {
			resultData.Details = append(resultData.Details, fmt.Sprintf("クラウド更新失敗: %s", existingMemo.Title))
			continue
		}
		resultData.CloudOverwritten++
		processed[key] = true
	}

	return result.OkResult(resultData)
}

func uploadMemoContent(ctx context.Context, client *s3.Client, bucket string, game models.Game, memoData models.Memo) error {
	key := memo.BuildMemoPath(game.Title, memoData.Title, memoData.ID)
	payload := memo.GenerateCloudMemoFileContent(memoData.Title, memoData.Content, game.Title)
	return storage.UploadBytes(ctx, client, bucket, key, []byte(payload), "text/markdown")
}

func fetchLocalMemos(ctx context.Context, repository *db.Repository, gameID string) ([]models.Memo, error) {
	if strings.TrimSpace(gameID) == "" {
		return repository.ListAllMemos(ctx)
	}
	return repository.ListMemosByGame(ctx, strings.TrimSpace(gameID))
}
