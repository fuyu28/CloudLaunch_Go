// @fileoverview Wails バインディング用の API メソッドを提供する。
package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"CloudLaunch_Go/internal/credentials"
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/services"
	"CloudLaunch_Go/internal/storage"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ListGames はゲーム一覧を取得する。
func (app *App) ListGames(ctx context.Context, searchText string, filter string, sortBy string, sortDirection string) result.ApiResult[[]models.Game] {
	status := normalizePlayStatus(filter)
	return app.GameService.ListGames(ctx, searchText, status, sortBy, sortDirection)
}

// GetGameByID はゲームを取得する。
func (app *App) GetGameByID(ctx context.Context, gameID string) result.ApiResult[*models.Game] {
	return app.GameService.GetGameByID(ctx, gameID)
}

// CreateGame はゲームを作成する。
func (app *App) CreateGame(ctx context.Context, input services.GameInput) result.ApiResult[*models.Game] {
	return app.GameService.CreateGame(ctx, input)
}

// UpdateGame はゲームを更新する。
func (app *App) UpdateGame(ctx context.Context, gameID string, input services.GameUpdateInput) result.ApiResult[*models.Game] {
	return app.GameService.UpdateGame(ctx, gameID, input)
}

// UpdatePlayTime はプレイ時間を更新する。
func (app *App) UpdatePlayTime(ctx context.Context, gameID string, totalPlayTime int64, lastPlayed time.Time) result.ApiResult[*models.Game] {
	return app.GameService.UpdatePlayTime(ctx, gameID, totalPlayTime, lastPlayed)
}

// DeleteGame はゲームを削除する。
func (app *App) DeleteGame(ctx context.Context, gameID string) result.ApiResult[bool] {
	return app.GameService.DeleteGame(ctx, gameID)
}

// ListChaptersByGame は章一覧を取得する。
func (app *App) ListChaptersByGame(ctx context.Context, gameID string) result.ApiResult[[]models.Chapter] {
	return app.ChapterService.ListChaptersByGame(ctx, gameID)
}

// CreateChapter は章を作成する。
func (app *App) CreateChapter(ctx context.Context, input services.ChapterInput) result.ApiResult[*models.Chapter] {
	return app.ChapterService.CreateChapter(ctx, input)
}

// UpdateChapter は章を更新する。
func (app *App) UpdateChapter(ctx context.Context, chapterID string, input services.ChapterUpdateInput) result.ApiResult[*models.Chapter] {
	return app.ChapterService.UpdateChapter(ctx, chapterID, input)
}

// DeleteChapter は章を削除する。
func (app *App) DeleteChapter(ctx context.Context, chapterID string) result.ApiResult[bool] {
	return app.ChapterService.DeleteChapter(ctx, chapterID)
}

// CreateSession はセッションを作成する。
func (app *App) CreateSession(ctx context.Context, input services.SessionInput) result.ApiResult[*models.PlaySession] {
	return app.SessionService.CreateSession(ctx, input)
}

// ListSessionsByGame はセッション一覧を取得する。
func (app *App) ListSessionsByGame(ctx context.Context, gameID string) result.ApiResult[[]models.PlaySession] {
	return app.SessionService.ListSessionsByGame(ctx, gameID)
}

// DeleteSession はセッションを削除する。
func (app *App) DeleteSession(ctx context.Context, sessionID string) result.ApiResult[bool] {
	return app.SessionService.DeleteSession(ctx, sessionID)
}

// CreateMemo はメモを作成する。
func (app *App) CreateMemo(ctx context.Context, input services.MemoInput) result.ApiResult[*models.Memo] {
	return app.MemoService.CreateMemo(ctx, input)
}

// UpdateMemo はメモを更新する。
func (app *App) UpdateMemo(ctx context.Context, memoID string, input services.MemoUpdateInput) result.ApiResult[*models.Memo] {
	return app.MemoService.UpdateMemo(ctx, memoID, input)
}

// GetMemoByID はメモを取得する。
func (app *App) GetMemoByID(ctx context.Context, memoID string) result.ApiResult[*models.Memo] {
	return app.MemoService.GetMemoByID(ctx, memoID)
}

