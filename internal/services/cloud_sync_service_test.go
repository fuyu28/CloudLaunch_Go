package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/storage"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type fakeCloudSyncRepository struct {
	getGameByIDFn              func(ctx context.Context, gameID string) (*models.Game, error)
	listGamesFn                func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error)
	listPlaySessionsByGameFn   func(ctx context.Context, gameID string) ([]models.PlaySession, error)
	upsertGameSyncFn           func(ctx context.Context, game models.Game) error
	deletePlaySessionsByGameFn func(ctx context.Context, gameID string) error
	upsertPlaySessionSyncFn    func(ctx context.Context, session models.PlaySession) error
	sumPlaySessionDurationsFn  func(ctx context.Context, gameID string) (int64, error)
	updateGameTotalPlayTimeFn  func(ctx context.Context, gameID string, totalPlayTime int64) error
	updateGameTotalWithLastFn  func(ctx context.Context, gameID string, totalPlayTime int64, playedAt time.Time) error
}

func (repository fakeCloudSyncRepository) GetGameByID(ctx context.Context, gameID string) (*models.Game, error) {
	return repository.getGameByIDFn(ctx, gameID)
}

func (repository fakeCloudSyncRepository) ListGames(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
	return repository.listGamesFn(ctx, searchText, filter, sortBy, sortDirection)
}

func (repository fakeCloudSyncRepository) ListPlaySessionsByGame(ctx context.Context, gameID string) ([]models.PlaySession, error) {
	return repository.listPlaySessionsByGameFn(ctx, gameID)
}

func (repository fakeCloudSyncRepository) UpsertGameSync(ctx context.Context, game models.Game) error {
	return repository.upsertGameSyncFn(ctx, game)
}

func (repository fakeCloudSyncRepository) DeletePlaySessionsByGame(ctx context.Context, gameID string) error {
	return repository.deletePlaySessionsByGameFn(ctx, gameID)
}

func (repository fakeCloudSyncRepository) UpsertPlaySessionSync(ctx context.Context, session models.PlaySession) error {
	return repository.upsertPlaySessionSyncFn(ctx, session)
}

func (repository fakeCloudSyncRepository) SumPlaySessionDurationsByGame(ctx context.Context, gameID string) (int64, error) {
	return repository.sumPlaySessionDurationsFn(ctx, gameID)
}

func (repository fakeCloudSyncRepository) UpdateGameTotalPlayTime(ctx context.Context, gameID string, totalPlayTime int64) error {
	if repository.updateGameTotalPlayTimeFn == nil {
		return nil
	}
	return repository.updateGameTotalPlayTimeFn(ctx, gameID, totalPlayTime)
}

func (repository fakeCloudSyncRepository) UpdateGameTotalPlayTimeWithLastPlayed(ctx context.Context, gameID string, totalPlayTime int64, playedAt time.Time) error {
	if repository.updateGameTotalWithLastFn == nil {
		return nil
	}
	return repository.updateGameTotalWithLastFn(ctx, gameID, totalPlayTime, playedAt)
}

func TestCloudSyncServiceLoadLocalGamesUsesRepositoryBoundary(t *testing.T) {
	t.Parallel()

	service := NewCloudSyncService(config.Config{}, nil, fakeCloudSyncRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) {
			return &models.Game{ID: gameID, Title: "Game"}, nil
		},
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
		listPlaySessionsByGameFn: func(ctx context.Context, gameID string) ([]models.PlaySession, error) {
			return []models.PlaySession{{ID: "session-1", GameID: gameID, PlayedAt: time.Now(), Duration: 10}}, nil
		},
		upsertGameSyncFn:           func(ctx context.Context, game models.Game) error { return nil },
		deletePlaySessionsByGameFn: func(ctx context.Context, gameID string) error { return nil },
		upsertPlaySessionSyncFn:    func(ctx context.Context, session models.PlaySession) error { return nil },
		sumPlaySessionDurationsFn:  func(ctx context.Context, gameID string) (int64, error) { return 0, nil },
		updateGameTotalPlayTimeFn:  func(ctx context.Context, gameID string, totalPlayTime int64) error { return nil },
		updateGameTotalWithLastFn:  func(ctx context.Context, gameID string, totalPlayTime int64, playedAt time.Time) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := service.loadLocalGames(context.Background(), "game-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected one local game bundle")
	}
	if result["game-1"].Game.ID != "game-1" {
		t.Fatalf("expected bundle keyed by requested game id")
	}
}

