package app

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/services"
)

type startupGameRepository struct {
	pending  []string
	cleared  []string
	listCall int
}

func (repository *startupGameRepository) ListGames(context.Context, string, domain.PlayStatus, string, string) ([]domain.Game, error) {
	return nil, nil
}

func (repository *startupGameRepository) GetGameByID(context.Context, string) (*domain.Game, error) {
	return nil, nil
}

func (repository *startupGameRepository) CreateGameWithInitialRoute(context.Context, domain.Game, domain.Route) (*domain.Game, error) {
	return nil, nil
}

func (repository *startupGameRepository) UpdateGame(_ context.Context, game domain.Game) (*domain.Game, error) {
	return &game, nil
}

func (repository *startupGameRepository) DeleteGameAndQueueMemoCleanup(context.Context, string) error {
	return nil
}

func (repository *startupGameRepository) ListPendingMemoCleanup(context.Context) ([]string, error) {
	repository.listCall++
	return repository.pending, nil
}

func (repository *startupGameRepository) RefreshGamePlayTimeFromSessions(context.Context, string) error {
	return nil
}

func (repository *startupGameRepository) ClearPendingMemoCleanup(_ context.Context, gameID string) error {
	repository.cleared = append(repository.cleared, gameID)
	return nil
}

type startupMemoCleaner struct {
	cleaned []string
}

func (cleaner *startupMemoCleaner) DeleteGameMemoFiles(gameID string) error {
	cleaner.cleaned = append(cleaner.cleaned, gameID)
	return nil
}

func TestStartupRetriesPendingMemoCleanup(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repository := &startupGameRepository{pending: []string{"game-1"}}
	cleaner := &startupMemoCleaner{}
	app := &App{
		Logger:      logger,
		GameService: services.NewGameService(repository, logger, cleaner),
	}

	app.Startup(context.Background())

	if repository.listCall != 1 {
		t.Fatalf("pending cleanup list calls = %d, want 1", repository.listCall)
	}
	if len(cleaner.cleaned) != 1 || cleaner.cleaned[0] != "game-1" {
		t.Fatalf("cleaned game IDs = %#v", cleaner.cleaned)
	}
	if len(repository.cleared) != 1 || repository.cleared[0] != "game-1" {
		t.Fatalf("cleared game IDs = %#v", repository.cleared)
	}
}

type startupContentSyncRepository struct {
	listPendingCalls int
	failList         bool
}

func (r *startupContentSyncRepository) GetGameByID(context.Context, string) (*domain.Game, error) {
	return nil, nil
}
func (r *startupContentSyncRepository) ListPlaySessionsByGame(context.Context, string) ([]domain.PlaySession, error) {
	return nil, nil
}
func (r *startupContentSyncRepository) GetLocalSaveTree(context.Context, string) (string, error) {
	return "", nil
}
func (r *startupContentSyncRepository) SetLocalSyncState(context.Context, string, string, string) error {
	return nil
}
func (r *startupContentSyncRepository) BeginPendingPush(context.Context, domain.PendingPush) error {
	return nil
}
func (r *startupContentSyncRepository) FinalizePendingPush(context.Context, string, string, string) error {
	return nil
}
func (r *startupContentSyncRepository) ClearPendingPush(context.Context, string) error {
	return nil
}
func (r *startupContentSyncRepository) ListPendingPushes(context.Context) ([]domain.PendingPush, error) {
	r.listPendingCalls++
	if r.failList {
		return nil, context.DeadlineExceeded
	}
	return nil, nil
}
func (r *startupContentSyncRepository) ApplyPullResult(context.Context, domain.Game, []domain.PlaySession, string, string) error {
	return nil
}
func (r *startupContentSyncRepository) GetSetting(context.Context, string) (string, error) {
	return "", nil
}
func (r *startupContentSyncRepository) UpsertSetting(context.Context, string, string) error {
	return nil
}

func TestStartupRecoversPendingPushes(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := &startupContentSyncRepository{}
	app := &App{
		Logger:             logger,
		ContentSyncService: services.NewContentSyncService(config.Config{}, nil, repo, logger),
	}

	app.Startup(context.Background())

	if repo.listPendingCalls != 1 {
		t.Fatalf("ListPendingPushes calls = %d, want 1", repo.listPendingCalls)
	}
}

func TestStartupContinuesWhenPendingPushRecoveryFails(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := &startupContentSyncRepository{failList: true}
	app := &App{
		Logger:             logger,
		ContentSyncService: services.NewContentSyncService(config.Config{}, nil, repo, logger),
	}

	// ネットワーク／DB 失敗でも Startup 自体はパニックせず完了する。
	app.Startup(context.Background())

	if repo.listPendingCalls != 1 {
		t.Fatalf("ListPendingPushes calls = %d, want 1", repo.listPendingCalls)
	}
}
