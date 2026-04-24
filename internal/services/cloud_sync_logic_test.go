package services

import (
	"testing"
	"time"

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
