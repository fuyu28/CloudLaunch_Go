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

	if err := service.SetCurrentChapter(context.Background(), "game-1", "chapter-1"); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestChapterServiceListCreateUpdateDeleteUseRepositoryBoundary(t *testing.T) {
	t.Parallel()

	chapter := models.Chapter{ID: "chapter-1", Name: "Chapter 1", Order: 1, GameID: "game-1"}
	repository := fakeChapterRepository{
		listChaptersByGameFn: func(ctx context.Context, gameID string) ([]models.Chapter, error) {
			return []models.Chapter{chapter}, nil
		},
		createChapterFn: func(ctx context.Context, created models.Chapter) (*models.Chapter, error) {
			created.ID = "chapter-1"
			return &created, nil
		},
		getChapterByIDFn: func(ctx context.Context, chapterID string) (*models.Chapter, error) {
			return &chapter, nil
		},
		updateChapterFn: func(ctx context.Context, updated models.Chapter) (*models.Chapter, error) {
			return &updated, nil
		},
		deleteChapterFn:      func(ctx context.Context, chapterID string) error { return nil },
		updateChapterOrderFn: func(ctx context.Context, chapterID string, order int64) error { return nil },
		getChapterStatsFn:    func(ctx context.Context, gameID string) ([]models.ChapterStat, error) { return nil, nil },
		getGameByIDFn:        func(ctx context.Context, gameID string) (*models.Game, error) { return nil, nil },
		updateGameFn:         func(ctx context.Context, game models.Game) (*models.Game, error) { return &game, nil },
	}
	service := NewChapterService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	listed, err := service.ListChaptersByGame(context.Background(), "game-1")
	if err != nil || len(listed) != 1 || listed[0].ID != "chapter-1" {
		t.Fatalf("unexpected listed chapters: %#v", listed)
	}

	created, err := service.CreateChapter(context.Background(), ChapterInput{Name: " Chapter 1 ", Order: 1, GameID: "game-1"})
	if err != nil || created == nil || created.Name != "Chapter 1" {
		t.Fatalf("unexpected create result: %#v", created)
	}

	updated, err := service.UpdateChapter(context.Background(), "chapter-1", ChapterUpdateInput{Name: " Chapter X ", Order: 2})
	if err != nil || updated == nil || updated.Name != "Chapter X" || updated.Order != 2 {
		t.Fatalf("unexpected update result: %#v", updated)
	}

	if err := service.DeleteChapter(context.Background(), "chapter-1"); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
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

	_, err := service.GetChapterStats(context.Background(), "game-1")
	if err == nil {
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

	if err := service.SetCurrentChapter(context.Background(), "game-1", "chapter-1"); err == nil {
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

	if err := service.UpdateChapterOrders(context.Background(), "game-1", []ChapterOrderUpdate{{ID: "chapter-1", Order: -1}}); err == nil {
		t.Fatalf("expected invalid order to fail")
	}
}
