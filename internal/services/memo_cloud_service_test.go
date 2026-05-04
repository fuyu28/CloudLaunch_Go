package services

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/credentials"
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/storage"
)

type fakeMemoCloudGameRepository struct {
	games []models.Game
	game  *models.Game
}

func (repository fakeMemoCloudGameRepository) ListGames(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
	return repository.games, nil
}

func (repository fakeMemoCloudGameRepository) GetGameByID(ctx context.Context, gameID string) (*models.Game, error) {
	return repository.game, nil
}

func (repository fakeMemoCloudGameRepository) CreateGame(ctx context.Context, game models.Game) (*models.Game, error) {
	return nil, nil
}

func (repository fakeMemoCloudGameRepository) UpdateGame(ctx context.Context, game models.Game) (*models.Game, error) {
	return &game, nil
}

func (repository fakeMemoCloudGameRepository) DeleteGame(ctx context.Context, gameID string) error {
	return nil
}

type fakeMemoCloudMemoRepository struct {
	memo       *models.Memo
	memoByGame []models.Memo
}

func (repository fakeMemoCloudMemoRepository) CreateMemo(ctx context.Context, memo models.Memo) (*models.Memo, error) {
	return &memo, nil
}

func (repository fakeMemoCloudMemoRepository) UpdateMemo(ctx context.Context, memo models.Memo) (*models.Memo, error) {
	return &memo, nil
}

func (repository fakeMemoCloudMemoRepository) GetMemoByID(ctx context.Context, memoID string) (*models.Memo, error) {
	return repository.memo, nil
}

func (repository fakeMemoCloudMemoRepository) FindMemoByTitle(ctx context.Context, gameID string, title string) (*models.Memo, error) {
	return nil, nil
}

func (repository fakeMemoCloudMemoRepository) ListMemosByGame(ctx context.Context, gameID string) ([]models.Memo, error) {
	return repository.memoByGame, nil
}

func (repository fakeMemoCloudMemoRepository) ListAllMemos(ctx context.Context) ([]models.Memo, error) {
	return repository.memoByGame, nil
}

func (repository fakeMemoCloudMemoRepository) DeleteMemo(ctx context.Context, memoID string) error {
	return nil
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

	game := &models.Game{ID: "game-1", Title: "Game"}
	memoData := &models.Memo{
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
	if objectStore.uploadedKey == "" {
		t.Fatal("expected upload key to be recorded")
	}
}
