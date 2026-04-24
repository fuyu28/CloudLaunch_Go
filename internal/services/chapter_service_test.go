package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"CloudLaunch_Go/internal/models"
)

type fakeChapterRepository struct {
	listChaptersByGameFn func(ctx context.Context, gameID string) ([]models.Chapter, error)
	createChapterFn      func(ctx context.Context, chapter models.Chapter) (*models.Chapter, error)
	getChapterByIDFn     func(ctx context.Context, chapterID string) (*models.Chapter, error)
	updateChapterFn      func(ctx context.Context, chapter models.Chapter) (*models.Chapter, error)
	deleteChapterFn      func(ctx context.Context, chapterID string) error
	updateChapterOrderFn func(ctx context.Context, chapterID string, order int64) error
	getChapterStatsFn    func(ctx context.Context, gameID string) ([]models.ChapterStat, error)
	getGameByIDFn        func(ctx context.Context, gameID string) (*models.Game, error)
	updateGameFn         func(ctx context.Context, game models.Game) (*models.Game, error)
}

func (repository fakeChapterRepository) ListChaptersByGame(ctx context.Context, gameID string) ([]models.Chapter, error) {
	return repository.listChaptersByGameFn(ctx, gameID)
}

func (repository fakeChapterRepository) CreateChapter(ctx context.Context, chapter models.Chapter) (*models.Chapter, error) {
	return repository.createChapterFn(ctx, chapter)
}

func (repository fakeChapterRepository) GetChapterByID(ctx context.Context, chapterID string) (*models.Chapter, error) {
	return repository.getChapterByIDFn(ctx, chapterID)
}

func (repository fakeChapterRepository) UpdateChapter(ctx context.Context, chapter models.Chapter) (*models.Chapter, error) {
	return repository.updateChapterFn(ctx, chapter)
}

func (repository fakeChapterRepository) DeleteChapter(ctx context.Context, chapterID string) error {
	return repository.deleteChapterFn(ctx, chapterID)
}

func (repository fakeChapterRepository) UpdateChapterOrder(ctx context.Context, chapterID string, order int64) error {
	return repository.updateChapterOrderFn(ctx, chapterID, order)
}

func (repository fakeChapterRepository) GetChapterStats(ctx context.Context, gameID string) ([]models.ChapterStat, error) {
	return repository.getChapterStatsFn(ctx, gameID)
}

func (repository fakeChapterRepository) GetGameByID(ctx context.Context, gameID string) (*models.Game, error) {
	return repository.getGameByIDFn(ctx, gameID)
}

func (repository fakeChapterRepository) UpdateGame(ctx context.Context, game models.Game) (*models.Game, error) {
	return repository.updateGameFn(ctx, game)
}

func TestChapterServiceSetCurrentChapterUsesRepositoryBoundary(t *testing.T) {
	t.Parallel()

	repository := fakeChapterRepository{
		listChaptersByGameFn: func(ctx context.Context, gameID string) ([]models.Chapter, error) { return nil, nil },
		createChapterFn:      func(ctx context.Context, chapter models.Chapter) (*models.Chapter, error) { return &chapter, nil },
		getChapterByIDFn:     func(ctx context.Context, chapterID string) (*models.Chapter, error) { return nil, nil },
		updateChapterFn:      func(ctx context.Context, chapter models.Chapter) (*models.Chapter, error) { return &chapter, nil },
		deleteChapterFn:      func(ctx context.Context, chapterID string) error { return nil },
		updateChapterOrderFn: func(ctx context.Context, chapterID string, order int64) error { return nil },
		getChapterStatsFn:    func(ctx context.Context, gameID string) ([]models.ChapterStat, error) { return nil, nil },
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) {
			return &models.Game{ID: gameID, Title: "Game"}, nil
		},
		updateGameFn: func(ctx context.Context, game models.Game) (*models.Game, error) {
			return &game, nil
		},
	}
	service := NewChapterService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.SetCurrentChapter(context.Background(), "game-1", "chapter-1")

	if !result.Success {
		t.Fatalf("expected success, got %#v", result.Error)
	}
}

