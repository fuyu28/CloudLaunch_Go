// ゲーム管理のビジネスロジックを提供する。
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"CloudLaunch_Go/internal/domain"
)

// GameService はゲーム関連の操作を提供する。
type GameService struct {
	repository GameRepository
	memoFiles  MemoDirectoryCleaner
	logger     *slog.Logger
}

// MemoDirectoryCleaner はゲーム単位のローカルメモ削除境界を定義する。
type MemoDirectoryCleaner interface {
	DeleteGameMemoFiles(gameID string) error
}

// NewGameService は GameService を生成する。
func NewGameService(repository GameRepository, logger *slog.Logger, memoFiles ...MemoDirectoryCleaner) *GameService {
	var cleaner MemoDirectoryCleaner
	if len(memoFiles) > 0 {
		cleaner = memoFiles[0]
	}
	return &GameService{repository: repository, memoFiles: cleaner, logger: logger}
}

// ListGames は検索・フィルタ・ソート付きでゲーム一覧を取得する。
func (service *GameService) ListGames(
	ctx context.Context,
	searchText string,
	filter domain.PlayStatus,
	sortBy string,
	sortDirection string,
) ([]domain.Game, error) {
	games, error := service.repository.ListGames(ctx, strings.TrimSpace(searchText), filter, sortBy, sortDirection)
	if error != nil {
		service.logger.Error("ゲーム一覧取得に失敗", "error", error)
		return nil, newServiceError("ゲーム一覧取得に失敗しました", error.Error())
	}
	return games, nil
}

// GetGameByID はID指定でゲームを取得する。
func (service *GameService) GetGameByID(ctx context.Context, gameID string) (*domain.Game, error) {
	game, error := service.repository.GetGameByID(ctx, strings.TrimSpace(gameID))
	if error != nil {
		service.logger.Error("ゲーム取得に失敗", "error", error)
		return nil, newServiceError("ゲーム取得に失敗しました", error.Error())
	}
	return game, nil
}

// CreateGame はゲームを新規作成する。
func (service *GameService) CreateGame(ctx context.Context, input GameInput) (*domain.Game, error) {
	if error := validateGameInput(input); error != nil {
		service.logger.Warn("ゲーム入力が不正です", "error", error)
		return nil, newServiceError("ゲーム入力が不正です", error.Error())
	}

	game := domain.Game{
		Title:          strings.TrimSpace(input.Title),
		Publisher:      strings.TrimSpace(input.Publisher),
		ImagePath:      input.ImagePath,
		ExePath:        strings.TrimSpace(input.ExePath),
		SaveFolderPath: input.SaveFolderPath,
		PlayStatus:     domain.PlayStatusUnplayed,
		TotalPlayTime:  0,
	}

	created, error := service.repository.CreateGameWithInitialRoute(ctx, game, domain.Route{
		Name:  "メインルート",
		Order: 1,
	})
	if error != nil {
		service.logger.Error("ゲーム作成に失敗", "error", error)
		return nil, newServiceError("ゲーム作成に失敗しました", error.Error())
	}

	service.logger.Info("ゲームを作成", "title", game.Title)
	return created, nil
}

// UpdateGame はゲーム情報を更新する。
func (service *GameService) UpdateGame(ctx context.Context, gameID string, input GameUpdateInput) (*domain.Game, error) {
	trimmedID, detail, ok := requireNonEmpty(gameID, "gameID")
	if !ok {
		service.logger.Warn("ゲームIDが不正です", "detail", detail, "gameId", gameID)
		return nil, newServiceError("ゲームIDが不正です", detail)
	}

	current, error := service.repository.GetGameByID(ctx, trimmedID)
	if error != nil {
		service.logger.Error("ゲーム取得に失敗", "error", error)
		return nil, newServiceError("ゲーム取得に失敗しました", error.Error())
	}
	if current == nil {
		service.logger.Warn("ゲームが見つかりません", "gameId", trimmedID)
		return nil, newServiceError("ゲームが見つかりません", "指定されたIDが存在しません")
	}

	if input.PlayStatus != "" && !domain.IsValidPlayStatus(input.PlayStatus) {
		service.logger.Warn("playStatus が不正です", "playStatus", input.PlayStatus)
		return nil, newServiceError("playStatus が不正です", string(input.PlayStatus))
	}

	current.Title = strings.TrimSpace(input.Title)
	current.Publisher = strings.TrimSpace(input.Publisher)
	current.ImagePath = input.ImagePath
	current.ExePath = strings.TrimSpace(input.ExePath)
	current.SaveFolderPath = input.SaveFolderPath

	// PlayStatus, ClearedAt, CurrentRouteID は「未指定なら現状維持」の規約。
	// フロントの updateGame() は通常編集時にこれら3つを undefined で送ってくるため、
	// 無条件代入だとタイトル編集だけで clearedAt と currentRouteId が消える。
	if input.PlayStatus != "" {
		current.PlayStatus = input.PlayStatus
	}
	if input.ClearedAt != nil {
		current.ClearedAt = input.ClearedAt
		current.PlayStatus = domain.PlayStatusPlayed
	} else if input.PlayStatus != "" && input.PlayStatus != domain.PlayStatusPlayed {
		// playStatus を played 以外に変更したら ClearedAt は整合性のためクリアする。
		current.ClearedAt = nil
	}
	if input.CurrentRouteID != nil {
		current.CurrentRouteID = input.CurrentRouteID
	}

	updated, error := service.repository.UpdateGame(ctx, *current)
	if error != nil {
		service.logger.Error("ゲーム更新に失敗", "error", error)
		return nil, newServiceError("ゲーム更新に失敗しました", error.Error())
	}
	return updated, nil
}

