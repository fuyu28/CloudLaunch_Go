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

func (r noopAppGameRepository) CreateGame(ctx context.Context, game domain.Game) (*domain.Game, error) {
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

func (r noopAppGameRepository) DeleteGame(ctx context.Context, gameID string) error {
	return r.deleteErr
}

func (r noopAppGameRepository) CreateRoute(ctx context.Context, route domain.Route) (*domain.Route, error) {
	return &route, nil
}

type noopAppSessionRepository struct {
	createErr error
	getErr    error
	deleteErr error
	updateErr error
	session   *domain.PlaySession
}

func (r noopAppSessionRepository) CreatePlaySession(ctx context.Context, session domain.PlaySession) (*domain.PlaySession, error) {
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
func (r noopAppSessionRepository) DeletePlaySession(ctx context.Context, sessionID string) error {
	return r.deleteErr
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
func (r noopAppSessionRepository) SumPlaySessionDurationsByGame(ctx context.Context, gameID string) (int64, error) {
	return 0, nil
}
func (r noopAppSessionRepository) UpdateGameTotalPlayTime(ctx context.Context, gameID string, totalPlayTime int64) error {
	return nil
}
func (r noopAppSessionRepository) UpdateGameTotalPlayTimeWithLastPlayed(ctx context.Context, gameID string, totalPlayTime int64, playedAt time.Time) error {
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

	app := &App{
		Logger:         newAdapterTestLogger(),
		SessionService: services.NewSessionService(noopAppSessionRepository{deleteErr: errors.New("db fail")}, newAdapterTestLogger()),
	}

	result := app.DeleteSession("session-1")

	if result.Success {
		t.Fatalf("expected delete failure, got success: %#v", result)
	}
	if result.Error == nil || result.Error.Message != "セッション削除に失敗しました" {
		t.Fatalf("expected converted service error, got %#v", result.Error)
	}
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