func TestCloudSyncServiceLoadLocalGamesReturnsRepositoryError(t *testing.T) {
	t.Parallel()

	service := NewCloudSyncService(config.Config{}, nil, fakeCloudSyncRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) {
			return nil, nil
		},
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, errors.New("db down")
		},
		listPlaySessionsByGameFn:   func(ctx context.Context, gameID string) ([]models.PlaySession, error) { return nil, nil },
		upsertGameSyncFn:           func(ctx context.Context, game models.Game) error { return nil },
		deletePlaySessionsByGameFn: func(ctx context.Context, gameID string) error { return nil },
		upsertPlaySessionSyncFn:    func(ctx context.Context, session models.PlaySession) error { return nil },
		sumPlaySessionDurationsFn:  func(ctx context.Context, gameID string) (int64, error) { return 0, nil },
		updateGameTotalPlayTimeFn:  func(ctx context.Context, gameID string, totalPlayTime int64) error { return nil },
		updateGameTotalWithLastFn:  func(ctx context.Context, gameID string, totalPlayTime int64, playedAt time.Time) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.loadLocalGames(context.Background(), "")
	if err == nil {
		t.Fatalf("expected repository error")
	}
}

func TestCloudSyncServiceLoadLocalGamesLoadsAllGamesWithSessions(t *testing.T) {
	t.Parallel()

	playedAt := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	service := NewCloudSyncService(config.Config{}, nil, fakeCloudSyncRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) {
			return nil, nil
		},
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return []models.Game{
				{ID: "game-2", Title: "Second"},
				{ID: "game-1", Title: "First"},
			}, nil
		},
		listPlaySessionsByGameFn: func(ctx context.Context, gameID string) ([]models.PlaySession, error) {
			return []models.PlaySession{{ID: "session-" + gameID, GameID: gameID, PlayedAt: playedAt, Duration: 30}}, nil
		},
		upsertGameSyncFn:           func(ctx context.Context, game models.Game) error { return nil },
		deletePlaySessionsByGameFn: func(ctx context.Context, gameID string) error { return nil },
		upsertPlaySessionSyncFn:    func(ctx context.Context, session models.PlaySession) error { return nil },
		sumPlaySessionDurationsFn:  func(ctx context.Context, gameID string) (int64, error) { return 0, nil },
		updateGameTotalPlayTimeFn:  func(ctx context.Context, gameID string, totalPlayTime int64) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := service.loadLocalGames(context.Background(), "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected two local game bundles, got %d", len(result))
	}
	if result["game-1"].Game.Title != "First" || len(result["game-1"].Sessions) != 1 {
		t.Fatalf("expected first game and sessions to be loaded: %#v", result["game-1"])
	}
	if result["game-2"].Sessions[0].GameID != "game-2" {
		t.Fatalf("expected sessions to be loaded per game: %#v", result["game-2"].Sessions)
	}
}

func TestCloudSyncServiceLoadLocalGamesReturnsEmptyWhenRequestedGameIsMissing(t *testing.T) {
	t.Parallel()

	service := NewCloudSyncService(config.Config{}, nil, fakeCloudSyncRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) {
			return nil, nil
		},
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
		listPlaySessionsByGameFn:   func(ctx context.Context, gameID string) ([]models.PlaySession, error) { return nil, nil },
		upsertGameSyncFn:           func(ctx context.Context, game models.Game) error { return nil },
		deletePlaySessionsByGameFn: func(ctx context.Context, gameID string) error { return nil },
		upsertPlaySessionSyncFn:    func(ctx context.Context, session models.PlaySession) error { return nil },
		sumPlaySessionDurationsFn:  func(ctx context.Context, gameID string) (int64, error) { return 0, nil },
		updateGameTotalPlayTimeFn:  func(ctx context.Context, gameID string, totalPlayTime int64) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := service.loadLocalGames(context.Background(), "missing-game")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected no local games for missing id, got %#v", result)
	}
}

func TestCloudSyncServiceSyncGameRejectsInvalidGameID(t *testing.T) {
	t.Parallel()

	service := NewCloudSyncService(config.Config{}, nil, fakeCloudSyncRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) { return nil, nil },
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
		listPlaySessionsByGameFn:   func(ctx context.Context, gameID string) ([]models.PlaySession, error) { return nil, nil },
		upsertGameSyncFn:           func(ctx context.Context, game models.Game) error { return nil },
		deletePlaySessionsByGameFn: func(ctx context.Context, gameID string) error { return nil },
		upsertPlaySessionSyncFn:    func(ctx context.Context, session models.PlaySession) error { return nil },
		sumPlaySessionDurationsFn:  func(ctx context.Context, gameID string) (int64, error) { return 0, nil },
		updateGameTotalPlayTimeFn:  func(ctx context.Context, gameID string, totalPlayTime int64) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.SyncGame(context.Background(), "default", "   ")
	if err == nil {
		t.Fatalf("expected invalid game id to fail")
	}
}

