package services

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/storage"
)

func TestDetermineGameSyncAction(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)

	tests := []struct {
		name     string
		local    localGameBundle
		hasLocal bool
		cloud    storage.CloudGameMetadata
		hasCloud bool
		want     gameSyncAction
	}{
		{
			name:     "upload when only local exists",
			local:    localGameBundle{Game: models.Game{UpdatedAt: later}},
			hasLocal: true,
			want:     gameSyncActionUpload,
		},
		{
			name:     "download when only cloud exists",
			cloud:    storage.CloudGameMetadata{UpdatedAt: later},
			hasCloud: true,
			want:     gameSyncActionDownload,
		},
		{
			name:     "upload when local is newer",
			local:    localGameBundle{Game: models.Game{UpdatedAt: later}},
			hasLocal: true,
			cloud:    storage.CloudGameMetadata{UpdatedAt: now},
			hasCloud: true,
			want:     gameSyncActionUpload,
		},
		{
			name:     "download when cloud is newer",
			local:    localGameBundle{Game: models.Game{UpdatedAt: now}},
			hasLocal: true,
			cloud:    storage.CloudGameMetadata{UpdatedAt: later},
			hasCloud: true,
			want:     gameSyncActionDownload,
		},
		{
			name:     "skip when timestamps are equal",
			local:    localGameBundle{Game: models.Game{UpdatedAt: now}},
			hasLocal: true,
			cloud:    storage.CloudGameMetadata{UpdatedAt: now},
			hasCloud: true,
			want:     gameSyncActionSkip,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := determineGameSyncAction(test.local, test.hasLocal, test.cloud, test.hasCloud)
			if got != test.want {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
		})
	}
}

func TestMapToSortedGamesOrdersByTitleThenID(t *testing.T) {
	t.Parallel()

	games := mapToSortedGames(map[string]storage.CloudGameMetadata{
		"b": {ID: "b", Title: "Same"},
		"a": {ID: "a", Title: "Same"},
		"c": {ID: "c", Title: "Alpha"},
	})

	if len(games) != 3 {
		t.Fatalf("expected 3 games")
	}
	if games[0].ID != "c" || games[1].ID != "a" || games[2].ID != "b" {
		t.Fatalf("unexpected sort order: %#v", games)
	}
}

func TestCloudMetadataToMapReturnsEmptyMapForNilMetadata(t *testing.T) {
	t.Parallel()

	mapped := cloudMetadataToMap(nil)
	if len(mapped) != 0 {
		t.Fatalf("expected empty map")
	}
}

func TestCollectUnionGameIDsFiltersAndSorts(t *testing.T) {
	t.Parallel()

	localGames := map[string]localGameBundle{
		"game-b": {},
		"game-a": {},
	}
	cloudGames := map[string]storage.CloudGameMetadata{
		"game-c": {ID: "game-c"},
		"game-a": {ID: "game-a"},
	}

	allIDs := collectUnionGameIDs(localGames, cloudGames, "")
	if len(allIDs) != 3 {
		t.Fatalf("expected 3 ids")
	}
	if allIDs[0] != "game-a" || allIDs[1] != "game-b" || allIDs[2] != "game-c" {
		t.Fatalf("unexpected ids order: %#v", allIDs)
	}

	filtered := collectUnionGameIDs(localGames, cloudGames, "game-c")
	if len(filtered) != 1 || filtered[0] != "game-c" {
		t.Fatalf("unexpected filtered ids: %#v", filtered)
	}
}

func TestCloudSyncSummaryAddAggregatesFields(t *testing.T) {
	t.Parallel()

	summary := CloudSyncSummary{
		UploadedGames:   1,
		UploadedImages:  2,
		SkippedGames:    3,
		DownloadedGames: 4,
	}
	summary.add(CloudSyncSummary{
		UploadedGames:      5,
		DownloadedGames:    6,
		UploadedSessions:   7,
		DownloadedSessions: 8,
		UploadedImages:     9,
		DownloadedImages:   10,
		SkippedGames:       11,
	})

	if summary.UploadedGames != 6 ||
		summary.DownloadedGames != 10 ||
		summary.UploadedSessions != 7 ||
		summary.DownloadedSessions != 8 ||
		summary.UploadedImages != 11 ||
		summary.DownloadedImages != 10 ||
		summary.SkippedGames != 14 {
		t.Fatalf("unexpected aggregated summary: %#v", summary)
	}
}

func TestCloudSyncServiceSyncSingleGameSkipKeepsCloudMetadata(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
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

	cloud := storage.CloudGameMetadata{ID: "game-1", Title: "Game", UpdatedAt: now}
	local := localGameBundle{Game: models.Game{ID: "game-1", Title: "Game", UpdatedAt: now}}

	iteration, err := service.syncSingleGame(context.Background(), nil, "", "game-1", local, true, cloud, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if iteration.cloudGame == nil || iteration.cloudGame.ID != "game-1" {
		t.Fatalf("expected cloud metadata to be kept")
	}
	if iteration.summary.SkippedGames != 1 {
		t.Fatalf("expected skip summary")
	}
	if iteration.shouldSaveMetadata {
		t.Fatalf("did not expect metadata save on skip")
	}
}

func TestComposeSyncedLocalGamePreservesLocalWindowsSpecificFields(t *testing.T) {
	t.Parallel()

	hash := "abc"
	hashTime := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	saveFolder := `C:\Users\fuyu\Saved Games\Game`
	imagePath := `C:\CloudLaunch\thumbs\game.png`
	local := &models.Game{
		ID:                     "game-1",
		ExePath:                `C:\Games\game.exe`,
		SaveFolderPath:         &saveFolder,
		LocalSaveHash:          &hash,
		LocalSaveHashUpdatedAt: &hashTime,
	}
	cloud := storage.CloudGameMetadata{
		ID:            "game-1",
		Title:         "Game",
		Publisher:     "Publisher",
		PlayStatus:    string(models.PlayStatusPlaying),
		TotalPlayTime: 120,
		UpdatedAt:     hashTime.Add(time.Hour),
	}

	composed := composeSyncedLocalGame(cloud, local, &imagePath)

	if composed.ExePath != `C:\Games\game.exe` {
		t.Fatalf("expected local exe path to be preserved")
	}
	if composed.SaveFolderPath == nil || *composed.SaveFolderPath != saveFolder {
		t.Fatalf("expected local save folder to be preserved")
	}
	if composed.LocalSaveHash == nil || *composed.LocalSaveHash != hash {
		t.Fatalf("expected local save hash to be preserved")
	}
	if composed.LocalSaveHashUpdatedAt == nil || !composed.LocalSaveHashUpdatedAt.Equal(hashTime) {
		t.Fatalf("expected local save hash timestamp to be preserved")
	}
	if composed.ImagePath == nil || *composed.ImagePath != imagePath {
		t.Fatalf("expected image path to be applied")
	}
}

func TestComposeSyncedLocalGameUsesFallbacksWithoutLocalGame(t *testing.T) {
	t.Parallel()

	cloud := storage.CloudGameMetadata{
		ID:            "game-1",
		Title:         "Game",
		Publisher:     "Publisher",
		PlayStatus:    string(models.PlayStatusPlayed),
		TotalPlayTime: 240,
	}

	composed := composeSyncedLocalGame(cloud, nil, nil)

	if composed.ExePath != UnconfiguredExePath {
		t.Fatalf("expected unconfigured exe path fallback")
	}
	if composed.SaveFolderPath != nil || composed.LocalSaveHash != nil || composed.LocalSaveHashUpdatedAt != nil {
		t.Fatalf("expected local-only fields to be nil without local game")
	}
}