// ListAllMemos は全メモを取得する。
func (app *App) ListAllMemos(ctx context.Context) result.ApiResult[[]models.Memo] {
	return app.MemoService.ListAllMemos(ctx)
}

// ListMemosByGame はメモ一覧を取得する。
func (app *App) ListMemosByGame(ctx context.Context, gameID string) result.ApiResult[[]models.Memo] {
	return app.MemoService.ListMemosByGame(ctx, gameID)
}

// DeleteMemo はメモを削除する。
func (app *App) DeleteMemo(ctx context.Context, memoID string) result.ApiResult[bool] {
	return app.MemoService.DeleteMemo(ctx, memoID)
}

// FileFilterInput はファイル選択フィルタを表す。
type FileFilterInput struct {
	Name       string   `json:"name"`
	Extensions []string `json:"extensions"`
}

// SelectFile はファイル選択ダイアログを開く。
func (app *App) SelectFile(ctx context.Context, filters []FileFilterInput) result.ApiResult[string] {
	dialogContext := app.runtimeContext(ctx)
	fileFilters := buildFileFilters(filters)
	path, error := runtime.OpenFileDialog(dialogContext, runtime.OpenDialogOptions{
		Filters: fileFilters,
	})
	if error != nil {
		app.Logger.Error("ファイル選択に失敗", "error", error)
		return result.ErrorResult[string]("ファイル選択に失敗しました", error.Error())
	}
	if path == "" {
		return result.ErrorResult[string]("ファイルが選択されませんでした", "")
	}
	return result.OkResult(path)
}

// SelectFolder はフォルダ選択ダイアログを開く。
func (app *App) SelectFolder(ctx context.Context) result.ApiResult[string] {
	dialogContext := app.runtimeContext(ctx)
	path, error := runtime.OpenDirectoryDialog(dialogContext, runtime.OpenDialogOptions{})
	if error != nil {
		app.Logger.Error("フォルダ選択に失敗", "error", error)
		return result.ErrorResult[string]("フォルダ選択に失敗しました", error.Error())
	}
	if path == "" {
		return result.ErrorResult[string]("フォルダが選択されませんでした", "")
	}
	return result.OkResult(path)
}

// CheckFileExists はファイル存在を確認する。
func (app *App) CheckFileExists(ctx context.Context, filePath string) result.ApiResult[bool] {
	_, error := os.Stat(filePath)
	if error != nil {
		if os.IsNotExist(error) {
			return result.OkResult(false)
		}
		app.Logger.Error("ファイル存在チェックに失敗", "error", error)
		return result.ErrorResult[bool]("ファイル存在チェックに失敗しました", error.Error())
	}
	return result.OkResult(true)
}

// CheckDirectoryExists はディレクトリ存在を確認する。
func (app *App) CheckDirectoryExists(ctx context.Context, dirPath string) result.ApiResult[bool] {
	info, error := os.Stat(dirPath)
	if error != nil {
		if os.IsNotExist(error) {
			return result.OkResult(false)
		}
		app.Logger.Error("ディレクトリ存在チェックに失敗", "error", error)
		return result.ErrorResult[bool]("ディレクトリ存在チェックに失敗しました", error.Error())
	}
	return result.OkResult(info.IsDir())
}

// OpenFolder は指定パスをOSで開く。
func (app *App) OpenFolder(ctx context.Context, path string) result.ApiResult[bool] {
	if strings.TrimSpace(path) == "" {
		return result.ErrorResult[bool]("パスが不正です", "pathが空です")
	}
	runtime.BrowserOpenURL(app.runtimeContext(ctx), fileURLFromPath(path))
	return result.OkResult(true)
}

// OpenLogsDirectory はログ保存ディレクトリを開く。
func (app *App) OpenLogsDirectory(ctx context.Context) result.ApiResult[string] {
	path := app.Config.AppDataDir
	if path == "" {
		return result.ErrorResult[string]("ログディレクトリが不明です", "AppDataDirが空です")
	}
	runtime.BrowserOpenURL(app.runtimeContext(ctx), fileURLFromPath(path))
	return result.OkResult(path)
}

// CreateUpload はアップロード履歴を作成する。
func (app *App) CreateUpload(ctx context.Context, input services.UploadInput) result.ApiResult[*models.Upload] {
	return app.UploadService.CreateUpload(ctx, input)
}

