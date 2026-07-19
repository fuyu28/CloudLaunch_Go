package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/infrastructure/credentials"
	"CloudLaunch_Go/internal/services"
)

type noopAppGameRepository struct {
	listErr   error
	createErr error
	deleteErr error
	created   *domain.Game
}

func (r noopAppGameRepository) ListGames(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error) {
	return nil, r.listErr
}

func (r noopAppGameRepository) GetGameByID(ctx context.Context, gameID string) (*domain.Game, error) {
	return nil, nil
}

func (r noopAppGameRepository) CreateGameWithInitialRoute(ctx context.Context, game domain.Game, initialRoute domain.Route) (*domain.Game, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	if r.created != nil {
		return r.created, nil
	}
	return &game, nil
}

func (r noopAppGameRepository) UpdateGame(ctx context.Context, game domain.Game) (*domain.Game, error) {
	return &game, nil
}

func (r noopAppGameRepository) DeleteGameAndQueueMemoCleanup(ctx context.Context, gameID string) error {
	return r.deleteErr
}

func (r noopAppGameRepository) ListPendingMemoCleanup(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (r noopAppGameRepository) ClearPendingMemoCleanup(ctx context.Context, gameID string) error {
	return nil
}

func (r noopAppGameRepository) RefreshGamePlayTimeFromSessions(ctx context.Context, gameID string) error {
	return nil
}

type noopAppSessionRepository struct {
	createErr error
	getErr    error
	deleteErr error
	updateErr error
	session   *domain.PlaySession
}

func (r noopAppSessionRepository) CreatePlaySessionAndRefreshGame(ctx context.Context, session domain.PlaySession) (*domain.PlaySession, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	if r.session != nil {
		return r.session, nil
	}
	return &session, nil
}
func (r noopAppSessionRepository) ListPlaySessionsByGame(ctx context.Context, gameID string) ([]domain.PlaySession, error) {
	return nil, nil
}
func (r noopAppSessionRepository) GetPlaySessionByID(ctx context.Context, sessionID string) (*domain.PlaySession, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.session != nil {
		return r.session, nil
	}
	return &domain.PlaySession{ID: sessionID, GameID: "game-1"}, nil
}
func (r noopAppSessionRepository) DeletePlaySessionAndRefreshGame(ctx context.Context, sessionID string) (string, error) {
	if r.deleteErr != nil {
		return "", r.deleteErr
	}
	if r.session != nil {
		return r.session.GameID, nil
	}
	return "game-1", nil
}
func (r noopAppSessionRepository) UpdatePlaySessionRoute(ctx context.Context, sessionID string, routeID *string) error {
	return r.updateErr
}
func (r noopAppSessionRepository) UpdatePlaySessionName(ctx context.Context, sessionID string, sessionName string) error {
	return r.updateErr
}
func (r noopAppSessionRepository) TouchGameUpdatedAt(ctx context.Context, gameID string) error {
	return nil
}

type noopAppRouteRepository struct {
	listErr error
}

func (r noopAppRouteRepository) ListRoutesByGame(ctx context.Context, gameID string) ([]domain.Route, error) {
	return nil, r.listErr
}
func (r noopAppRouteRepository) CreateRoute(ctx context.Context, route domain.Route) (*domain.Route, error) {
	return &route, nil
}
func (r noopAppRouteRepository) GetRouteByID(ctx context.Context, routeID string) (*domain.Route, error) {
	return nil, nil
}
func (r noopAppRouteRepository) UpdateRoute(ctx context.Context, route domain.Route) (*domain.Route, error) {
	return &route, nil
}
func (r noopAppRouteRepository) DeleteRoute(ctx context.Context, routeID string) error { return nil }
func (r noopAppRouteRepository) UpdateRouteOrder(ctx context.Context, routeID string, order int64) error {
	return nil
}
func (r noopAppRouteRepository) UpdateRouteOrders(ctx context.Context, gameID string, items []domain.RouteOrderItem) error {
	return nil
}
func (r noopAppRouteRepository) GetRouteStats(ctx context.Context, gameID string) ([]domain.RouteStat, error) {
	return nil, nil
}
func (r noopAppRouteRepository) GetGameByID(ctx context.Context, gameID string) (*domain.Game, error) {
	return nil, nil
}
func (r noopAppRouteRepository) UpdateGame(ctx context.Context, game domain.Game) (*domain.Game, error) {
	return &game, nil
}

type noopAppMemoRepository struct{}

func (noopAppMemoRepository) CreateMemo(ctx context.Context, memo domain.Memo) (*domain.Memo, error) {
	return &memo, nil
}

func (noopAppMemoRepository) UpdateMemo(ctx context.Context, memo domain.Memo) (*domain.Memo, error) {
	return &memo, nil
}

func (noopAppMemoRepository) GetMemoByID(ctx context.Context, memoID string) (*domain.Memo, error) {
	return nil, nil
}

func (noopAppMemoRepository) FindMemoByTitle(ctx context.Context, gameID string, title string) (*domain.Memo, error) {
	return nil, nil
}

func (noopAppMemoRepository) ListMemosByGame(ctx context.Context, gameID string) ([]domain.Memo, error) {
	return nil, nil
}

func (noopAppMemoRepository) ListAllMemos(ctx context.Context) ([]domain.Memo, error) {
	return nil, nil
}

func (noopAppMemoRepository) DeleteMemo(ctx context.Context, memoID string) error {
	return nil
}

type adapterTestCredentialStore struct {
	loadResult *credentials.Credential
	loadErr    error
}

func (store *adapterTestCredentialStore) Save(ctx context.Context, key string, credential credentials.Credential) error {
	return nil
}

func (store *adapterTestCredentialStore) Load(ctx context.Context, key string) (*credentials.Credential, error) {
	return store.loadResult, store.loadErr
}

func (store *adapterTestCredentialStore) Delete(ctx context.Context, key string) error {
	return nil
}

func newAdapterTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestAppGetCloudMemosConvertsServiceError(t *testing.T) {
	t.Parallel()

	app := &App{
		Logger: newAdapterTestLogger(),
		MemoCloudService: services.NewMemoCloudService(
			config.Config{},
			&adapterTestCredentialStore{},
			services.NewGameService(noopAppGameRepository{}, newAdapterTestLogger()),
			services.NewMemoService(noopAppMemoRepository{}, nil, newAdapterTestLogger()),
			newAdapterTestLogger(),
		),
	}

	result := app.GetCloudMemos()

	if result.Success {
		t.Fatalf("expected cloud memo failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "クラウドメモ取得に失敗しました" {
		t.Fatalf("expected converted service error, got %#v", result.Error)
	}
}

func TestAppUploadMemoToCloudConvertsNotFoundError(t *testing.T) {
	t.Parallel()

	store := &adapterTestCredentialStore{
		loadResult: &credentials.Credential{
			AccessKeyID:     "access",
			SecretAccessKey: "secret",
			BucketName:      "bucket",
			Region:          "region",
			Endpoint:        "endpoint",
		},
	}
	app := &App{
		Logger: newAdapterTestLogger(),
		MemoCloudService: services.NewMemoCloudService(
			config.Config{},
			store,
			services.NewGameService(noopAppGameRepository{}, newAdapterTestLogger()),
			services.NewMemoService(noopAppMemoRepository{}, nil, newAdapterTestLogger()),
			newAdapterTestLogger(),
		),
	}

	result := app.UploadMemoToCloud("missing-memo")

	if result.Success {
		t.Fatalf("expected upload failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "メモが見つかりません" {
		t.Fatalf("expected converted not found error, got %#v", result.Error)
	}
}

func TestAppListGamesConvertsServiceError(t *testing.T) {
	t.Parallel()

	app := &App{
		Logger:      newAdapterTestLogger(),
		GameService: services.NewGameService(noopAppGameRepository{listErr: errors.New("db fail")}, newAdapterTestLogger()),
	}

	result := app.ListGames("", "", "title", "asc")

	if result.Success {
		t.Fatalf("expected list failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "ゲーム一覧取得に失敗しました" {
		t.Fatalf("expected converted service error, got %#v", result.Error)
	}
}

func TestAppCreateGameConvertsServiceError(t *testing.T) {
	t.Parallel()

	app := &App{
		Logger:      newAdapterTestLogger(),
		GameService: services.NewGameService(noopAppGameRepository{createErr: errors.New("db fail")}, newAdapterTestLogger()),
	}

	result := app.CreateGame(services.GameInput{Title: "Game", Publisher: "Pub", ExePath: "/game.exe"})

	if result.Success {
		t.Fatalf("expected create failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "ゲーム作成に失敗しました" {
		t.Fatalf("expected converted service error, got %#v", result.Error)
	}
}

func TestAppCreateGameReturnsGameOnSuccess(t *testing.T) {
	t.Parallel()

	created := &domain.Game{ID: "game-1", Title: "Game", Publisher: "Pub", ExePath: "/game.exe"}
	app := &App{
		Logger:      newAdapterTestLogger(),
		GameService: services.NewGameService(noopAppGameRepository{created: created}, newAdapterTestLogger()),
	}

	result := app.CreateGame(services.GameInput{Title: "Game", Publisher: "Pub", ExePath: "/game.exe"})

	if !result.Success {
		t.Fatalf("expected create success, got %#v", result)
	}
	if result.Data == nil || result.Data.ID != "game-1" {
		t.Fatalf("expected created game id, got %#v", result.Data)
	}
}

func TestAppDeleteGameConvertsServiceError(t *testing.T) {
	t.Parallel()

	app := &App{
		Logger:      newAdapterTestLogger(),
		GameService: services.NewGameService(noopAppGameRepository{deleteErr: errors.New("db fail")}, newAdapterTestLogger()),
	}

	result := app.DeleteGame("game-1")

	if result.Success {
		t.Fatalf("expected delete failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "ゲーム削除に失敗しました" {
		t.Fatalf("expected converted service error, got %#v", result.Error)
	}
}

func TestAppCreateSessionConvertsServiceError(t *testing.T) {
	t.Parallel()

	app := &App{
		Logger:         newAdapterTestLogger(),
		SessionService: services.NewSessionService(noopAppSessionRepository{createErr: errors.New("db fail")}, newAdapterTestLogger()),
	}

	result := app.CreateSession(services.SessionInput{
		GameID:   "game-1",
		PlayedAt: time.Now(),
		Duration: 60,
	})

	if result.Success {
		t.Fatalf("expected create failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "セッション作成に失敗しました" {
		t.Fatalf("expected converted service error, got %#v", result.Error)
	}
}

func TestAppDeleteSessionConvertsServiceError(t *testing.T) {
	t.Parallel()

	repo := noopAppSessionRepository{deleteErr: errors.New("db fail")}
	app := &App{
		Logger:            newAdapterTestLogger(),
		SessionService:    services.NewSessionService(repo, newAdapterTestLogger()),
		playSessionLookup: repo,
	}

	result := app.DeleteSession("session-1")

	if result.Success {
		t.Fatalf("expected delete failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "セッション削除に失敗しました" {
		t.Fatalf("expected converted service error, got %#v", result.Error)
	}
}

// sessionMutationOrderFixture は adapter の lookup → mutation → sync 順序を記録する。
type sessionMutationOrderFixture struct {
	calls     []string
	session   *domain.PlaySession
	lookupErr error
	deleteErr error
	updateErr error
}

func (f *sessionMutationOrderFixture) GetPlaySessionByID(ctx context.Context, sessionID string) (*domain.PlaySession, error) {
	f.calls = append(f.calls, "lookup")
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	return f.session, nil
}

func (f *sessionMutationOrderFixture) CreatePlaySessionAndRefreshGame(ctx context.Context, session domain.PlaySession) (*domain.PlaySession, error) {
	return &session, nil
}
func (f *sessionMutationOrderFixture) ListPlaySessionsByGame(ctx context.Context, gameID string) ([]domain.PlaySession, error) {
	return nil, nil
}
func (f *sessionMutationOrderFixture) DeletePlaySessionAndRefreshGame(ctx context.Context, sessionID string) (string, error) {
	f.calls = append(f.calls, "mutation")
	if f.deleteErr != nil {
		return "", f.deleteErr
	}
	if f.session != nil {
		return f.session.GameID, nil
	}
	return "", nil
}
func (f *sessionMutationOrderFixture) UpdatePlaySessionRoute(ctx context.Context, sessionID string, routeID *string) error {
	f.calls = append(f.calls, "mutation")
	return f.updateErr
}
func (f *sessionMutationOrderFixture) UpdatePlaySessionName(ctx context.Context, sessionID string, sessionName string) error {
	f.calls = append(f.calls, "mutation")
	return f.updateErr
}
func (f *sessionMutationOrderFixture) TouchGameUpdatedAt(ctx context.Context, gameID string) error {
	return nil
}

func newSessionMutationOrderApp(t *testing.T, fixture *sessionMutationOrderFixture, synced chan string) *App {
	t.Helper()
	app := &App{
		Logger:             newAdapterTestLogger(),
		SessionService:     services.NewSessionService(fixture, newAdapterTestLogger()),
		playSessionLookup:  fixture,
		ContentSyncService: &services.ContentSyncService{},
		syncCoalescer: newAsyncCoalescer(func(id string) {
			fixture.calls = append(fixture.calls, "sync")
			if synced != nil {
				synced <- id
			}
		}),
	}
	return app
}

func waitSessionSync(t *testing.T, synced <-chan string) string {
	t.Helper()
	select {
	case id := <-synced:
		return id
	case <-time.After(2 * time.Second):
		t.Fatal("expected sync to be triggered")
		return ""
	}
}

func assertNoSessionSync(t *testing.T, synced <-chan string) {
	t.Helper()
	select {
	case id := <-synced:
		t.Fatalf("expected no sync, got %q", id)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestAppDeleteSessionLookupMutationSyncOrder(t *testing.T) {
	t.Parallel()

	fixture := &sessionMutationOrderFixture{
		session: &domain.PlaySession{ID: "session-1", GameID: "game-1"},
	}
	synced := make(chan string, 1)
	app := newSessionMutationOrderApp(t, fixture, synced)

	result := app.DeleteSession("session-1")
	if !result.Success || !result.Data {
		t.Fatalf("expected ApiResult[bool] success, got %#v", result)
	}
	if got := waitSessionSync(t, synced); got != "game-1" {
		t.Fatalf("expected sync game-1, got %q", got)
	}
	if want := []string{"lookup", "mutation", "sync"}; !stringSliceEqual(fixture.calls, want) {
		t.Fatalf("expected order %v, got %v", want, fixture.calls)
	}
}

func TestAppDeleteSessionLookupFailureSkipsMutationAndSync(t *testing.T) {
	t.Parallel()

	fixture := &sessionMutationOrderFixture{
		session:   &domain.PlaySession{ID: "session-1", GameID: "game-1"},
		lookupErr: errors.New("db lookup fail"),
	}
	synced := make(chan string, 1)
	app := newSessionMutationOrderApp(t, fixture, synced)

	result := app.DeleteSession("session-1")
	if result.Success {
		t.Fatalf("expected lookup failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "セッション取得に失敗しました" {
		t.Fatalf("expected lookup error message, got %#v", result.Error)
	}
	assertNoSessionSync(t, synced)
	if want := []string{"lookup"}; !stringSliceEqual(fixture.calls, want) {
		t.Fatalf("expected only lookup, got %v", fixture.calls)
	}
}

func TestAppDeleteSessionMutationFailureSkipsSync(t *testing.T) {
	t.Parallel()

	fixture := &sessionMutationOrderFixture{
		session:   &domain.PlaySession{ID: "session-1", GameID: "game-1"},
		deleteErr: errors.New("db delete fail"),
	}
	synced := make(chan string, 1)
	app := newSessionMutationOrderApp(t, fixture, synced)

	result := app.DeleteSession("session-1")
	if result.Success {
		t.Fatalf("expected mutation failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "セッション削除に失敗しました" {
		t.Fatalf("expected delete error message, got %#v", result.Error)
	}
	assertNoSessionSync(t, synced)
	if want := []string{"lookup", "mutation"}; !stringSliceEqual(fixture.calls, want) {
		t.Fatalf("expected lookup then mutation, got %v", fixture.calls)
	}
}

func TestAppDeleteSessionNilSessionMutatesWithoutSync(t *testing.T) {
	t.Parallel()

	fixture := &sessionMutationOrderFixture{session: nil}
	synced := make(chan string, 1)
	app := newSessionMutationOrderApp(t, fixture, synced)

	result := app.DeleteSession("missing-session")
	if !result.Success || !result.Data {
		t.Fatalf("expected success without sync for nil session, got %#v", result)
	}
	assertNoSessionSync(t, synced)
	if want := []string{"lookup", "mutation"}; !stringSliceEqual(fixture.calls, want) {
		t.Fatalf("expected lookup then mutation, got %v", fixture.calls)
	}
}

func TestAppUpdateSessionNameLookupMutationSyncOrder(t *testing.T) {
	t.Parallel()

	fixture := &sessionMutationOrderFixture{
		session: &domain.PlaySession{ID: "session-1", GameID: "game-1"},
	}
	synced := make(chan string, 1)
	app := newSessionMutationOrderApp(t, fixture, synced)

	result := app.UpdateSessionName("session-1", "Chapter 1")
	if !result.Success || !result.Data {
		t.Fatalf("expected ApiResult[bool] success, got %#v", result)
	}
	if got := waitSessionSync(t, synced); got != "game-1" {
		t.Fatalf("expected sync game-1, got %q", got)
	}
	// サービス内部の GetPlaySessionByID（TouchGameUpdatedAt 用）も同一ポート経由で記録される。
	if want := []string{"lookup", "lookup", "mutation", "sync"}; !stringSliceEqual(fixture.calls, want) {
		t.Fatalf("expected order %v, got %v", want, fixture.calls)
	}
}

func TestAppUpdateSessionRouteMutationFailureSkipsSync(t *testing.T) {
	t.Parallel()

	fixture := &sessionMutationOrderFixture{
		session:   &domain.PlaySession{ID: "session-1", GameID: "game-1"},
		updateErr: errors.New("db update fail"),
	}
	synced := make(chan string, 1)
	app := newSessionMutationOrderApp(t, fixture, synced)

	routeID := "route-1"
	result := app.UpdateSessionRoute("session-1", &routeID)
	if result.Success {
		t.Fatalf("expected mutation failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "セッションルート更新に失敗しました" {
		t.Fatalf("expected update error message, got %#v", result.Error)
	}
	assertNoSessionSync(t, synced)
	if want := []string{"lookup", "lookup", "mutation"}; !stringSliceEqual(fixture.calls, want) {
		t.Fatalf("expected lookup(s) then mutation, got %v", fixture.calls)
	}
}

func stringSliceEqual(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestAppListRoutesByGameConvertsServiceError(t *testing.T) {
	t.Parallel()

	app := &App{
		Logger:       newAdapterTestLogger(),
		RouteService: services.NewRouteService(noopAppRouteRepository{listErr: errors.New("db fail")}, newAdapterTestLogger()),
	}

	result := app.ListRoutesByGame("game-1")

	if result.Success {
		t.Fatalf("expected list failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "ルート取得に失敗しました" {
		t.Fatalf("expected converted service error, got %#v", result.Error)
	}
}

func TestAppSaveCredentialConvertsServiceError(t *testing.T) {
	t.Parallel()

	app := &App{
		Logger:            newAdapterTestLogger(),
		CredentialService: services.NewCredentialService(&adapterTestCredentialStore{}, newAdapterTestLogger()),
	}

	// 空の入力は CredentialService でバリデーションエラーになる
	result := app.SaveCredential("key", services.CredentialInput{})

	if result.Success {
		t.Fatalf("expected save failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "認証情報が不正です" {
		t.Fatalf("expected converted service error, got %#v", result.Error)
	}
}

func TestAppLoadCredentialConvertsServiceError(t *testing.T) {
	t.Parallel()

	app := &App{
		Logger:            newAdapterTestLogger(),
		CredentialService: services.NewCredentialService(&adapterTestCredentialStore{loadErr: errors.New("not found")}, newAdapterTestLogger()),
	}

	result := app.LoadCredential("key")

	if result.Success {
		t.Fatalf("expected load failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "認証情報取得に失敗しました" {
		t.Fatalf("expected converted service error, got %#v", result.Error)
	}
}

func TestDeleteCloudDataRejectsEmptyOrWildcardPath(t *testing.T) {
	t.Parallel()

	app := &App{}
	cases := []string{"", "   ", "*"}
	for _, tc := range cases {
		result := app.DeleteCloudData(tc)
		if result.Success {
			t.Fatalf("DeleteCloudData(%q) succeeded, want error", tc)
		}
		if result.Error == nil || result.Error.Detail == "" {
			t.Fatalf("DeleteCloudData(%q) should return error detail", tc)
		}
	}
}

func TestDeleteFileRejectsEmptyKey(t *testing.T) {
	t.Parallel()

	app := &App{}
	result := app.DeleteFile("   ")
	if result.Success {
		t.Fatal("DeleteFile succeeded, want error")
	}
	if result.Error == nil || result.Error.Detail == "" {
		t.Fatal("DeleteFile should return error detail")
	}
}

func TestNormalizeDeletePrefixSeparatesExactKeyAndChildren(t *testing.T) {
	t.Parallel()

	exact, children, ok := normalizeDeletePrefix("games/game-1/")
	if !ok {
		t.Fatal("expected prefix to be valid")
	}
	if exact != "games/game-1" {
		t.Fatalf("exact = %q, want %q", exact, "games/game-1")
	}
	if children != "games/game-1/" {
		t.Fatalf("children = %q, want %q", children, "games/game-1/")
	}
	if strings.HasPrefix("games/game-10/", children) {
		t.Fatal("child prefix should not match sibling game IDs")
	}
}

func TestNormalizeDeletePrefixRejectsEmptyOrWildcard(t *testing.T) {
	t.Parallel()

	for _, tc := range []string{"", " ", "*", "/"} {
		if exact, children, ok := normalizeDeletePrefix(tc); ok {
			t.Fatalf("normalizeDeletePrefix(%q) = %q, %q, true; want invalid", tc, exact, children)
		}
	}
}