func TestChapterServiceGetChapterStatsHandlesRepositoryError(t *testing.T) {
	t.Parallel()

	repository := fakeChapterRepository{
		listChaptersByGameFn: func(ctx context.Context, gameID string) ([]models.Chapter, error) { return nil, nil },
		createChapterFn:      func(ctx context.Context, chapter models.Chapter) (*models.Chapter, error) { return &chapter, nil },
		getChapterByIDFn:     func(ctx context.Context, chapterID string) (*models.Chapter, error) { return nil, nil },
		updateChapterFn:      func(ctx context.Context, chapter models.Chapter) (*models.Chapter, error) { return &chapter, nil },
		deleteChapterFn:      func(ctx context.Context, chapterID string) error { return nil },
		updateChapterOrderFn: func(ctx context.Context, chapterID string, order int64) error { return nil },
		getChapterStatsFn: func(ctx context.Context, gameID string) ([]models.ChapterStat, error) {
			return nil, errors.New("db down")
		},
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) { return nil, nil },
		updateGameFn:  func(ctx context.Context, game models.Game) (*models.Game, error) { return &game, nil },
	}
	service := NewChapterService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.GetChapterStats(context.Background(), "game-1")

	if result.Success {
		t.Fatalf("expected failure")
	}
}

func TestChapterServiceSetCurrentChapterReturnsNotFoundWhenGameMissing(t *testing.T) {
	t.Parallel()

	repository := fakeChapterRepository{
		listChaptersByGameFn: func(ctx context.Context, gameID string) ([]models.Chapter, error) { return nil, nil },
		createChapterFn:      func(ctx context.Context, chapter models.Chapter) (*models.Chapter, error) { return &chapter, nil },
		getChapterByIDFn:     func(ctx context.Context, chapterID string) (*models.Chapter, error) { return nil, nil },
		updateChapterFn:      func(ctx context.Context, chapter models.Chapter) (*models.Chapter, error) { return &chapter, nil },
		deleteChapterFn:      func(ctx context.Context, chapterID string) error { return nil },
		updateChapterOrderFn: func(ctx context.Context, chapterID string, order int64) error { return nil },
		getChapterStatsFn:    func(ctx context.Context, gameID string) ([]models.ChapterStat, error) { return nil, nil },
		getGameByIDFn:        func(ctx context.Context, gameID string) (*models.Game, error) { return nil, nil },
		updateGameFn:         func(ctx context.Context, game models.Game) (*models.Game, error) { return &game, nil },
	}
	service := NewChapterService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.SetCurrentChapter(context.Background(), "game-1", "chapter-1")

	if result.Success {
		t.Fatalf("expected missing game to fail")
	}
}

func TestChapterServiceUpdateChapterOrdersRejectsNegativeOrder(t *testing.T) {
	t.Parallel()

	repository := fakeChapterRepository{
		listChaptersByGameFn: func(ctx context.Context, gameID string) ([]models.Chapter, error) { return nil, nil },
		createChapterFn:      func(ctx context.Context, chapter models.Chapter) (*models.Chapter, error) { return &chapter, nil },
		getChapterByIDFn:     func(ctx context.Context, chapterID string) (*models.Chapter, error) { return nil, nil },
		updateChapterFn:      func(ctx context.Context, chapter models.Chapter) (*models.Chapter, error) { return &chapter, nil },
		deleteChapterFn:      func(ctx context.Context, chapterID string) error { return nil },
		updateChapterOrderFn: func(ctx context.Context, chapterID string, order int64) error { return nil },
		getChapterStatsFn:    func(ctx context.Context, gameID string) ([]models.ChapterStat, error) { return nil, nil },
		getGameByIDFn:        func(ctx context.Context, gameID string) (*models.Game, error) { return nil, nil },
		updateGameFn:         func(ctx context.Context, game models.Game) (*models.Game, error) { return &game, nil },
	}
	service := NewChapterService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := service.UpdateChapterOrders(context.Background(), "game-1", []ChapterOrderUpdate{{ID: "chapter-1", Order: -1}})

	if result.Success {
		t.Fatalf("expected invalid order to fail")
	}
}