// ListUploadsByGame はアップロード履歴を取得する。
func (app *App) ListUploadsByGame(ctx context.Context, gameID string) result.ApiResult[[]models.Upload] {
	return app.UploadService.ListUploadsByGame(ctx, gameID)
}

// SaveCredential は認証情報を保存する。
func (app *App) SaveCredential(ctx context.Context, key string, input services.CredentialInput) result.ApiResult[bool] {
	return app.CredentialService.SaveCredential(ctx, key, input)
}

// LoadCredential は認証情報を取得する。
func (app *App) LoadCredential(ctx context.Context, key string) result.ApiResult[*services.CredentialOutput] {
	return toCredentialOutput(app.CredentialService.LoadCredential(ctx, key))
}

// DeleteCredential は認証情報を削除する。
func (app *App) DeleteCredential(ctx context.Context, key string) result.ApiResult[bool] {
	return app.CredentialService.DeleteCredential(ctx, key)
}

// UploadFolder はフォルダをクラウドへアップロードする。
func (app *App) UploadFolder(ctx context.Context, credentialKey string, folderPath string, prefix string) result.ApiResult[storage.UploadSummary] {
	return app.CloudService.UploadFolder(ctx, credentialKey, folderPath, prefix)
}

// SaveCloudMetadata はメタ情報をクラウドに保存する。
func (app *App) SaveCloudMetadata(ctx context.Context, credentialKey string, metadata storage.CloudMetadata) result.ApiResult[bool] {
	return app.CloudService.SaveCloudMetadata(ctx, credentialKey, metadata)
}

// LoadCloudMetadata はメタ情報をクラウドから取得する。
func (app *App) LoadCloudMetadata(ctx context.Context, credentialKey string) result.ApiResult[*storage.CloudMetadata] {
	return app.CloudService.LoadCloudMetadata(ctx, credentialKey)
}

func (app *App) runtimeContext(ctx context.Context) context.Context {
	if app.ctx != nil {
		return app.ctx
	}
	return ctx
}

func buildFileFilters(filters []FileFilterInput) []runtime.FileFilter {
	if len(filters) == 0 {
		return nil
	}
	fileFilters := make([]runtime.FileFilter, 0, len(filters))
	for _, filter := range filters {
		if len(filter.Extensions) == 0 {
			continue
		}
		patterns := make([]string, 0, len(filter.Extensions))
		for _, ext := range filter.Extensions {
			trimmed := strings.TrimSpace(ext)
			if trimmed == "" {
				continue
			}
			trimmed = strings.TrimPrefix(trimmed, ".")
			patterns = append(patterns, "*."+trimmed)
		}
		if len(patterns) == 0 {
			continue
		}
		fileFilters = append(fileFilters, runtime.FileFilter{
			DisplayName: filter.Name,
			Pattern:     strings.Join(patterns, ";"),
		})
	}
	return fileFilters
}

func fileURLFromPath(path string) string {
	cleaned := filepath.ToSlash(path)
	if strings.HasPrefix(cleaned, "/") {
		return "file://" + cleaned
	}
	return "file:///" + cleaned
}

// normalizePlayStatus はUIのフィルタ文字列をモデル値へ変換する。
func normalizePlayStatus(filter string) models.PlayStatus {
	value := strings.ToLower(strings.TrimSpace(filter))
	switch value {
	case "unplayed":
		return models.PlayStatusUnplayed
	case "playing":
		return models.PlayStatusPlaying
	case "played":
		return models.PlayStatusPlayed
	default:
		return ""
	}
}

// toCredentialOutput は内部認証情報をUI向けに変換する。
func toCredentialOutput(resultData result.ApiResult[*credentials.Credential]) result.ApiResult[*services.CredentialOutput] {
	if !resultData.Success {
		return result.ApiResult[*services.CredentialOutput]{
			Success: false,
			Error:   resultData.Error,
		}
	}
	if resultData.Data == nil {
		return result.OkResult[*services.CredentialOutput](nil)
	}
	return result.OkResult(&services.CredentialOutput{
		AccessKeyID: resultData.Data.AccessKeyID,
	})
}
