package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/models"
	"CloudLaunch_Go/internal/storage"

	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	service.cloudStorage = &fakeCloudSyncStorage{loadedSessions: []storage.CloudSessionRecord{}}

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

func TestCloudSyncServiceSyncSingleGameUploadSavesSessions(t *testing.T) {
	t.Parallel()

	playedAt := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	cloudStorage := &fakeCloudSyncStorage{}
	service := NewCloudSyncService(config.Config{}, nil, newNoopCloudSyncRepository(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage
	local := localGameBundle{
		Game: models.Game{
			ID:            "game-1",
			Title:         "Game",
			Publisher:     "Publisher",
			PlayStatus:    models.PlayStatusPlaying,
			UpdatedAt:     playedAt.Add(time.Hour),
			CreatedAt:     playedAt,
			LastPlayed:    &playedAt,
			TotalPlayTime: 90,
		},
		Sessions: []models.PlaySession{
			{ID: "session-1", GameID: "game-1", PlayedAt: playedAt, Duration: 30, UpdatedAt: playedAt},
			{ID: "session-2", GameID: "game-1", PlayedAt: playedAt.Add(time.Hour), Duration: 60, UpdatedAt: playedAt.Add(time.Hour)},
		},
	}

	iteration, err := service.syncSingleGame(context.Background(), nil, "bucket", "game-1", local, true, storage.CloudGameMetadata{}, false)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if iteration.cloudGame == nil || iteration.cloudGame.ID != "game-1" || iteration.cloudGame.TotalPlayTime != 90 {
		t.Fatalf("expected uploaded cloud metadata, got %#v", iteration.cloudGame)
	}
	if iteration.summary.UploadedGames != 1 || iteration.summary.UploadedSessions != 2 {
		t.Fatalf("expected upload summary, got %#v", iteration.summary)
	}
	if !iteration.shouldSaveMetadata {
		t.Fatalf("expected metadata save to be requested")
	}
	if cloudStorage.savedSessionsKey != cloudSessionsKey("game-1") {
		t.Fatalf("expected sessions to be saved under game key, got %q", cloudStorage.savedSessionsKey)
	}
	if len(cloudStorage.savedSessions) != 2 || cloudStorage.savedSessions[1].Duration != 60 {
		t.Fatalf("expected cloud sessions to be saved, got %#v", cloudStorage.savedSessions)
	}
}

func TestCloudSyncServiceSyncSingleGameUploadReturnsSessionSaveError(t *testing.T) {
	t.Parallel()

	saveErr := errors.New("save sessions failed")
	cloudStorage := &fakeCloudSyncStorage{saveSessionsErr: saveErr}
	service := NewCloudSyncService(config.Config{}, nil, newNoopCloudSyncRepository(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage
	local := localGameBundle{
		Game: models.Game{ID: "game-1", Title: "Game", Publisher: "Publisher", UpdatedAt: time.Now()},
	}

	_, err := service.syncSingleGame(context.Background(), nil, "bucket", "game-1", local, true, storage.CloudGameMetadata{}, false)

	if !errors.Is(err, saveErr) {
		t.Fatalf("expected save sessions error, got %v", err)
	}
}

func TestCloudSyncServiceSyncSingleGameDownloadAppliesGameAndSessions(t *testing.T) {
	t.Parallel()

	playedAt := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	repository := &trackingCloudSyncRepository{}
	cloudStorage := &fakeCloudSyncStorage{
		loadedSessions: []storage.CloudSessionRecord{
			{ID: "session-1", PlayedAt: playedAt, Duration: 30, UpdatedAt: playedAt},
			{ID: "session-2", PlayedAt: playedAt.Add(time.Hour), Duration: 45, UpdatedAt: playedAt.Add(time.Hour)},
		},
	}
	service := NewCloudSyncService(config.Config{}, nil, repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage
	cloud := storage.CloudGameMetadata{
		ID:            "game-1",
		Title:         "Cloud Game",
		Publisher:     "Cloud Publisher",
		PlayStatus:    "played",
		TotalPlayTime: 75,
		CreatedAt:     playedAt.Add(-24 * time.Hour),
		UpdatedAt:     playedAt,
	}

	iteration, err := service.syncSingleGame(context.Background(), nil, "bucket", "game-1", localGameBundle{}, false, cloud, true)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if iteration.summary.DownloadedGames != 1 || iteration.summary.DownloadedSessions != 2 {
		t.Fatalf("expected download summary, got %#v", iteration.summary)
	}
	if repository.upsertedGame.ID != "game-1" || repository.upsertedGame.Title != "Cloud Game" {
		t.Fatalf("expected cloud game to be upserted locally, got %#v", repository.upsertedGame)
	}
	if repository.upsertedGame.PlayStatus != models.PlayStatusCleared {
		t.Fatalf("expected legacy cloud play status to normalize to cleared, got %#v", repository.upsertedGame.PlayStatus)
	}
	if len(repository.upsertedSessions) != 2 || repository.upsertedSessions[1].Duration != 45 {
		t.Fatalf("expected cloud sessions to be upserted, got %#v", repository.upsertedSessions)
	}
	if repository.updatedTotalGameID != "game-1" || repository.updatedTotalPlayTime != 75 {
		t.Fatalf("expected total play time to be updated, got game=%q total=%d", repository.updatedTotalGameID, repository.updatedTotalPlayTime)
	}
}

func TestCloudSyncServiceSyncSingleGameDownloadReturnsSessionLoadError(t *testing.T) {
	t.Parallel()

	loadErr := errors.New("load sessions failed")
	repository := &trackingCloudSyncRepository{}
	cloudStorage := &fakeCloudSyncStorage{loadSessionsErr: loadErr}
	service := NewCloudSyncService(config.Config{}, nil, repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage
	cloud := storage.CloudGameMetadata{ID: "game-1", Title: "Cloud Game", UpdatedAt: time.Now()}

	_, err := service.syncSingleGame(context.Background(), nil, "bucket", "game-1", localGameBundle{}, false, cloud, true)

	if !errors.Is(err, loadErr) {
		t.Fatalf("expected load sessions error, got %v", err)
	}
	if len(repository.upsertedSessions) != 0 {
		t.Fatalf("expected no sessions to be upserted after load failure")
	}
}

func TestCloudSyncServiceSyncSingleGameDownloadReturnsTotalUpdateError(t *testing.T) {
	t.Parallel()

	updateErr := errors.New("update total failed")
	repository := &trackingCloudSyncRepository{updateTotalErr: updateErr}
	cloudStorage := &fakeCloudSyncStorage{
		loadedSessions: []storage.CloudSessionRecord{
			{ID: "session-1", PlayedAt: time.Now(), Duration: 30, UpdatedAt: time.Now()},
		},
	}
	service := NewCloudSyncService(config.Config{}, nil, repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage
	cloud := storage.CloudGameMetadata{ID: "game-1", Title: "Cloud Game", UpdatedAt: time.Now()}

	_, err := service.syncSingleGame(context.Background(), nil, "bucket", "game-1", localGameBundle{}, false, cloud, true)

	if !errors.Is(err, updateErr) {
		t.Fatalf("expected total update error, got %v", err)
	}
	if len(repository.upsertedSessions) != 1 {
		t.Fatalf("expected sessions to be upserted before total update failure")
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

func TestCloudSyncServiceDownloadImageIfNeededSkipsExistingLocalFile(t *testing.T) {
	t.Parallel()

	imageFiles := &fakeCloudImageFileStore{
		existingPaths: map[string]bool{
			"appdata/thumbnails/hash_game-1.png": true,
		},
	}
	cloudStorage := &fakeCloudSyncStorage{downloadedObject: []byte("image")}
	service := NewCloudSyncService(config.Config{AppDataDir: "appdata"}, nil, newNoopCloudSyncRepository(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage
	service.imageFiles = imageFiles

	path, downloaded, err := service.downloadImageIfNeeded(context.Background(), nil, "bucket", "game-1", "games/game-1/thumbnail/hash.png")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if downloaded {
		t.Fatalf("expected existing image to be reused")
	}
	if path != "appdata/thumbnails/hash_game-1.png" {
		t.Fatalf("unexpected image path: %q", path)
	}
	if cloudStorage.downloadedKey != "" {
		t.Fatalf("expected cloud object not to be downloaded, got %q", cloudStorage.downloadedKey)
	}
	if len(imageFiles.writtenPayload) != 0 {
		t.Fatalf("expected no local write for existing image")
	}
}

func TestCloudSyncServiceDownloadImageIfNeededReturnsWriteError(t *testing.T) {
	t.Parallel()

	writeErr := errors.New("disk full")
	imageFiles := &fakeCloudImageFileStore{
		existingPaths: map[string]bool{},
		writeErr:      writeErr,
	}
	cloudStorage := &fakeCloudSyncStorage{downloadedObject: []byte("image")}
	service := NewCloudSyncService(config.Config{AppDataDir: "appdata"}, nil, newNoopCloudSyncRepository(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage
	service.imageFiles = imageFiles

	_, _, err := service.downloadImageIfNeeded(context.Background(), nil, "bucket", "game-1", "games/game-1/thumbnail/hash.png")

	if !errors.Is(err, writeErr) {
		t.Fatalf("expected write error, got %v", err)
	}
	if cloudStorage.downloadedKey != "games/game-1/thumbnail/hash.png" {
		t.Fatalf("expected cloud object to be downloaded before write, got %q", cloudStorage.downloadedKey)
	}
}

func TestCloudSyncServiceUploadImageIfNeededUploadsLoadedPayload(t *testing.T) {
	t.Parallel()

	payload := []byte("image payload")
	hash := sha256.Sum256(payload)
	expectedKey := "games/game-1/thumbnail/" + hex.EncodeToString(hash[:]) + ".jpeg"
	cloudStorage := &fakeCloudSyncStorage{}
	imageLoader := &fakeCloudImageLoader{
		payload:     payload,
		ext:         ".jpeg",
		contentType: "image/jpeg",
	}
	service := NewCloudSyncService(config.Config{}, nil, newNoopCloudSyncRepository(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage
	service.imageLoader = imageLoader

	key, uploaded, err := service.uploadImageIfNeeded(context.Background(), nil, "bucket", "game-1", "/tmp/source.jpeg", nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if key != expectedKey {
		t.Fatalf("unexpected uploaded key: %q", key)
	}
	if !uploaded {
		t.Fatalf("expected image to be uploaded")
	}
	if imageLoader.loadedPath != "/tmp/source.jpeg" {
		t.Fatalf("expected source path to be loaded, got %q", imageLoader.loadedPath)
	}
	if cloudStorage.uploadedKey != expectedKey {
		t.Fatalf("expected payload to be uploaded to %q, got %q", expectedKey, cloudStorage.uploadedKey)
	}
	if string(cloudStorage.uploadedPayload) != string(payload) || cloudStorage.uploadedContentType != "image/jpeg" {
		t.Fatalf("expected uploaded payload and content type to be preserved")
	}
}

func TestCloudSyncServiceUploadImageIfNeededSkipsExistingImageKey(t *testing.T) {
	t.Parallel()

	payload := []byte("image payload")
	hash := sha256.Sum256(payload)
	existingKey := "games/game-1/thumbnail/" + hex.EncodeToString(hash[:]) + ".png"
	cloudStorage := &fakeCloudSyncStorage{}
	imageLoader := &fakeCloudImageLoader{
		payload:     payload,
		ext:         ".png",
		contentType: "image/png",
	}
	service := NewCloudSyncService(config.Config{}, nil, newNoopCloudSyncRepository(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.cloudStorage = cloudStorage
	service.imageLoader = imageLoader

	key, uploaded, err := service.uploadImageIfNeeded(
		context.Background(),
		nil,
		"bucket",
		"game-1",
		"/tmp/source.png",
		&storage.CloudGameMetadata{ImageKey: &existingKey},
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if key != existingKey {
		t.Fatalf("expected existing key, got %q", key)
	}
	if uploaded {
		t.Fatalf("expected upload to be skipped")
	}
	if cloudStorage.uploadedKey != "" {
		t.Fatalf("expected no upload, got %q", cloudStorage.uploadedKey)
	}
}

func TestCloudImageObjectKeyBuildsHashBasedThumbnailKey(t *testing.T) {
	t.Parallel()

	payload := []byte("image payload")
	hash := sha256.Sum256(payload)
	expectedKey := "games/game-1/thumbnail/" + hex.EncodeToString(hash[:]) + ".png"

	key := cloudImageObjectKey("game-1", payload, "", "image/png")

	if key != expectedKey {
		t.Fatalf("unexpected object key: %q", key)
	}
}

func TestCloudImageLocalPathBuildsThumbnailPathFromImageKey(t *testing.T) {
	t.Parallel()

	path, err := cloudImageLocalPath("appdata", "game-1", "games/game-1/thumbnail/hash.jpg")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if path != "appdata/thumbnails/hash_game-1.jpg" {
		t.Fatalf("unexpected image path: %q", path)
	}
}

func TestCloudImageLocalPathRejectsKeyWithoutHash(t *testing.T) {
	t.Parallel()

	_, err := cloudImageLocalPath("appdata", "game-1", "games/game-1/thumbnail/.png")

	if err == nil {
		t.Fatalf("expected hash validation error")
	}
}

type fakeCloudSyncStorage struct {
	savedMetadata       *storage.CloudMetadata
	deletedPrefix       string
	savedRoutesKey      string
	savedRouteKeys      []string
	savedRoutes         []storage.CloudPlayRouteRecord
	saveRoutesErr       error
	loadRoutesErr       error
	loadedRoutes        []storage.CloudPlayRouteRecord
	savedSessionsKey    string
	savedSessionKeys    []string
	savedSessions       []storage.CloudSessionRecord
	saveSessionsErr     error
	loadSessionsErr     error
	loadedSessions      []storage.CloudSessionRecord
	uploadedKey         string
	uploadedPayload     []byte
	uploadedContentType string
	downloadedKey       string
	downloadedObject    []byte
}

type fakeCloudImageLoader struct {
	loadedPath  string
	payload     []byte
	ext         string
	contentType string
	err         error
}

func (fake *fakeCloudImageLoader) Load(path string) ([]byte, string, string, error) {
	fake.loadedPath = path
	if fake.err != nil {
		return nil, "", "", fake.err
	}
	return append([]byte(nil), fake.payload...), fake.ext, fake.contentType, nil
}

type fakeCloudImageFileStore struct {
	existingPaths  map[string]bool
	ensuredDir     string
	writtenPath    string
	writtenPayload []byte
	writtenPerm    os.FileMode
	ensureDirErr   error
	existsErr      error
	writeErr       error
}

func (fake *fakeCloudImageFileStore) EnsureDir(path string) error {
	fake.ensuredDir = path
	return fake.ensureDirErr
}

func (fake *fakeCloudImageFileStore) Exists(path string) (bool, error) {
	if fake.existsErr != nil {
		return false, fake.existsErr
	}
	return fake.existingPaths[path], nil
}

func (fake *fakeCloudImageFileStore) WriteFile(path string, payload []byte, perm os.FileMode) error {
	if fake.writeErr != nil {
		return fake.writeErr
	}
	fake.writtenPath = path
	fake.writtenPayload = append([]byte(nil), payload...)
	fake.writtenPerm = perm
	return nil
}

func (fake *fakeCloudSyncStorage) LoadMetadata(ctx context.Context, client *s3.Client, bucket string, key string) (*storage.CloudMetadata, error) {
	return fake.savedMetadata, nil
}

func (fake *fakeCloudSyncStorage) SaveMetadata(ctx context.Context, client *s3.Client, bucket string, key string, metadata storage.CloudMetadata) error {
	stored := metadata
	fake.savedMetadata = &stored
	return nil
}

func (fake *fakeCloudSyncStorage) DeleteObjectsByPrefix(ctx context.Context, client *s3.Client, bucket string, prefix string) error {
	fake.deletedPrefix = prefix
	return nil
}

func (fake *fakeCloudSyncStorage) SavePlayRoutes(ctx context.Context, client *s3.Client, bucket string, key string, routes []storage.CloudPlayRouteRecord) error {
	if fake.saveRoutesErr != nil {
		return fake.saveRoutesErr
	}
	fake.savedRoutesKey = key
	fake.savedRouteKeys = append(fake.savedRouteKeys, key)
	fake.savedRoutes = append([]storage.CloudPlayRouteRecord(nil), routes...)
	return nil
}

func (fake *fakeCloudSyncStorage) LoadPlayRoutes(ctx context.Context, client *s3.Client, bucket string, key string) ([]storage.CloudPlayRouteRecord, error) {
	if fake.loadRoutesErr != nil {
		return nil, fake.loadRoutesErr
	}
	return append([]storage.CloudPlayRouteRecord(nil), fake.loadedRoutes...), nil
}

func (fake *fakeCloudSyncStorage) SaveSessions(ctx context.Context, client *s3.Client, bucket string, key string, sessions []storage.CloudSessionRecord) error {
	if fake.saveSessionsErr != nil {
		return fake.saveSessionsErr
	}
	fake.savedSessionsKey = key
	fake.savedSessionKeys = append(fake.savedSessionKeys, key)
	fake.savedSessions = append([]storage.CloudSessionRecord(nil), sessions...)
	return nil
}

func (fake *fakeCloudSyncStorage) LoadSessions(ctx context.Context, client *s3.Client, bucket string, key string) ([]storage.CloudSessionRecord, error) {
	if fake.loadSessionsErr != nil {
		return nil, fake.loadSessionsErr
	}
	return append([]storage.CloudSessionRecord(nil), fake.loadedSessions...), nil
}

func (fake *fakeCloudSyncStorage) UploadBytes(ctx context.Context, client *s3.Client, bucket string, key string, payload []byte, contentType string) error {
	fake.uploadedKey = key
	fake.uploadedPayload = append([]byte(nil), payload...)
	fake.uploadedContentType = contentType
	return nil
}

func (fake *fakeCloudSyncStorage) DownloadObject(ctx context.Context, client *s3.Client, bucket string, key string) ([]byte, error) {
	fake.downloadedKey = key
	return append([]byte(nil), fake.downloadedObject...), nil
}

type trackingCloudSyncRepository struct {
	upsertedGame          models.Game
	deletedRoutesGameID   string
	deletedSessionsGameID string
	upsertedRoutes        []models.PlayRoute
	upsertedSessions      []models.PlaySession
	updatedTotalGameID    string
	updatedTotalPlayTime  int64
	updatedLastPlayed     time.Time
	upsertGameErr         error
	deleteSessionsErr     error
	upsertSessionErr      error
	sumErr                error
	updateTotalErr        error
}

func newNoopCloudSyncRepository() fakeCloudSyncRepository {
	return fakeCloudSyncRepository{
		getGameByIDFn: func(ctx context.Context, gameID string) (*models.Game, error) { return nil, nil },
		listGamesFn: func(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
			return nil, nil
		},
		listPlayRoutesByGameFn:     func(ctx context.Context, gameID string) ([]models.PlayRoute, error) { return nil, nil },
		listPlaySessionsByGameFn:   func(ctx context.Context, gameID string) ([]models.PlaySession, error) { return nil, nil },
		upsertGameSyncFn:           func(ctx context.Context, game models.Game) error { return nil },
		deletePlayRoutesByGameFn:   func(ctx context.Context, gameID string) error { return nil },
		deletePlaySessionsByGameFn: func(ctx context.Context, gameID string) error { return nil },
		upsertPlayRouteSyncFn:      func(ctx context.Context, route models.PlayRoute) error { return nil },
		upsertPlaySessionSyncFn:    func(ctx context.Context, session models.PlaySession) error { return nil },
		sumPlaySessionDurationsFn:  func(ctx context.Context, gameID string) (int64, error) { return 0, nil },
		updateGameTotalPlayTimeFn:  func(ctx context.Context, gameID string, totalPlayTime int64) error { return nil },
	}
}

func (repository *trackingCloudSyncRepository) GetGameByID(ctx context.Context, gameID string) (*models.Game, error) {
	return nil, nil
}

func (repository *trackingCloudSyncRepository) ListGames(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error) {
	return nil, nil
}

func (repository *trackingCloudSyncRepository) ListPlaySessionsByGame(ctx context.Context, gameID string) ([]models.PlaySession, error) {
	return nil, nil
}

func (repository *trackingCloudSyncRepository) ListPlayRoutesByGame(ctx context.Context, gameID string) ([]models.PlayRoute, error) {
	return nil, nil
}

func (repository *trackingCloudSyncRepository) UpsertGameSync(ctx context.Context, game models.Game) error {
	if repository.upsertGameErr != nil {
		return repository.upsertGameErr
	}
	repository.upsertedGame = game
	return nil
}

func (repository *trackingCloudSyncRepository) DeletePlaySessionsByGame(ctx context.Context, gameID string) error {
	if repository.deleteSessionsErr != nil {
		return repository.deleteSessionsErr
	}
	repository.deletedSessionsGameID = gameID
	return nil
}

func (repository *trackingCloudSyncRepository) DeletePlayRoutesByGame(ctx context.Context, gameID string) error {
	repository.deletedRoutesGameID = gameID
	return nil
}

func (repository *trackingCloudSyncRepository) UpsertPlaySessionSync(ctx context.Context, session models.PlaySession) error {
	if repository.upsertSessionErr != nil {
		return repository.upsertSessionErr
	}
	repository.upsertedSessions = append(repository.upsertedSessions, session)
	return nil
}

func (repository *trackingCloudSyncRepository) UpsertPlayRouteSync(ctx context.Context, route models.PlayRoute) error {
	repository.upsertedRoutes = append(repository.upsertedRoutes, route)
	return nil
}

func (repository *trackingCloudSyncRepository) SumPlaySessionDurationsByGame(ctx context.Context, gameID string) (int64, error) {
	if repository.sumErr != nil {
		return 0, repository.sumErr
	}
	var total int64
	for _, session := range repository.upsertedSessions {
		total += session.Duration
	}
	return total, nil
}

func (repository *trackingCloudSyncRepository) UpdateGameTotalPlayTime(ctx context.Context, gameID string, totalPlayTime int64) error {
	if repository.updateTotalErr != nil {
		return repository.updateTotalErr
	}
	repository.updatedTotalGameID = gameID
	repository.updatedTotalPlayTime = totalPlayTime
	return nil
}

func (repository *trackingCloudSyncRepository) UpdateGameTotalPlayTimeWithLastPlayed(ctx context.Context, gameID string, totalPlayTime int64, playedAt time.Time) error {
	if repository.updateTotalErr != nil {
		return repository.updateTotalErr
	}
	repository.updatedTotalGameID = gameID
	repository.updatedTotalPlayTime = totalPlayTime
	repository.updatedLastPlayed = playedAt
	return nil
}

func TestComposeSyncedLocalGameUsesFallbacksWithoutLocalGame(t *testing.T) {
	t.Parallel()

	cloud := storage.CloudGameMetadata{
		ID:            "game-1",
		Title:         "Game",
		Publisher:     "Publisher",
		PlayStatus:    "played",
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

func TestComposeCloudGameMetadataCopiesSyncFields(t *testing.T) {
	t.Parallel()

	lastPlayed := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	clearedAt := lastPlayed.Add(2 * time.Hour)
	game := models.Game{
		ID:            "game-1",
		Title:         "Game",
		Publisher:     "Publisher",
		PlayStatus:    models.PlayStatusCleared,
		TotalPlayTime: 180,
		LastPlayed:    &lastPlayed,
		ClearedAt:     &clearedAt,
		CreatedAt:     lastPlayed.Add(-24 * time.Hour),
		UpdatedAt:     lastPlayed.Add(time.Hour),
	}

	composed := composeCloudGameMetadata(game)

	if composed.ID != game.ID || composed.Title != game.Title || composed.Publisher != game.Publisher {
		t.Fatalf("expected identity fields to be copied")
	}
	if composed.PlayStatus != string(game.PlayStatus) || composed.TotalPlayTime != game.TotalPlayTime {
		t.Fatalf("expected play state to be copied")
	}
	if composed.LastPlayed == nil || !composed.LastPlayed.Equal(lastPlayed) {
		t.Fatalf("expected last played to be copied")
	}
	if composed.ClearedAt == nil || !composed.ClearedAt.Equal(clearedAt) {
		t.Fatalf("expected cleared at to be copied")
	}
	if !composed.CreatedAt.Equal(game.CreatedAt) ||
		!composed.UpdatedAt.Equal(game.UpdatedAt) {
		t.Fatalf("expected sync timestamps to be copied")
	}
}

func TestComposeCloudSessionsCopiesOrderAndSessionFields(t *testing.T) {
	t.Parallel()

	playedAt := time.Date(2026, 4, 24, 8, 0, 0, 0, time.UTC)
	updatedAt := playedAt.Add(30 * time.Minute)
	routeID := "route-1"
	sessions := []models.PlaySession{
		{
			ID:          "session-1",
			PlayRouteID: &routeID,
			PlayedAt:    playedAt,
			Duration:    45,
			UpdatedAt:   updatedAt,
		},
		{
			ID:        "session-2",
			PlayedAt:  playedAt.Add(time.Hour),
			Duration:  30,
			UpdatedAt: updatedAt.Add(time.Hour),
		},
	}

	composed := composeCloudSessions(sessions)

	if len(composed) != 2 {
		t.Fatalf("expected two session records")
	}
	if composed[0].ID != "session-1" || composed[1].ID != "session-2" {
		t.Fatalf("expected session order to be preserved")
	}
	if composed[0].Duration != 45 {
		t.Fatalf("expected first session fields to be copied")
	}
	if composed[0].PlayRouteID == nil || *composed[0].PlayRouteID != routeID {
		t.Fatalf("expected play route id to be copied")
	}
	if !composed[1].PlayedAt.Equal(sessions[1].PlayedAt) || !composed[1].UpdatedAt.Equal(sessions[1].UpdatedAt) {
		t.Fatalf("expected second session timestamps to be copied")
	}
}

func TestComposeCloudRoutesCopiesOrderAndRouteFields(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 4, 24, 7, 0, 0, 0, time.UTC)
	routes := []models.PlayRoute{
		{ID: "route-1", Name: "Common", SortOrder: 0, CreatedAt: createdAt},
		{ID: "route-2", Name: "Heroine", SortOrder: 1, CreatedAt: createdAt.Add(time.Minute)},
	}

	composed := composeCloudRoutes(routes)

	if len(composed) != 2 {
		t.Fatalf("expected two route records")
	}
	if composed[0].ID != "route-1" || composed[1].Name != "Heroine" {
		t.Fatalf("expected route fields to be copied")
	}
	if composed[1].SortOrder != 1 || !composed[1].CreatedAt.Equal(routes[1].CreatedAt) {
		t.Fatalf("expected route metadata to be copied")
	}
}

func TestComposeLocalPlaySessionCopiesCloudFields(t *testing.T) {
	t.Parallel()

	playedAt := time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC)
	updatedAt := playedAt.Add(15 * time.Minute)
	routeID := "route-1"
	cloudSession := storage.CloudSessionRecord{
		ID:          "session-1",
		PlayRouteID: &routeID,
		PlayedAt:    playedAt,
		Duration:    75,
		UpdatedAt:   updatedAt,
	}

	composed := composeLocalPlaySession("game-1", cloudSession)

	if composed.ID != "session-1" || composed.GameID != "game-1" {
		t.Fatalf("expected identifiers to be copied")
	}
	if composed.Duration != 75 {
		t.Fatalf("expected cloud session fields to be copied")
	}
	if composed.PlayRouteID == nil || *composed.PlayRouteID != routeID {
		t.Fatalf("expected play route id to be copied")
	}
	if !composed.PlayedAt.Equal(playedAt) || !composed.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("expected timestamps to be copied")
	}
}

func TestComposeLocalPlayRouteCopiesCloudFields(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 4, 24, 6, 0, 0, 0, time.UTC)
	cloudRoute := storage.CloudPlayRouteRecord{
		ID:        "route-1",
		Name:      "Common",
		SortOrder: 0,
		CreatedAt: createdAt,
	}

	composed := composeLocalPlayRoute("game-1", cloudRoute)

	if composed.ID != "route-1" || composed.GameID != "game-1" {
		t.Fatalf("expected identifiers to be copied")
	}
	if composed.Name != "Common" || composed.SortOrder != 0 || !composed.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected route fields to be copied")
	}
}
