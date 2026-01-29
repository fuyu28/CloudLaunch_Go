// @fileoverview Wails バインディング用の API メソッドを提供する。
package app

import (
	"context"
	"os"
	"os/exec"
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
func (app *App) ListGames(searchText string, filter string, sortBy string, sortDirection string) result.ApiResult[[]models.Game] {
	ctx := app.context()
	status := normalizePlayStatus(filter)
	return app.GameService.ListGames(ctx, searchText, status, sortBy, sortDirection)
}

// GetGameByID はゲームを取得する。
func (app *App) GetGameByID(gameID string) result.ApiResult[*models.Game] {
	return app.GameService.GetGameByID(app.context(), gameID)
}

// CreateGame はゲームを作成する。
func (app *App) CreateGame(input services.GameInput) result.ApiResult[*models.Game] {
	return app.GameService.CreateGame(app.context(), input)
}

// UpdateGame はゲームを更新する。
func (app *App) UpdateGame(gameID string, input services.GameUpdateInput) result.ApiResult[*models.Game] {
	return app.GameService.UpdateGame(app.context(), gameID, input)
}

// UpdatePlayTime はプレイ時間を更新する。
func (app *App) UpdatePlayTime(gameID string, totalPlayTime int64, lastPlayed time.Time) result.ApiResult[*models.Game] {
	return app.GameService.UpdatePlayTime(app.context(), gameID, totalPlayTime, lastPlayed)
}

// DeleteGame はゲームを削除する。
func (app *App) DeleteGame(gameID string) result.ApiResult[bool] {
	return app.GameService.DeleteGame(app.context(), gameID)
}

// ListChaptersByGame は章一覧を取得する。
func (app *App) ListChaptersByGame(gameID string) result.ApiResult[[]models.Chapter] {
	return app.ChapterService.ListChaptersByGame(app.context(), gameID)
}

// CreateChapter は章を作成する。
func (app *App) CreateChapter(input services.ChapterInput) result.ApiResult[*models.Chapter] {
	return app.ChapterService.CreateChapter(app.context(), input)
}

// UpdateChapter は章を更新する。
func (app *App) UpdateChapter(chapterID string, input services.ChapterUpdateInput) result.ApiResult[*models.Chapter] {
	return app.ChapterService.UpdateChapter(app.context(), chapterID, input)
}

// UpdateChapterOrders は章の並び順を更新する。
func (app *App) UpdateChapterOrders(gameID string, orders []services.ChapterOrderUpdate) result.ApiResult[bool] {
	return app.ChapterService.UpdateChapterOrders(app.context(), gameID, orders)
}

// GetChapterStats は章の統計を取得する。
func (app *App) GetChapterStats(gameID string) result.ApiResult[[]models.ChapterStat] {
	return app.ChapterService.GetChapterStats(app.context(), gameID)
}

// SetCurrentChapter はゲームの現在章を設定する。
func (app *App) SetCurrentChapter(gameID string, chapterID string) result.ApiResult[bool] {
	return app.ChapterService.SetCurrentChapter(app.context(), gameID, chapterID)
}

// DeleteChapter は章を削除する。
func (app *App) DeleteChapter(chapterID string) result.ApiResult[bool] {
	return app.ChapterService.DeleteChapter(app.context(), chapterID)
}

// CreateSession はセッションを作成する。
func (app *App) CreateSession(input services.SessionInput) result.ApiResult[*models.PlaySession] {
	return app.SessionService.CreateSession(app.context(), input)
}

// ListSessionsByGame はセッション一覧を取得する。
func (app *App) ListSessionsByGame(gameID string) result.ApiResult[[]models.PlaySession] {
	return app.SessionService.ListSessionsByGame(app.context(), gameID)
}

// DeleteSession はセッションを削除する。
func (app *App) DeleteSession(sessionID string) result.ApiResult[bool] {
	return app.SessionService.DeleteSession(app.context(), sessionID)
}

// UpdateSessionChapter はセッション章を更新する。
func (app *App) UpdateSessionChapter(sessionID string, chapterID *string) result.ApiResult[bool] {
	return app.SessionService.UpdateSessionChapter(app.context(), sessionID, chapterID)
}

// UpdateSessionName はセッション名を更新する。
func (app *App) UpdateSessionName(sessionID string, sessionName string) result.ApiResult[bool] {
	return app.SessionService.UpdateSessionName(app.context(), sessionID, sessionName)
}

// CreateMemo はメモを作成する。
func (app *App) CreateMemo(input services.MemoInput) result.ApiResult[*models.Memo] {
	return app.MemoService.CreateMemo(app.context(), input)
}

// UpdateMemo はメモを更新する。
func (app *App) UpdateMemo(memoID string, input services.MemoUpdateInput) result.ApiResult[*models.Memo] {
	return app.MemoService.UpdateMemo(app.context(), memoID, input)
}

// GetMemoByID はメモを取得する。
func (app *App) GetMemoByID(memoID string) result.ApiResult[*models.Memo] {
	return app.MemoService.GetMemoByID(app.context(), memoID)
}

// ListAllMemos は全メモを取得する。
func (app *App) ListAllMemos() result.ApiResult[[]models.Memo] {
	return app.MemoService.ListAllMemos(app.context())
}

// ListMemosByGame はメモ一覧を取得する。
func (app *App) ListMemosByGame(gameID string) result.ApiResult[[]models.Memo] {
	return app.MemoService.ListMemosByGame(app.context(), gameID)
}

// DeleteMemo はメモを削除する。
func (app *App) DeleteMemo(memoID string) result.ApiResult[bool] {
	return app.MemoService.DeleteMemo(app.context(), memoID)
}

// FileFilterInput はファイル選択フィルタを表す。
type FileFilterInput struct {
	Name       string   `json:"name"`
	Extensions []string `json:"extensions"`
}

// UpdateAutoTracking は自動計測設定を更新する。
func (app *App) UpdateAutoTracking(enabled bool) result.ApiResult[bool] {
	app.autoTracking = enabled
	if app.ProcessMonitor != nil {
		app.ProcessMonitor.UpdateAutoTracking(enabled)
		app.isMonitoring = app.ProcessMonitor.IsMonitoring()
	}
	return result.OkResult(true)
}

// GetMonitoringStatus は監視状態を取得する。
func (app *App) GetMonitoringStatus() result.ApiResult[[]models.MonitoringGameStatus] {
	if app.ProcessMonitor == nil {
		return result.OkResult([]models.MonitoringGameStatus{})
	}
	status := app.ProcessMonitor.GetMonitoringStatus()
	return result.OkResult(status)
}

// GetProcessSnapshot はプロセス一覧のデバッグ情報を取得する。
func (app *App) GetProcessSnapshot() result.ApiResult[models.ProcessSnapshot] {
	if app.ProcessMonitor == nil {
		return result.OkResult(models.ProcessSnapshot{Source: "none", Items: []models.ProcessSnapshotItem{}})
	}
	snapshot := app.ProcessMonitor.GetProcessSnapshot()
	return result.OkResult(snapshot)
}

// SelectFile はファイル選択ダイアログを開く。
func (app *App) SelectFile(filters []FileFilterInput) result.ApiResult[string] {
	dialogContext := app.runtimeContext()
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
func (app *App) SelectFolder() result.ApiResult[string] {
	dialogContext := app.runtimeContext()
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
func (app *App) CheckFileExists(filePath string) result.ApiResult[bool] {
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
func (app *App) CheckDirectoryExists(dirPath string) result.ApiResult[bool] {
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
func (app *App) OpenFolder(path string) result.ApiResult[bool] {
	if strings.TrimSpace(path) == "" {
		return result.ErrorResult[bool]("パスが不正です", "pathが空です")
	}
	runtime.BrowserOpenURL(app.runtimeContext(), fileURLFromPath(path))
	return result.OkResult(true)
}

// OpenLogsDirectory はログ保存ディレクトリを開く。
func (app *App) OpenLogsDirectory() result.ApiResult[string] {
	path := app.Config.AppDataDir
	if path == "" {
		return result.ErrorResult[string]("ログディレクトリが不明です", "AppDataDirが空です")
	}
	runtime.BrowserOpenURL(app.runtimeContext(), fileURLFromPath(path))
	return result.OkResult(path)
}

// CreateUpload はアップロード履歴を作成する。
func (app *App) CreateUpload(input services.UploadInput) result.ApiResult[*models.Upload] {
	return app.UploadService.CreateUpload(app.context(), input)
}

// ListUploadsByGame はアップロード履歴を取得する。
func (app *App) ListUploadsByGame(gameID string) result.ApiResult[[]models.Upload] {
	return app.UploadService.ListUploadsByGame(app.context(), gameID)
}

// SaveCredential は認証情報を保存する。
func (app *App) SaveCredential(key string, input services.CredentialInput) result.ApiResult[bool] {
	return app.CredentialService.SaveCredential(app.context(), key, input)
}

// LoadCredential は認証情報を取得する。
func (app *App) LoadCredential(key string) result.ApiResult[*services.CredentialOutput] {
	return toCredentialOutput(app.CredentialService.LoadCredential(app.context(), key))
}

// DeleteCredential は認証情報を削除する。
func (app *App) DeleteCredential(key string) result.ApiResult[bool] {
	return app.CredentialService.DeleteCredential(app.context(), key)
}

// UploadFolder はフォルダをクラウドへアップロードする。
func (app *App) UploadFolder(credentialKey string, folderPath string, prefix string) result.ApiResult[storage.UploadSummary] {
	return app.CloudService.UploadFolder(app.context(), credentialKey, folderPath, prefix)
}

// SaveCloudMetadata はメタ情報をクラウドに保存する。
func (app *App) SaveCloudMetadata(credentialKey string, metadata storage.CloudMetadata) result.ApiResult[bool] {
	return app.CloudService.SaveCloudMetadata(app.context(), credentialKey, metadata)
}

// LoadCloudMetadata はメタ情報をクラウドから取得する。
func (app *App) LoadCloudMetadata(credentialKey string) result.ApiResult[*storage.CloudMetadata] {
	return app.CloudService.LoadCloudMetadata(app.context(), credentialKey)
}

// LaunchGame は指定された実行ファイルを起動する。
func (app *App) LaunchGame(exePath string) result.ApiResult[bool] {
	if strings.TrimSpace(exePath) == "" {
		return result.ErrorResult[bool]("実行ファイルが不正です", "exePathが空です")
	}
	command := exec.Command(exePath)
	command.Dir = filepath.Dir(exePath)
	if error := command.Start(); error != nil {
		app.Logger.Error("ゲーム起動に失敗", "error", error)
		return result.ErrorResult[bool]("ゲーム起動に失敗しました", error.Error())
	}
	return result.OkResult(true)
}

func (app *App) runtimeContext() context.Context {
	return app.context()
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
		BucketName:  resultData.Data.BucketName,
		Region:      resultData.Data.Region,
		Endpoint:    resultData.Data.Endpoint,
	})
}