func TestCloudSyncServiceSyncAllGamesFailsInOfflineMode(t *testing.T) {
	t.Parallel()

	service := NewCloudSyncService(config.Config{}, nil, fakeCloudSyncRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) { return nil, nil },
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
		listPlaySessionsByGameFn:   func(ctx context.Context, gameID string) ([]models.PlaySession, error) { return nil, nil },
		upsertGameSyncFn:           func(ctx context.Context, game models.Game) error { return nil },
		deletePlaySessionsByGameFn: func(ctx context.Context, gameID string) error { return nil },
		upsertPlaySessionSyncFn:    func(ctx context.Context, session models.PlaySession) error { return nil },
		sumPlaySessionDurationsFn:  func(ctx context.Context, gameID string) (int64, error) { return 0, nil },
		updateGameTotalPlayTimeFn:  func(ctx context.Context, gameID string, totalPlayTime int64) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.SetOfflineMode(true)

	_, err := service.SyncAllGames(context.Background(), "default")
	if err == nil {
		t.Fatalf("expected offline sync to fail")
	}
}

func TestCloudSyncServiceSyncAllGamesUploadsLocalGamesAndSavesMetadata(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	cloudStorage := &fakeCloudSyncStorage{}
	service := NewCloudSyncService(config.Config{CloudMetadataKey: "metadata.json"}, nil, fakeCloudSyncRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) {
			return nil, nil
		},
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return []models.Game{
				{ID: "game-b", Title: "Beta", Publisher: "Publisher", UpdatedAt: updatedAt},
				{ID: "game-a", Title: "Alpha", Publisher: "Publisher", UpdatedAt: updatedAt.Add(time.Hour)},
			}, nil
		},
		listPlaySessionsByGameFn: func(ctx context.Context, gameID string) ([]models.PlaySession, error) {
			return []models.PlaySession{{ID: "session-" + gameID, GameID: gameID, PlayedAt: updatedAt, Duration: 30, UpdatedAt: updatedAt}}, nil
		},
		upsertGameSyncFn:           func(ctx context.Context, game models.Game) error { return nil },
		deletePlaySessionsByGameFn: func(ctx context.Context, gameID string) error { return nil },
		upsertPlaySessionSyncFn:    func(ctx context.Context, session models.PlaySession) error { return nil },
		sumPlaySessionDurationsFn:  func(ctx context.Context, gameID string) (int64, error) { return 0, nil },
		updateGameTotalPlayTimeFn:  func(ctx context.Context, gameID string, totalPlayTime int64) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage
	service.newClient = func(ctx context.Context, credentialKey string) (*s3.Client, storage.S3Config, string, string, bool) {
		return nil, storage.S3Config{Bucket: "bucket"}, "", "", true
	}

	result, err := service.SyncAllGames(context.Background(), "default")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if result.UploadedGames != 2 || result.UploadedSessions != 2 {
		t.Fatalf("expected upload summary, got %#v", result)
	}
	if cloudStorage.savedMetadata == nil {
		t.Fatalf("expected metadata to be saved")
	}
	if len(cloudStorage.savedMetadata.Games) != 2 {
		t.Fatalf("expected two metadata games, got %#v", cloudStorage.savedMetadata.Games)
	}
	if cloudStorage.savedMetadata.Games[0].ID != "game-a" || cloudStorage.savedMetadata.Games[1].ID != "game-b" {
		t.Fatalf("expected metadata games to be sorted by title, got %#v", cloudStorage.savedMetadata.Games)
	}
	if len(cloudStorage.savedSessionKeys) != 2 {
		t.Fatalf("expected sessions for both games to be saved, got %#v", cloudStorage.savedSessionKeys)
	}
}

