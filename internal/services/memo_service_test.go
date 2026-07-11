package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/memo"
)

type fakeMemoRepository struct {
	createMemoFn    func(ctx context.Context, memo domain.Memo) (*domain.Memo, error)
	updateMemoFn    func(ctx context.Context, memo domain.Memo) (*domain.Memo, error)
	getMemoByIDFn   func(ctx context.Context, memoID string) (*domain.Memo, error)
	findMemoByTitle func(ctx context.Context, gameID string, title string) (*domain.Memo, error)
	listMemosByGame func(ctx context.Context, gameID string) ([]domain.Memo, error)
	listAllMemosFn  func(ctx context.Context) ([]domain.Memo, error)
	deleteMemoFn    func(ctx context.Context, memoID string) error
}

func (repository fakeMemoRepository) CreateMemo(ctx context.Context, memo domain.Memo) (*domain.Memo, error) {
	return repository.createMemoFn(ctx, memo)
}

func (repository fakeMemoRepository) UpdateMemo(ctx context.Context, memo domain.Memo) (*domain.Memo, error) {
	return repository.updateMemoFn(ctx, memo)
}

func (repository fakeMemoRepository) GetMemoByID(ctx context.Context, memoID string) (*domain.Memo, error) {
	return repository.getMemoByIDFn(ctx, memoID)
}

func (repository fakeMemoRepository) FindMemoByTitle(ctx context.Context, gameID string, title string) (*domain.Memo, error) {
	return repository.findMemoByTitle(ctx, gameID, title)
}

func (repository fakeMemoRepository) ListMemosByGame(ctx context.Context, gameID string) ([]domain.Memo, error) {
	return repository.listMemosByGame(ctx, gameID)
}

func (repository fakeMemoRepository) ListAllMemos(ctx context.Context) ([]domain.Memo, error) {
	return repository.listAllMemosFn(ctx)
}

func (repository fakeMemoRepository) DeleteMemo(ctx context.Context, memoID string) error {
	return repository.deleteMemoFn(ctx, memoID)
}