// UpdatePlayTime はプレイ時間と最終プレイ日時を更新する。
func (service *GameService) UpdatePlayTime(ctx context.Context, gameID string, totalPlayTime int64, lastPlayed time.Time) (*domain.Game, error) {
	trimmedID, detail, ok := requireNonEmpty(gameID, "gameID")
	if !ok {
		service.logger.Warn("ゲームIDが不正です", "detail", detail, "gameId", gameID)
		return nil, newServiceError("ゲームIDが不正です", detail)
	}

	current, error := service.repository.GetGameByID(ctx, trimmedID)
	if error != nil {
		service.logger.Error("ゲーム取得に失敗", "error", error)
		return nil, newServiceError("ゲーム取得に失敗しました", error.Error())
	}
	if current == nil {
		service.logger.Warn("ゲームが見つかりません", "gameId", trimmedID)
		return nil, newServiceError("ゲームが見つかりません", "指定されたIDが存在しません")
	}

	current.TotalPlayTime = totalPlayTime
	current.LastPlayed = &lastPlayed

	updated, error := service.repository.UpdateGame(ctx, *current)
	if error != nil {
		service.logger.Error("プレイ時間更新に失敗", "error", error)
		return nil, newServiceError("プレイ時間更新に失敗しました", error.Error())
	}
	return updated, nil
}

// DeleteGame はゲームを削除する。
func (service *GameService) DeleteGame(ctx context.Context, gameID string) error {
	trimmedID, detail, ok := requireNonEmpty(gameID, "gameID")
	if !ok {
		service.logger.Warn("ゲームIDが不正です", "detail", detail, "gameId", gameID)
		return newServiceError("ゲームIDが不正です", detail)
	}

	if error := service.repository.DeleteGameAndQueueMemoCleanup(ctx, trimmedID); error != nil {
		service.logger.Error("ゲーム削除に失敗", "error", error)
		return newServiceError("ゲーム削除に失敗しました", error.Error())
	}
	if error := service.cleanupMemoDirectory(ctx, trimmedID); error != nil {
		return newServiceError("ゲームのローカルメモ削除に失敗しました", error.Error())
	}
	return nil
}

// RetryPendingMemoCleanup は保留中のローカルメモ削除を再実行する。
func (service *GameService) RetryPendingMemoCleanup(ctx context.Context) error {
	gameIDs, err := service.repository.ListPendingMemoCleanup(ctx)
	if err != nil {
		service.logger.Error("メモ削除保留一覧の取得に失敗", "error", err)
		return err
	}

	var cleanupErrors []error
	for _, gameID := range gameIDs {
		if err := service.cleanupMemoDirectory(ctx, gameID); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("%s: %w", gameID, err))
		}
	}
	return errors.Join(cleanupErrors...)
}

func (service *GameService) cleanupMemoDirectory(ctx context.Context, gameID string) error {
	if service.memoFiles == nil {
		return errors.New("memo directory cleaner is not configured")
	}
	if err := service.memoFiles.DeleteGameMemoFiles(gameID); err != nil {
		service.logger.Warn("ゲームのローカルメモ削除に失敗", "gameId", gameID, "error", err)
		return err
	}
	if err := service.repository.ClearPendingMemoCleanup(ctx, gameID); err != nil {
		service.logger.Warn("メモ削除保留の解除に失敗", "gameId", gameID, "error", err)
		return err
	}
	return nil
}

// GameInput はゲーム作成入力を表す。
type GameInput struct {
	Title          string
	Publisher      string
	ImagePath      *string
	ExePath        string
	SaveFolderPath *string
}

// GameUpdateInput はゲーム更新入力を表す。
type GameUpdateInput struct {
	Title          string
	Publisher      string
	ImagePath      *string
	ExePath        string
	SaveFolderPath *string
	PlayStatus     domain.PlayStatus
	ClearedAt      *time.Time
	CurrentRouteID *string
}

// validateGameInput はゲーム作成入力の簡易検証を行う。
func validateGameInput(input GameInput) error {
	if _, detail, ok := requireNonEmpty(input.Title, "title"); !ok {
		return errors.New(detail)
	}
	if _, detail, ok := requireNonEmpty(input.Publisher, "publisher"); !ok {
		return errors.New(detail)
	}
	if _, detail, ok := requireNonEmpty(input.ExePath, "exePath"); !ok {
		return errors.New(detail)
	}
	return nil
}