func TestCloudSyncServiceDeleteGameFromCloudFailsInOfflineMode(t *testing.T) {
	t.Parallel()

	service := NewCloudSyncService(config.Config{}, nil, fakeCloudSyncRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) { return nil, nil },
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
		listPlaySessionsByGameFn:   func(ctx context.Context, gameID string) ([]models.PlaySession, error) { return nil, nil },
		upsertGameSyncFn:           func(ctx context.Context, game models.Game) error { return nil },
		deletePlaySessionsByGameFn: func(ctx context.Context, gameID string) error { return nil },
		upsertPlaySessionSyncFn:    func(ctx context.Context, session models.PlaySession) error { return nil },
		sumPlaySessionDurationsFn:  func(ctx context.Context, gameID string) (int64, error) { return 0, nil },
		updateGameTotalPlayTimeFn:  func(ctx context.Context, gameID string, totalPlayTime int64) error { return nil },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.SetOfflineMode(true)

	err := service.DeleteGameFromCloud(context.Background(), "default", "game-1")
	if err == nil {
		t.Fatalf("expected offline delete to fail")
	}
}

func TestCloudSyncServiceDeleteGameFromCloudDeletesObjectsAndMetadata(t *testing.T) {
	t.Parallel()

	cloudStorage := &fakeCloudSyncStorage{
		savedMetadata: &storage.CloudMetadata{
			Version: 2,
			Games: []storage.CloudGameMetadata{
				{ID: "game-1", Title: "Delete Me"},
				{ID: "game-2", Title: "Keep Me"},
			},
		},
	}
	service := NewCloudSyncService(config.Config{CloudMetadataKey: "metadata.json"}, nil, newNoopCloudSyncRepository(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage
	service.newClient = func(ctx context.Context, credentialKey string) (*s3.Client, storage.S3Config, string, string, bool) {
		return nil, storage.S3Config{Bucket: "bucket"}, "", "", true
	}

	err := service.DeleteGameFromCloud(context.Background(), "default", "game-1")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if cloudStorage.deletedPrefix != "games/game-1/" {
		t.Fatalf("expected game object prefix to be deleted, got %q", cloudStorage.deletedPrefix)
	}
	if cloudStorage.savedMetadata == nil || len(cloudStorage.savedMetadata.Games) != 1 {
		t.Fatalf("expected metadata to keep one game, got %#v", cloudStorage.savedMetadata)
	}
	if cloudStorage.savedMetadata.Games[0].ID != "game-2" {
		t.Fatalf("expected remaining metadata game to be game-2, got %#v", cloudStorage.savedMetadata.Games)
	}
	if cloudStorage.savedMetadata.UpdatedAt.IsZero() {
		t.Fatalf("expected metadata updated timestamp to be refreshed")
	}
}

func TestMergeSessions_PreservesUniqueSessionsOnBothSides(t *testing.T) {
	localUpdatedAt := time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC)
	cloudUpdatedAt := time.Date(2026, 4, 30, 11, 0, 0, 0, time.UTC)
	local := []models.PlaySession{
		{
			ID:        "local-only",
			PlayedAt:  time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC),
			Duration:  120,
			UpdatedAt: localUpdatedAt,
		},
	}
	cloud := []storage.CloudSessionRecord{
		{
			ID:        "cloud-only",
			PlayedAt:  time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC),
			Duration:  240,
			UpdatedAt: cloudUpdatedAt,
		},
	}

	result := mergeSessions(local, cloud)

	if !result.Changed {
		t.Fatalf("expected merged result to be marked as changed")
	}
	if result.UploadedCount != 1 {
		t.Fatalf("expected UploadedCount=1, got %d", result.UploadedCount)
	}
	if result.DownloadedCount != 1 {
		t.Fatalf("expected DownloadedCount=1, got %d", result.DownloadedCount)
	}
	if len(result.Sessions) != 2 {
		t.Fatalf("expected 2 merged sessions, got %d", len(result.Sessions))
	}
}

func TestMergeSessions_PrefersNewerUpdatedSession(t *testing.T) {
	oldName := "old"
	newName := "new"
	local := []models.PlaySession{
		{
			ID:          "shared",
			PlayedAt:    time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC),
			Duration:    120,
			SessionName: &oldName,
			UpdatedAt:   time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC),
		},
	}
	cloud := []storage.CloudSessionRecord{
		{
			ID:          "shared",
			PlayedAt:    time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC),
			Duration:    300,
			SessionName: &newName,
			UpdatedAt:   time.Date(2026, 4, 30, 11, 0, 0, 0, time.UTC),
		},
	}

	result := mergeSessions(local, cloud)

	if result.DownloadedCount != 1 {
		t.Fatalf("expected DownloadedCount=1, got %d", result.DownloadedCount)
	}
	if result.UploadedCount != 0 {
		t.Fatalf("expected UploadedCount=0, got %d", result.UploadedCount)
	}
	if len(result.Sessions) != 1 {
		t.Fatalf("expected 1 merged session, got %d", len(result.Sessions))
	}
	if result.Sessions[0].Duration != 300 {
		t.Fatalf("expected newer cloud duration to win, got %d", result.Sessions[0].Duration)
	}
	if result.Sessions[0].SessionName == nil || *result.Sessions[0].SessionName != "new" {
		t.Fatalf("expected newer cloud session name to win")
	}
}

