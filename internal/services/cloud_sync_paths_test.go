package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/infrastructure/storage"
)

func TestPrepareGameSyncStateBuildsMergedStateAndSkipsUpsertWhenSessionsUnchanged(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	sessionTime := now.Add(-time.Hour)
	repository := &trackingCloudSyncRepository{}
	cloudStorage := &fakeCloudSyncStorage{
		loadedSessions: []storage.CloudSessionRecord{
			{ID: "session-1", PlayedAt: sessionTime, Duration: 1800, UpdatedAt: sessionTime},
		},
	}
	service := NewCloudSyncService(config.Config{}, nil, repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage

	local := localGameBundle{
		Game: domain.Game{ID: "game-1", Title: "Local Title", UpdatedAt: now.Add(-2 * time.Hour)},
		Sessions: []domain.PlaySession{
			{ID: "session-1", GameID: "game-1", PlayedAt: sessionTime, Duration: 1800, UpdatedAt: sessionTime},
		},
	}
	cloud := storage.CloudGameMetadata{ID: "game-1", Title: "Cloud Title", UpdatedAt: now}

	state, err := service.prepareGameSyncState(context.Background(), nil, "bucket", "game-1", local, cloud)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if state.mergedSessions.Changed {
		t.Fatalf("expected sessions to be unchanged, got %#v", state.mergedSessions)
	}
	if len(repository.upsertedSessions) != 0 {
		t.Fatalf("expected no local session upsert when sessions unchanged, got %#v", repository.upsertedSessions)
	}

	wantMergedGame := service.mergeCloudGameMetadata(cloud, &local.Game, state.mergedSessions.Sessions)
	if !reflect.DeepEqual(state.mergedGame, wantMergedGame) {
		t.Fatalf("expected merged game %#v, got %#v", wantMergedGame, state.mergedGame)
	}
	wantMergedCloudGame := cloudMetadataFromGame(wantMergedGame, cloud.ImageKey)
	if !reflect.DeepEqual(state.mergedCloudGame, wantMergedCloudGame) {
		t.Fatalf("expected merged cloud game %#v, got %#v", wantMergedCloudGame, state.mergedCloudGame)
	}
}

func TestPrepareGameSyncStateUpsertsLocalSessionsWhenSessionsChanged(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	repository := &trackingCloudSyncRepository{}
	service := NewCloudSyncService(config.Config{}, nil, repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = &fakeCloudSyncStorage{
		loadedSessions: []storage.CloudSessionRecord{
			{ID: "cloud-session-1", PlayedAt: now, Duration: 1800, UpdatedAt: now},
		},
	}

	local := localGameBundle{Game: domain.Game{ID: "game-1", UpdatedAt: now}}
	cloud := storage.CloudGameMetadata{ID: "game-1", UpdatedAt: now}

	state, err := service.prepareGameSyncState(context.Background(), nil, "bucket", "game-1", local, cloud)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !state.mergedSessions.Changed {
		t.Fatalf("expected sessions to be marked changed, got %#v", state.mergedSessions)
	}
	if len(repository.upsertedSessions) != 1 || repository.upsertedSessions[0].ID != "cloud-session-1" {
		t.Fatalf("expected merged cloud session to be upserted locally, got %#v", repository.upsertedSessions)
	}
}

func TestPrepareGameSyncStatePropagatesLoadCloudSessionsError(t *testing.T) {
	t.Parallel()

	loadErr := errors.New("load sessions failed")
	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	service := NewCloudSyncService(config.Config{}, nil, newNoopCloudSyncRepository(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = &fakeCloudSyncStorage{loadSessionsErr: loadErr}

	local := localGameBundle{Game: domain.Game{ID: "game-1", UpdatedAt: now}}
	cloud := storage.CloudGameMetadata{ID: "game-1", UpdatedAt: now}

	_, err := service.prepareGameSyncState(context.Background(), nil, "bucket", "game-1", local, cloud)
	if !errors.Is(err, loadErr) {
		t.Fatalf("expected load cloud sessions error, got %v", err)
	}
}

func TestPrepareGameSyncStatePropagatesUpsertSessionError(t *testing.T) {
	t.Parallel()

	upsertErr := errors.New("upsert session failed")
	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	repository := &trackingCloudSyncRepository{upsertSessionErr: upsertErr}
	service := NewCloudSyncService(config.Config{}, nil, repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = &fakeCloudSyncStorage{
		loadedSessions: []storage.CloudSessionRecord{
			{ID: "cloud-session-1", PlayedAt: now, Duration: 1800, UpdatedAt: now},
		},
	}

	local := localGameBundle{Game: domain.Game{ID: "game-1", UpdatedAt: now}}
	cloud := storage.CloudGameMetadata{ID: "game-1", UpdatedAt: now}

	_, err := service.prepareGameSyncState(context.Background(), nil, "bucket", "game-1", local, cloud)
	if !errors.Is(err, upsertErr) {
		t.Fatalf("expected upsert session error, got %v", err)
	}
}

func TestSyncUploadPathBuildsCloudGameAndAggregatesSessionCounts(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	repository := &trackingCloudSyncRepository{}
	cloudStorage := &fakeCloudSyncStorage{}
	service := NewCloudSyncService(config.Config{}, nil, repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage

	state := gameSyncState{
		mergedGame: domain.Game{ID: "game-1", Title: "Merged Title", Publisher: "Pub", UpdatedAt: now},
		mergedSessions: mergedSessionsResult{
			Sessions:        []storage.CloudSessionRecord{{ID: "session-1", PlayedAt: now, Duration: 600, UpdatedAt: now}},
			UploadedCount:   2,
			DownloadedCount: 1,
			Changed:         true,
		},
	}
	originalCloud := &storage.CloudGameMetadata{ID: "game-1", UpdatedAt: now.Add(-time.Hour)}

	result, err := service.syncUploadPath(context.Background(), nil, "bucket", "game-1", state, originalCloud)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.cloudGame == nil || result.cloudGame.ID != "game-1" || result.cloudGame.Title != "Merged Title" {
		t.Fatalf("expected cloud game built from merged game, got %#v", result.cloudGame)
	}
	want := CloudSyncSummary{
		UploadedGames:      1,
		UploadedSessions:   2,
		DownloadedSessions: 1,
	}
	if result.summary != want {
		t.Fatalf("expected summary %#v, got %#v", want, result.summary)
	}
	if !result.shouldSaveMetadata {
		t.Fatalf("expected shouldSaveMetadata=true on upload path")
	}
	if cloudStorage.savedSessionsKey != cloudSessionsKey("game-1") {
		t.Fatalf("expected merged sessions to be saved to cloud, got key %q", cloudStorage.savedSessionsKey)
	}
}

func TestSyncUploadPathPropagatesBuildCloudGameError(t *testing.T) {
	t.Parallel()

	saveErr := errors.New("save sessions failed")
	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	service := NewCloudSyncService(config.Config{}, nil, newNoopCloudSyncRepository(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = &fakeCloudSyncStorage{saveSessionsErr: saveErr}

	state := gameSyncState{mergedGame: domain.Game{ID: "game-1", UpdatedAt: now}}

	_, err := service.syncUploadPath(context.Background(), nil, "bucket", "game-1", state, nil)
	if !errors.Is(err, saveErr) {
		t.Fatalf("expected build cloud game error, got %v", err)
	}
}

func TestSyncDownloadPathSavesCloudSessionsAndAppliesLocalGameWhenSessionsChanged(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	repository := &trackingCloudSyncRepository{}
	cloudStorage := &fakeCloudSyncStorage{}
	service := NewCloudSyncService(config.Config{}, nil, repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage

	mergedCloudGame := storage.CloudGameMetadata{ID: "game-1", Title: "Cloud Title", UpdatedAt: now}
	state := gameSyncState{
		mergedCloudGame: mergedCloudGame,
		mergedSessions: mergedSessionsResult{
			Sessions:        []storage.CloudSessionRecord{{ID: "session-1", PlayedAt: now, Duration: 600, UpdatedAt: now}},
			UploadedCount:   1,
			DownloadedCount: 2,
			Changed:         true,
		},
	}
	localGame := &domain.Game{ID: "game-1", ExePath: "/local/game.exe", UpdatedAt: now.Add(-time.Hour)}

	result, err := service.syncDownloadPath(context.Background(), nil, "bucket", "game-1", state, localGame)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cloudStorage.savedSessionsKey != cloudSessionsKey("game-1") {
		t.Fatalf("expected merged cloud sessions to be saved, got key %q", cloudStorage.savedSessionsKey)
	}
	if repository.upsertedGame.ID != "game-1" || repository.upsertedGame.ExePath != "/local/game.exe" {
		t.Fatalf("expected merged cloud game applied locally with local exe path preserved, got %#v", repository.upsertedGame)
	}
	if result.cloudGame == nil || result.cloudGame.ID != mergedCloudGame.ID || result.cloudGame.Title != mergedCloudGame.Title {
		t.Fatalf("expected returned cloud game to be the merged cloud game, got %#v", result.cloudGame)
	}
	want := CloudSyncSummary{
		UploadedSessions:   1,
		DownloadedGames:    1,
		DownloadedSessions: 2,
	}
	if result.summary != want {
		t.Fatalf("expected summary %#v, got %#v", want, result.summary)
	}
	if !result.shouldSaveMetadata {
		t.Fatalf("expected shouldSaveMetadata=true when sessions changed")
	}
}

func TestSyncDownloadPathSkipsCloudSessionSaveWhenSessionsUnchanged(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	repository := &trackingCloudSyncRepository{}
	cloudStorage := &fakeCloudSyncStorage{}
	service := NewCloudSyncService(config.Config{}, nil, repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage

	state := gameSyncState{
		mergedCloudGame: storage.CloudGameMetadata{ID: "game-1", UpdatedAt: now},
		mergedSessions:  mergedSessionsResult{Changed: false},
	}
	localGame := &domain.Game{ID: "game-1", UpdatedAt: now.Add(-time.Hour)}

	result, err := service.syncDownloadPath(context.Background(), nil, "bucket", "game-1", state, localGame)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cloudStorage.savedSessionKeys) != 0 {
		t.Fatalf("expected cloud sessions not to be saved when unchanged, got %#v", cloudStorage.savedSessionKeys)
	}
	if result.shouldSaveMetadata {
		t.Fatalf("expected shouldSaveMetadata=false when sessions unchanged")
	}
	if result.summary.DownloadedGames != 1 {
		t.Fatalf("expected DownloadedGames=1, got %#v", result.summary)
	}
}

func TestSyncDownloadPathPropagatesCloudSessionSaveError(t *testing.T) {
	t.Parallel()

	saveErr := errors.New("cloud save sessions failed")
	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	repository := &trackingCloudSyncRepository{}
	service := NewCloudSyncService(config.Config{}, nil, repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = &fakeCloudSyncStorage{saveSessionsErr: saveErr}

	state := gameSyncState{
		mergedCloudGame: storage.CloudGameMetadata{ID: "game-1", UpdatedAt: now},
		mergedSessions:  mergedSessionsResult{Changed: true},
	}
	localGame := &domain.Game{ID: "game-1", UpdatedAt: now}

	_, err := service.syncDownloadPath(context.Background(), nil, "bucket", "game-1", state, localGame)
	if !errors.Is(err, saveErr) {
		t.Fatalf("expected cloud session save error, got %v", err)
	}
	if repository.upsertedGame.ID != "" {
		t.Fatalf("expected no local game upsert before cloud session save failure, got %#v", repository.upsertedGame)
	}
}

func TestSyncSkipPathMarksSkippedAndSkipsMetadataSaveWhenSessionsUnchanged(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	repository := &trackingCloudSyncRepository{}
	cloudStorage := &fakeCloudSyncStorage{}
	service := NewCloudSyncService(config.Config{}, nil, repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage

	state := gameSyncState{
		mergedCloudGame: storage.CloudGameMetadata{ID: "game-1", Title: "Merged", UpdatedAt: now},
		mergedSessions:  mergedSessionsResult{Changed: false},
	}
	localGame := &domain.Game{ID: "game-1", Title: "Local Title", UpdatedAt: now}

	result, err := service.syncSkipPath(context.Background(), nil, "bucket", "game-1", state, localGame)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cloudStorage.savedSessionKeys) != 0 {
		t.Fatalf("expected cloud sessions not to be saved when unchanged, got %#v", cloudStorage.savedSessionKeys)
	}
	if result.summary.SkippedGames != 1 {
		t.Fatalf("expected SkippedGames=1, got %#v", result.summary)
	}
	if result.shouldSaveMetadata {
		t.Fatalf("expected shouldSaveMetadata=false when sessions unchanged")
	}
	if repository.upsertedGame.ID != "game-1" || repository.upsertedGame.Title != "Merged" {
		t.Fatalf("expected merged cloud game to still be applied locally on skip, got %#v", repository.upsertedGame)
	}
}

func TestSyncSkipPathSavesSessionsWithoutSkipSummaryWhenSessionsChanged(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	repository := &trackingCloudSyncRepository{}
	cloudStorage := &fakeCloudSyncStorage{}
	service := NewCloudSyncService(config.Config{}, nil, repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage

	state := gameSyncState{
		mergedCloudGame: storage.CloudGameMetadata{ID: "game-1", UpdatedAt: now},
		mergedSessions: mergedSessionsResult{
			UploadedCount:   1,
			DownloadedCount: 1,
			Changed:         true,
		},
	}
	localGame := &domain.Game{ID: "game-1", UpdatedAt: now}

	result, err := service.syncSkipPath(context.Background(), nil, "bucket", "game-1", state, localGame)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cloudStorage.savedSessionsKey != cloudSessionsKey("game-1") {
		t.Fatalf("expected merged sessions to be saved to cloud, got key %q", cloudStorage.savedSessionsKey)
	}
	want := CloudSyncSummary{UploadedSessions: 1, DownloadedSessions: 1}
	if result.summary != want {
		t.Fatalf("expected summary %#v (no SkippedGames), got %#v", want, result.summary)
	}
	if !result.shouldSaveMetadata {
		t.Fatalf("expected shouldSaveMetadata=true when sessions changed")
	}
}

func TestSyncSkipPathPropagatesCloudSessionSaveError(t *testing.T) {
	t.Parallel()

	saveErr := errors.New("cloud save sessions failed")
	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	repository := &trackingCloudSyncRepository{}
	service := NewCloudSyncService(config.Config{}, nil, repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = &fakeCloudSyncStorage{saveSessionsErr: saveErr}

	state := gameSyncState{
		mergedCloudGame: storage.CloudGameMetadata{ID: "game-1", UpdatedAt: now},
		mergedSessions:  mergedSessionsResult{Changed: true},
	}
	localGame := &domain.Game{ID: "game-1", UpdatedAt: now}

	_, err := service.syncSkipPath(context.Background(), nil, "bucket", "game-1", state, localGame)
	if !errors.Is(err, saveErr) {
		t.Fatalf("expected cloud session save error, got %v", err)
	}
	if repository.upsertedGame.ID != "" {
		t.Fatalf("expected no local game upsert before cloud session save failure, got %#v", repository.upsertedGame)
	}
}