func TestMemoServiceGetMemoByIDUsesRepositoryBoundary(t *testing.T) {
	t.Parallel()

	service := NewMemoService(fakeMemoRepository{
		createMemoFn: func(ctx context.Context, memo domain.Memo) (*domain.Memo, error) {
			return &memo, nil
		},
		updateMemoFn: func(ctx context.Context, memo domain.Memo) (*domain.Memo, error) {
			return &memo, nil
		},
		getMemoByIDFn: func(ctx context.Context, memoID string) (*domain.Memo, error) {
			return &domain.Memo{ID: memoID, Title: "Memo", GameID: "game-1"}, nil
		},
		findMemoByTitle: func(ctx context.Context, gameID string, title string) (*domain.Memo, error) {
			return nil, nil
		},
		listMemosByGame: func(ctx context.Context, gameID string) ([]domain.Memo, error) {
			return nil, nil
		},
		listAllMemosFn: func(ctx context.Context) ([]domain.Memo, error) {
			return nil, nil
		},
		deleteMemoFn: func(ctx context.Context, memoID string) error {
			return nil
		},
	}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := service.GetMemoByID(context.Background(), "memo-1")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if result == nil || result.ID != "memo-1" {
		t.Fatalf("expected memo to be returned")
	}
}

func TestMemoServiceFindMemoByTitleRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	service := NewMemoService(fakeMemoRepository{
		createMemoFn:    func(ctx context.Context, memo domain.Memo) (*domain.Memo, error) { return &memo, nil },
		updateMemoFn:    func(ctx context.Context, memo domain.Memo) (*domain.Memo, error) { return &memo, nil },
		getMemoByIDFn:   func(ctx context.Context, memoID string) (*domain.Memo, error) { return nil, nil },
		findMemoByTitle: func(ctx context.Context, gameID string, title string) (*domain.Memo, error) { return nil, nil },
		listMemosByGame: func(ctx context.Context, gameID string) ([]domain.Memo, error) { return nil, nil },
		listAllMemosFn:  func(ctx context.Context) ([]domain.Memo, error) { return nil, nil },
		deleteMemoFn:    func(ctx context.Context, memoID string) error { return nil },
	}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.FindMemoByTitle(context.Background(), "", "title")
	if err == nil {
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
		createResult: &domain.Memo{ID: "memo-1", Title: "Memo", Content: "Body", GameID: "game-1"},
	}
	service := NewMemoService(repository, manager, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.CreateMemo(context.Background(), MemoInput{
		Title:   "Memo",
		Content: "Body",
		GameID:  "game-1",
	})

	if err == nil {
		t.Fatalf("expected file write failure")
	}
	if repository.deleteMemoCalls != 1 {
		t.Fatalf("expected memo delete rollback to be called once")
	}
}

func TestMemoServiceCreateMemoPreservesExplicitID(t *testing.T) {
	t.Parallel()

	manager := memo.NewFileManager(t.TempDir())
	repository := &trackingMemoRepository{}
	service := NewMemoService(repository, manager, slog.New(slog.NewTextHandler(io.Discard, nil)))

	created, err := service.CreateMemo(context.Background(), MemoInput{
		ID:      "cloud-memo-id",
		Title:   "Synced",
		Content: "Body",
		GameID:  "game-1",
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if created == nil || created.ID != "cloud-memo-id" {
		t.Fatalf("expected cloud memo ID to be preserved, got %#v", created)
	}
	if repository.lastCreated == nil || repository.lastCreated.ID != "cloud-memo-id" {
		t.Fatalf("expected repository to receive explicit ID, got %#v", repository.lastCreated)
	}
}

func TestMemoServiceCreateMemoWritesDatabaseAndLocalFile(t *testing.T) {
	t.Parallel()

	manager := memo.NewFileManager(t.TempDir())
	repository := &trackingMemoRepository{
		createResult: &domain.Memo{ID: "memo-1", Title: "Memo", Content: "Body", GameID: "game-1"},
	}
	service := NewMemoService(repository, manager, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.CreateMemo(context.Background(), MemoInput{
		Title:   " Memo ",
		Content: "Body",
		GameID:  " game-1 ",
	})

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	path := manager.MemoFilePath("game-1", "memo-1", "Memo")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected local memo file to be written: %v", err)
	}
	if !strings.Contains(string(payload), "# Memo") || !strings.Contains(string(payload), "Body") {
		t.Fatalf("expected memo title and body in local file, got %q", string(payload))
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
		getResult: &domain.Memo{ID: "memo-1", Title: "Old", Content: "Old body", GameID: "game-1"},
		updateResults: []*domain.Memo{
			{ID: "memo-1", Title: "New", Content: "New body", GameID: "game-1"},
			{ID: "memo-1", Title: "Old", Content: "Old body", GameID: "game-1"},
		},
	}
	service := NewMemoService(repository, manager, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.UpdateMemo(context.Background(), "memo-1", MemoUpdateInput{
		Title:   "New",
		Content: "New body",
	})

	if err == nil {
		t.Fatalf("expected file update failure")
	}
	if repository.updateMemoCalls != 2 {
		t.Fatalf("expected update rollback to call repository twice")
	}
}

func TestMemoServiceUpdateMemoRenamesLocalFile(t *testing.T) {
	t.Parallel()

	manager := memo.NewFileManager(t.TempDir())
	if _, err := manager.CreateMemoFile("game-1", "memo-1", "Old", "Old body"); err != nil {
		t.Fatalf("failed to create existing memo file: %v", err)
	}
	repository := &trackingMemoRepository{
		getResult: &domain.Memo{ID: "memo-1", Title: "Old", Content: "Old body", GameID: "game-1"},
		updateResults: []*domain.Memo{
			{ID: "memo-1", Title: "New", Content: "New body", GameID: "game-1"},
		},
	}
	service := NewMemoService(repository, manager, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.UpdateMemo(context.Background(), "memo-1", MemoUpdateInput{
		Title:   " New ",
		Content: "New body",
	})

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if _, err := os.Stat(manager.MemoFilePath("game-1", "memo-1", "Old")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected old memo file to be removed, got %v", err)
	}
	payload, err := os.ReadFile(manager.MemoFilePath("game-1", "memo-1", "New"))
	if err != nil {
		t.Fatalf("expected renamed memo file to exist: %v", err)
	}
	if !strings.Contains(string(payload), "# New") || !strings.Contains(string(payload), "New body") {
		t.Fatalf("expected updated memo content, got %q", string(payload))
	}
}

type trackingMemoRepository struct {
	createResult    *domain.Memo
	lastCreated     *domain.Memo
	getResult       *domain.Memo
	findResult      *domain.Memo
	updateResults   []*domain.Memo
	updateMemoCalls int
	deleteMemoCalls int
}

func (repository *trackingMemoRepository) CreateMemo(ctx context.Context, memo domain.Memo) (*domain.Memo, error) {
	copied := memo
	repository.lastCreated = &copied
	if repository.createResult != nil {
		return repository.createResult, nil
	}
	if memo.ID == "" {
		memo.ID = "generated-id"
	}
	return &memo, nil
}
func (repository *trackingMemoRepository) UpdateMemo(ctx context.Context, memo domain.Memo) (*domain.Memo, error) {
	repository.updateMemoCalls++
	if len(repository.updateResults) == 0 {
		return &memo, nil
	}
	result := repository.updateResults[0]
	repository.updateResults = repository.updateResults[1:]
	return result, nil
}
func (repository *trackingMemoRepository) GetMemoByID(ctx context.Context, memoID string) (*domain.Memo, error) {
	return repository.getResult, nil
}
func (repository *trackingMemoRepository) FindMemoByTitle(ctx context.Context, gameID string, title string) (*domain.Memo, error) {
	return repository.findResult, nil
}
func (repository *trackingMemoRepository) ListMemosByGame(ctx context.Context, gameID string) ([]domain.Memo, error) {
	return nil, nil
}
func (repository *trackingMemoRepository) ListAllMemos(ctx context.Context) ([]domain.Memo, error) {
	return nil, nil
}
func (repository *trackingMemoRepository) DeleteMemo(ctx context.Context, memoID string) error {
	repository.deleteMemoCalls++
	return nil
}

func TestMemoServiceListAllMemosHandlesRepositoryError(t *testing.T) {
	t.Parallel()

	service := NewMemoService(fakeMemoRepository{
		createMemoFn: func(ctx context.Context, memo domain.Memo) (*domain.Memo, error) {
			return &memo, nil
		},
		updateMemoFn: func(ctx context.Context, memo domain.Memo) (*domain.Memo, error) {
			return &memo, nil
		},
		getMemoByIDFn: func(ctx context.Context, memoID string) (*domain.Memo, error) {
			return nil, nil
		},
		findMemoByTitle: func(ctx context.Context, gameID string, title string) (*domain.Memo, error) {
			return nil, nil
		},
		listMemosByGame: func(ctx context.Context, gameID string) ([]domain.Memo, error) {
			return nil, nil
		},
		listAllMemosFn: func(ctx context.Context) ([]domain.Memo, error) {
			return nil, errors.New("db down")
		},
		deleteMemoFn: func(ctx context.Context, memoID string) error {
			return nil
		},
	}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.ListAllMemos(context.Background())
	if err == nil {
		t.Fatalf("expected failure")
	}
	if serviceErr := new(ServiceError); !errors.As(err, &serviceErr) || serviceErr.Message == "" {
		t.Fatalf("expected error details, got %v", err)
	}
}

func TestMemoServiceDeleteMemoRemovesDatabaseRecordAndLocalFile(t *testing.T) {
	t.Parallel()

	manager := memo.NewFileManager(t.TempDir())
	if _, err := manager.CreateMemoFile("game-1", "memo-1", "Memo", "Body"); err != nil {
		t.Fatalf("failed to create existing memo file: %v", err)
	}
	repository := &trackingMemoRepository{
		getResult: &domain.Memo{ID: "memo-1", Title: "Memo", Content: "Body", GameID: "game-1"},
	}
	service := NewMemoService(repository, manager, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := service.DeleteMemo(context.Background(), "memo-1"); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if repository.deleteMemoCalls != 1 {
		t.Fatalf("expected database memo to be deleted once")
	}
	if _, err := os.Stat(manager.MemoFilePath("game-1", "memo-1", "Memo")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected local memo file to be removed, got %v", err)
	}
}
