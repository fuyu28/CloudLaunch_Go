package app

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/credentials"
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/services"
)

type noopAppCloudSyncRepository struct{}

func (noopAppCloudSyncRepository) GetGameByID(ctx context.Context, gameID string) (*models.Game, error) {
	return nil, nil
}

func (noopAppCloudSyncRepository) ListGames(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
	return nil, nil
}

func (noopAppCloudSyncRepository) ListPlaySessionsByGame(ctx context.Context, gameID string) ([]models.PlaySession, error) {
	return nil, nil
}

func (noopAppCloudSyncRepository) UpsertGameSync(ctx context.Context, game models.Game) error {
	return nil
}

func (noopAppCloudSyncRepository) DeletePlaySessionsByGame(ctx context.Context, gameID string) error {
	return nil
}

func (noopAppCloudSyncRepository) UpsertPlaySessionSync(ctx context.Context, session models.PlaySession) error {
	return nil
}

func (noopAppCloudSyncRepository) SumPlaySessionDurationsByGame(ctx context.Context, gameID string) (int64, error) {
	return 0, nil
}

func (noopAppCloudSyncRepository) UpdateGameTotalPlayTime(ctx context.Context, gameID string, totalPlayTime int64) error {
	return nil
}

func (noopAppCloudSyncRepository) UpdateGameTotalPlayTimeWithLastPlayed(ctx context.Context, gameID string, totalPlayTime int64, playedAt time.Time) error {
	return nil
}

type noopAppGameRepository struct{}

func (noopAppGameRepository) ListGames(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
	return nil, nil
}

func (noopAppGameRepository) GetGameByID(ctx context.Context, gameID string) (*models.Game, error) {
	return nil, nil
}

func (noopAppGameRepository) CreateGame(ctx context.Context, game models.Game) (*models.Game, error) {
	return nil, nil
}

func (noopAppGameRepository) UpdateGame(ctx context.Context, game models.Game) (*models.Game, error) {
	return &game, nil
}

func (noopAppGameRepository) DeleteGame(ctx context.Context, gameID string) error {
	return nil
}

func (noopAppGameRepository) CreateChapter(ctx context.Context, chapter models.Chapter) (*models.Chapter, error) {
	return nil, nil
}

type noopAppMemoRepository struct{}

func (noopAppMemoRepository) CreateMemo(ctx context.Context, memo models.Memo) (*models.Memo, error) {
	return &memo, nil
}

func (noopAppMemoRepository) UpdateMemo(ctx context.Context, memo models.Memo) (*models.Memo, error) {
	return &memo, nil
}

func (noopAppMemoRepository) GetMemoByID(ctx context.Context, memoID string) (*models.Memo, error) {
	return nil, nil
}

func (noopAppMemoRepository) FindMemoByTitle(ctx context.Context, gameID string, title string) (*models.Memo, error) {
	return nil, nil
}

func (noopAppMemoRepository) ListMemosByGame(ctx context.Context, gameID string) ([]models.Memo, error) {
	return nil, nil
}

func (noopAppMemoRepository) ListAllMemos(ctx context.Context) ([]models.Memo, error) {
	return nil, nil
}

func (noopAppMemoRepository) DeleteMemo(ctx context.Context, memoID string) error {
	return nil
}

type adapterTestCredentialStore struct {
	loadResult *credentials.Credential
	loadErr    error
}

func (store *adapterTestCredentialStore) Save(ctx context.Context, key string, credential credentials.Credential) error {
	return nil
}

func (store *adapterTestCredentialStore) Load(ctx context.Context, key string) (*credentials.Credential, error) {
	return store.loadResult, store.loadErr
}

func (store *adapterTestCredentialStore) Delete(ctx context.Context, key string) error {
	return nil
}

func newAdapterTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestAppSyncAllGamesConvertsServiceError(t *testing.T) {
	t.Parallel()

	cloudSync := services.NewCloudSyncService(config.Config{}, nil, noopAppCloudSyncRepository{}, newAdapterTestLogger())
	cloudSync.SetOfflineMode(true)
	app := &App{
		Logger:           newAdapterTestLogger(),
		CloudSyncService: cloudSync,
	}

	result := app.SyncAllGames()

	if result.Success {
		t.Fatalf("expected sync failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "オフラインモードのため同期できません" {
		t.Fatalf("expected converted service error, got %#v", result.Error)
	}
}

func TestAppDeleteCloudGameConvertsServiceError(t *testing.T) {
	t.Parallel()

	cloudSync := services.NewCloudSyncService(config.Config{}, nil, noopAppCloudSyncRepository{}, newAdapterTestLogger())
	cloudSync.SetOfflineMode(true)
	app := &App{
		Logger:           newAdapterTestLogger(),
		CloudSyncService: cloudSync,
	}

	result := app.DeleteCloudGame("game-1")

	if result.Success {
		t.Fatalf("expected delete failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "オフラインモードのため削除できません" {
		t.Fatalf("expected converted service error, got %#v", result.Error)
	}
}

func TestAppGetCloudMemosConvertsServiceError(t *testing.T) {
	t.Parallel()

	app := &App{
		Logger: newAdapterTestLogger(),
		MemoCloudService: services.NewMemoCloudService(
			config.Config{},
			&adapterTestCredentialStore{},
			services.NewGameService(noopAppGameRepository{}, newAdapterTestLogger()),
			services.NewMemoService(noopAppMemoRepository{}, nil, newAdapterTestLogger()),
			newAdapterTestLogger(),
		),
	}

	result := app.GetCloudMemos()

	if result.Success {
		t.Fatalf("expected cloud memo failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "クラウドメモ取得に失敗しました" {
		t.Fatalf("expected converted service error, got %#v", result.Error)
	}
}

func TestAppUploadMemoToCloudConvertsNotFoundError(t *testing.T) {
	t.Parallel()

	store := &adapterTestCredentialStore{
		loadResult: &credentials.Credential{
			AccessKeyID:     "access",
			SecretAccessKey: "secret",
			BucketName:      "bucket",
			Region:          "region",
			Endpoint:        "endpoint",
		},
	}
	app := &App{
		Logger: newAdapterTestLogger(),
		MemoCloudService: services.NewMemoCloudService(
			config.Config{},
			store,
			services.NewGameService(noopAppGameRepository{}, newAdapterTestLogger()),
			services.NewMemoService(noopAppMemoRepository{}, nil, newAdapterTestLogger()),
			newAdapterTestLogger(),
		),
	}

	result := app.UploadMemoToCloud("missing-memo")

	if result.Success {
		t.Fatalf("expected upload failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "メモが見つかりません" {
		t.Fatalf("expected converted not found error, got %#v", result.Error)
	}
}