func TestMergeSessions_DoesNotMarkEquivalentWhitespaceOnlyDifferenceAsChanged(t *testing.T) {
	localName := "  chapter 1  "
	cloudName := "chapter 1"
	updatedAt := time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC)
	local := []models.PlaySession{
		{
			ID:          "shared",
			PlayedAt:    time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC),
			Duration:    120,
			SessionName: &localName,
			UpdatedAt:   updatedAt,
		},
	}
	cloud := []storage.CloudSessionRecord{
		{
			ID:          "shared",
			PlayedAt:    time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC),
			Duration:    120,
			SessionName: &cloudName,
			UpdatedAt:   updatedAt,
		},
	}

	result := mergeSessions(local, cloud)

	if result.Changed {
		t.Fatalf("expected equivalent whitespace-only difference to be ignored")
	}
	if result.UploadedCount != 0 {
		t.Fatalf("expected UploadedCount=0, got %d", result.UploadedCount)
	}
	if result.DownloadedCount != 0 {
		t.Fatalf("expected DownloadedCount=0, got %d", result.DownloadedCount)
	}
	if len(result.Sessions) != 1 {
		t.Fatalf("expected 1 merged session, got %d", len(result.Sessions))
	}
}

func TestMergeSessions_PrefersLocalWhenUpdatedAtMatchesButPayloadDiffers(t *testing.T) {
	localName := "local"
	cloudName := "cloud"
	updatedAt := time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC)
	local := []models.PlaySession{
		{
			ID:          "shared",
			PlayedAt:    time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC),
			Duration:    120,
			SessionName: &localName,
			UpdatedAt:   updatedAt,
		},
	}
	cloud := []storage.CloudSessionRecord{
		{
			ID:          "shared",
			PlayedAt:    time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC),
			Duration:    300,
			SessionName: &cloudName,
			UpdatedAt:   updatedAt,
		},
	}

	result := mergeSessions(local, cloud)

	if !result.Changed {
		t.Fatalf("expected conflicting payload with same UpdatedAt to be marked as changed")
	}
	if result.UploadedCount != 1 {
		t.Fatalf("expected UploadedCount=1, got %d", result.UploadedCount)
	}
	if result.DownloadedCount != 0 {
		t.Fatalf("expected DownloadedCount=0, got %d", result.DownloadedCount)
	}
	if len(result.Sessions) != 1 {
		t.Fatalf("expected 1 merged session, got %d", len(result.Sessions))
	}
	if result.Sessions[0].Duration != 120 {
		t.Fatalf("expected local session duration to be preserved, got %d", result.Sessions[0].Duration)
	}
	if result.Sessions[0].SessionName == nil || *result.Sessions[0].SessionName != "local" {
		t.Fatalf("expected local session name to be preserved")
	}
}

func TestMergeSessions_SortsByPlayedAtDescThenIDDesc(t *testing.T) {
	local := []models.PlaySession{
		{
			ID:        "a-session",
			PlayedAt:  time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC),
			Duration:  120,
			UpdatedAt: time.Date(2026, 4, 30, 9, 30, 0, 0, time.UTC),
		},
	}
	cloud := []storage.CloudSessionRecord{
		{
			ID:        "z-session",
			PlayedAt:  time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC),
			Duration:  240,
			UpdatedAt: time.Date(2026, 4, 30, 9, 45, 0, 0, time.UTC),
		},
		{
			ID:        "mid-session",
			PlayedAt:  time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC),
			Duration:  300,
			UpdatedAt: time.Date(2026, 4, 30, 10, 15, 0, 0, time.UTC),
		},
	}

	result := mergeSessions(local, cloud)

	if len(result.Sessions) != 3 {
		t.Fatalf("expected 3 merged sessions, got %d", len(result.Sessions))
	}
	if result.Sessions[0].ID != "mid-session" {
		t.Fatalf("expected newest PlayedAt first, got %q", result.Sessions[0].ID)
	}
	if result.Sessions[1].ID != "z-session" {
		t.Fatalf("expected higher ID first when PlayedAt matches, got %q", result.Sessions[1].ID)
	}
	if result.Sessions[2].ID != "a-session" {
		t.Fatalf("expected lower ID last when PlayedAt matches, got %q", result.Sessions[2].ID)
	}
}
