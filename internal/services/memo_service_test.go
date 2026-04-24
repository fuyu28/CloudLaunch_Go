package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"CloudLaunch_Go/internal/models"
)

type fakeMemoRepository struct {
	createMemoFn    func(ctx context.Context, memo models.Memo) (*models.Memo, error)
	updateMemoFn    func(ctx context.Context, memo models.Memo) (*models.Memo, error)
	getMemoByIDFn   func(ctx context.Context, memoID string) (*models.Memo, error)
	findMemoByTitle func(ctx context.Context, gameID string, title string) (*models.Memo, error)
	listMemosByGame func(ctx context.Context, gameID string) ([]models.Memo, error)
	listAllMemosFn  func(ctx context.Context) ([]models.Memo, error)
	deleteMemoFn    func(ctx context.Context, memoID string) error
}

func (repository fakeMemoRepository) CreateMemo(ctx context.Context, memo models.Memo) (*models.Memo, error) {
	return repository.createMemoFn(ctx, memo)
}

func (repository fakeMemoRepository) UpdateMemo(ctx context.Context, memo models.Memo) (*models.Memo, error) {
	return repository.updateMemoFn(ctx, memo)
}

func (repository fakeMemoRepository) GetMemoByID(ctx context.Context, memoID string) (*models.Memo, error) {
	return repository.getMemoByIDFn(ctx, memoID)
}

func (repository fakeMemoRepository) FindMemoByTitle(ctx context.Context, gameID string, title string) (*models.Memo, error) {
	return repository.findMemoByTitle(ctx, gameID, title)
}

func (repository fakeMemoRepository) ListMemosByGame(ctx context.Context, gameID string) ([]models.Memo, error) {
	return repository.listMemosByGame(ctx, gameID)
}

func (repository fakeMemoRepository) ListAllMemos(ctx context.Context) ([]models.Memo, error) {
	return repository.listAllMemosFn(ctx)
}

func (repository fakeMemoRepository) DeleteMemo(ctx context.Context, memoID string) error {
	return repository.deleteMemoFn(ctx, memoID)
}

func TestMemoServiceGetMemoByIDUsesRepositoryBoundary(t *testing.T) {
	t.Parallel()

	service := NewMemoService(fakeMemoRepository{
		createMemoFn: func(ctx context.Context, memo models.Memo) (*models.Memo, error) {
			return &memo, nil
		},
		updateMemoFn: func(ctx context.Context, memo models.Memo) (*models.Memo, error) {
			return &memo, nil
		},
		getMemoByIDFn: func(ctx context.Context, memoID string) (*models.Memo, error) {
			return &models.Memo{ID: memoID, Title: "Memo", GameID: "game-1"}, nil
		},
		findMemoByTitle: func(ctx context.Context, gameID string, title string) (*models.Memo, error) {
			return nil, nil
		},
		listMemosByGame: func(ctx context.Context, gameID string) ([]models.Memo, error) {
			return nil, nil
		},
		listAllMemosFn: func(ctx context.Context) ([]models.Memo, error) {
			return nil, nil
		},
		deleteMemoFn: func(ctx context.Context, memoID string) error {
			return nil
		},
	}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.GetMemoByID(context.Background(), "memo-1")

	if !result.Success {
		t.Fatalf("expected success, got %#v", result.Error)
	}
	if result.Data == nil || result.Data.ID != "memo-1" {
		t.Fatalf("expected memo to be returned")
	}
}

func TestMemoServiceListAllMemosHandlesRepositoryError(t *testing.T) {
	t.Parallel()

	service := NewMemoService(fakeMemoRepository{
		createMemoFn: func(ctx context.Context, memo models.Memo) (*models.Memo, error) {
			return &memo, nil
		},
		updateMemoFn: func(ctx context.Context, memo models.Memo) (*models.Memo, error) {
			return &memo, nil
		},
		getMemoByIDFn: func(ctx context.Context, memoID string) (*models.Memo, error) {
			return nil, nil
		},
		findMemoByTitle: func(ctx context.Context, gameID string, title string) (*models.Memo, error) {
			return nil, nil
		},
		listMemosByGame: func(ctx context.Context, gameID string) ([]models.Memo, error) {
			return nil, nil
		},
		listAllMemosFn: func(ctx context.Context) ([]models.Memo, error) {
			return nil, errors.New("db down")
		},
		deleteMemoFn: func(ctx context.Context, memoID string) error {
			return nil
		},
	}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.ListAllMemos(context.Background())

	if result.Success {
		t.Fatalf("expected failure")
	}
	if result.Error == nil || result.Error.Message == "" {
		t.Fatalf("expected error details")
	}
}
