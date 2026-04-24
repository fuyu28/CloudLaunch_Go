package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"CloudLaunch_Go/internal/memo"
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

func TestMemoServiceFindMemoByTitleRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	service := NewMemoService(fakeMemoRepository{
		createMemoFn:    func(ctx context.Context, memo models.Memo) (*models.Memo, error) { return &memo, nil },
		updateMemoFn:    func(ctx context.Context, memo models.Memo) (*models.Memo, error) { return &memo, nil },
		getMemoByIDFn:   func(ctx context.Context, memoID string) (*models.Memo, error) { return nil, nil },
		findMemoByTitle: func(ctx context.Context, gameID string, title string) (*models.Memo, error) { return nil, nil },
		listMemosByGame: func(ctx context.Context, gameID string) ([]models.Memo, error) { return nil, nil },
		listAllMemosFn:  func(ctx context.Context) ([]models.Memo, error) { return nil, nil },
		deleteMemoFn:    func(ctx context.Context, memoID string) error { return nil },
	}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.FindMemoByTitle(context.Background(), "", "title")

	if result.Success {
		t.Fatalf("expected invalid input to fail")
	}
}

func TestMemoServiceCreateMemoRollsBackDatabaseWhenFileWriteFails(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "memos"), []byte("not a dir"), 0o600); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}
	manager := memo.NewFileManager(tempDir)
	repository := &trackingMemoRepository{
		createResult: &models.Memo{ID: "memo-1", Title: "Memo", Content: "Body", GameID: "game-1"},
	}
	service := NewMemoService(repository, manager, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.CreateMemo(context.Background(), MemoInput{
		Title:   "Memo",
		Content: "Body",
		GameID:  "game-1",
	})

	if result.Success {
		t.Fatalf("expected file write failure")
	}
	if repository.deleteMemoCalls != 1 {
		t.Fatalf("expected memo delete rollback to be called once")
	}
}

func TestMemoServiceUpdateMemoRollsBackDatabaseWhenFileUpdateFails(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "memos")
	if err := os.WriteFile(filePath, []byte("not a dir"), 0o600); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}
	manager := memo.NewFileManager(tempDir)
	repository := &trackingMemoRepository{
		getResult: &models.Memo{ID: "memo-1", Title: "Old", Content: "Old body", GameID: "game-1"},
		updateResults: []*models.Memo{
			{ID: "memo-1", Title: "New", Content: "New body", GameID: "game-1"},
			{ID: "memo-1", Title: "Old", Content: "Old body", GameID: "game-1"},
		},
	}
	service := NewMemoService(repository, manager, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.UpdateMemo(context.Background(), "memo-1", MemoUpdateInput{
		Title:   "New",
		Content: "New body",
	})

	if result.Success {
		t.Fatalf("expected file update failure")
	}
	if repository.updateMemoCalls != 2 {
		t.Fatalf("expected update rollback to call repository twice")
	}
}

type trackingMemoRepository struct {
	createResult    *models.Memo
	getResult       *models.Memo
	findResult      *models.Memo
	updateResults   []*models.Memo
	updateMemoCalls int
	deleteMemoCalls int
}

func (repository *trackingMemoRepository) CreateMemo(ctx context.Context, memo models.Memo) (*models.Memo, error) {
	return repository.createResult, nil
}
func (repository *trackingMemoRepository) UpdateMemo(ctx context.Context, memo models.Memo) (*models.Memo, error) {
	repository.updateMemoCalls++
	if len(repository.updateResults) == 0 {
		return &memo, nil
	}
	result := repository.updateResults[0]
	repository.updateResults = repository.updateResults[1:]
	return result, nil
}
func (repository *trackingMemoRepository) GetMemoByID(ctx context.Context, memoID string) (*models.Memo, error) {
	return repository.getResult, nil
}
func (repository *trackingMemoRepository) FindMemoByTitle(ctx context.Context, gameID string, title string) (*models.Memo, error) {
	return repository.findResult, nil
}
func (repository *trackingMemoRepository) ListMemosByGame(ctx context.Context, gameID string) ([]models.Memo, error) {
	return nil, nil
}
func (repository *trackingMemoRepository) ListAllMemos(ctx context.Context) ([]models.Memo, error) {
	return nil, nil
}
func (repository *trackingMemoRepository) DeleteMemo(ctx context.Context, memoID string) error {
	repository.deleteMemoCalls++
	return nil
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
