package services

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/infrastructure/credentials"
	"CloudLaunch_Go/internal/infrastructure/storage"
)

type fakeMemoCloudGameRepository struct {
	games []domain.Game
	game  *domain.Game
}

func (repository fakeMemoCloudGameRepository) ListGames(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error) {
	return repository.games, nil
}

func (repository fakeMemoCloudGameRepository) GetGameByID(ctx context.Context, gameID string) (*domain.Game, error) {
	return repository.game, nil
}

func (repository fakeMemoCloudGameRepository) CreateGameWithInitialRoute(ctx context.Context, game domain.Game, initialRoute domain.Route) (*domain.Game, error) {
	return nil, nil
}

func (repository fakeMemoCloudGameRepository) UpdateGame(ctx context.Context, game domain.Game) (*domain.Game, error) {
	return &game, nil
}

func (repository fakeMemoCloudGameRepository) DeleteGameAndQueueMemoCleanup(ctx context.Context, gameID string) error {
	return nil
}

func (repository fakeMemoCloudGameRepository) ListPendingMemoCleanup(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (repository fakeMemoCloudGameRepository) ClearPendingMemoCleanup(ctx context.Context, gameID string) error {
	return nil
}

type fakeMemoCloudMemoRepository struct {
	memo       *domain.Memo
	memoByGame []domain.Memo
}

func (repository fakeMemoCloudMemoRepository) CreateMemo(ctx context.Context, memo domain.Memo) (*domain.Memo, error) {
	return &memo, nil
}

func (repository fakeMemoCloudMemoRepository) UpdateMemo(ctx context.Context, memo domain.Memo) (*domain.Memo, error) {
	return &memo, nil
}

func (repository fakeMemoCloudMemoRepository) GetMemoByID(ctx context.Context, memoID string) (*domain.Memo, error) {
	return repository.memo, nil
}

func (repository fakeMemoCloudMemoRepository) FindMemoByTitle(ctx context.Context, gameID string, title string) (*domain.Memo, error) {
	return nil, nil
}

func (repository fakeMemoCloudMemoRepository) ListMemosByGame(ctx context.Context, gameID string) ([]domain.Memo, error) {
	return repository.memoByGame, nil
}

func (repository fakeMemoCloudMemoRepository) ListAllMemos(ctx context.Context) ([]domain.Memo, error) {
	return repository.memoByGame, nil
}

func (repository fakeMemoCloudMemoRepository) DeleteMemo(ctx context.Context, memoID string) error {
	return nil
}

type fakeCloudObjectStore struct {
	listObjects    []storage.ObjectInfo
	uploadedKeys   []string
	downloadedKeys []string
	downloadData   []byte
}

func (f *fakeCloudObjectStore) ListObjects(_ context.Context, _ storage.S3Config, _ credentials.Credential, _ string) ([]storage.ObjectInfo, error) {
	return f.listObjects, nil
}

func (f *fakeCloudObjectStore) UploadBytes(_ context.Context, _ storage.S3Config, _ credentials.Credential, key string, _ []byte, _ string) error {
	f.uploadedKeys = append(f.uploadedKeys, key)
	return nil
}

func (f *fakeCloudObjectStore) DownloadObject(_ context.Context, _ storage.S3Config, _ credentials.Credential, key string) ([]byte, error) {
	f.downloadedKeys = append(f.downloadedKeys, key)
	if f.downloadData != nil {
		return f.downloadData, nil
	}
	return []byte("content"), nil
}

func TestMemoCloudServiceGetCloudMemosUsesObjectStorePort(t *testing.T) {
	t.Parallel()

	service := NewMemoCloudService(
		config.Config{},
		&fakeCredentialStore{loadResult: &credentials.Credential{
			AccessKeyID:     "access",
			SecretAccessKey: "secret",
			BucketName:      "bucket",
			Region:          "region",
			Endpoint:        "endpoint",
		}},
		NewGameService(fakeMemoCloudGameRepository{}, slog.New(slog.NewTextHandler(io.Discard, nil))),
		NewMemoService(fakeMemoCloudMemoRepository{}, nil, slog.New(slog.NewTextHandler(io.Discard, nil))),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	service.objectStore = &fakeCloudObjectStore{
		listObjects: []storage.ObjectInfo{
			{Key: "games/game-1/memo/Intro__memo-1.md", LastModified: time.Now().UnixMilli(), Size: 123},
			{Key: "games/game-1/image.png", LastModified: time.Now().UnixMilli(), Size: 456},
		},
	}

	result, err := service.GetCloudMemos(context.Background())
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 memo object, got %d", len(result))
	}
	if result[0].GameID != "game-1" || result[0].MemoID != "memo-1" {
		t.Fatalf("unexpected memo info: %#v", result[0])
	}
}

func TestMemoCloudServiceUploadMemoToCloudUsesObjectStorePort(t *testing.T) {
	t.Parallel()

	game := &domain.Game{ID: "game-1", Title: "Game"}
	memoData := &domain.Memo{
		ID:      "memo-1",
		GameID:  "game-1",
		Title:   "Memo",
		Content: "Body",
	}
	objectStore := &fakeCloudObjectStore{}
	service := NewMemoCloudService(
		config.Config{},
		&fakeCredentialStore{loadResult: &credentials.Credential{
			AccessKeyID:     "access",
			SecretAccessKey: "secret",
			BucketName:      "bucket",
			Region:          "region",
			Endpoint:        "endpoint",
		}},
		NewGameService(fakeMemoCloudGameRepository{game: game}, slog.New(slog.NewTextHandler(io.Discard, nil))),
		NewMemoService(fakeMemoCloudMemoRepository{memo: memoData}, nil, slog.New(slog.NewTextHandler(io.Discard, nil))),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	service.objectStore = objectStore

	err := service.UploadMemoToCloud(context.Background(), "memo-1")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(objectStore.uploadedKeys) == 0 {
		t.Fatal("expected upload key to be recorded")
	}
}
