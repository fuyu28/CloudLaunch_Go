package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"CloudLaunch_Go/internal/domain"
)

type fakeRouteRepository struct {
	listRoutesByGameFn  func(ctx context.Context, gameID string) ([]domain.Route, error)
	createRouteFn       func(ctx context.Context, route domain.Route) (*domain.Route, error)
	getRouteByIDFn      func(ctx context.Context, routeID string) (*domain.Route, error)
	updateRouteFn       func(ctx context.Context, route domain.Route) (*domain.Route, error)
	deleteRouteFn       func(ctx context.Context, routeID string) error
	updateRouteOrderFn  func(ctx context.Context, routeID string, order int64) error
	updateRouteOrdersFn func(ctx context.Context, items []domain.RouteOrderItem) error
	getRouteStatsFn     func(ctx context.Context, gameID string) ([]domain.RouteStat, error)
	getGameByIDFn       func(ctx context.Context, gameID string) (*domain.Game, error)
	updateGameFn        func(ctx context.Context, game domain.Game) (*domain.Game, error)
}

func (r fakeRouteRepository) ListRoutesByGame(ctx context.Context, gameID string) ([]domain.Route, error) {
	return r.listRoutesByGameFn(ctx, gameID)
}

func (r fakeRouteRepository) CreateRoute(ctx context.Context, route domain.Route) (*domain.Route, error) {
	return r.createRouteFn(ctx, route)
}

func (r fakeRouteRepository) GetRouteByID(ctx context.Context, routeID string) (*domain.Route, error) {
	return r.getRouteByIDFn(ctx, routeID)
}

func (r fakeRouteRepository) UpdateRoute(ctx context.Context, route domain.Route) (*domain.Route, error) {
	return r.updateRouteFn(ctx, route)
}

func (r fakeRouteRepository) DeleteRoute(ctx context.Context, routeID string) error {
	return r.deleteRouteFn(ctx, routeID)
}

func (r fakeRouteRepository) UpdateRouteOrder(ctx context.Context, routeID string, order int64) error {
	return r.updateRouteOrderFn(ctx, routeID, order)
}

func (r fakeRouteRepository) UpdateRouteOrders(ctx context.Context, items []domain.RouteOrderItem) error {
	return r.updateRouteOrdersFn(ctx, items)
}

func (r fakeRouteRepository) GetRouteStats(ctx context.Context, gameID string) ([]domain.RouteStat, error) {
	return r.getRouteStatsFn(ctx, gameID)
}

func (r fakeRouteRepository) GetGameByID(ctx context.Context, gameID string) (*domain.Game, error) {
	return r.getGameByIDFn(ctx, gameID)
}

func (r fakeRouteRepository) UpdateGame(ctx context.Context, game domain.Game) (*domain.Game, error) {
	return r.updateGameFn(ctx, game)
}

func newFullFakeRouteRepository() fakeRouteRepository {
	return fakeRouteRepository{
		listRoutesByGameFn:  func(ctx context.Context, gameID string) ([]domain.Route, error) { return nil, nil },
		createRouteFn:       func(ctx context.Context, route domain.Route) (*domain.Route, error) { return &route, nil },
		getRouteByIDFn:      func(ctx context.Context, routeID string) (*domain.Route, error) { return nil, nil },
		updateRouteFn:       func(ctx context.Context, route domain.Route) (*domain.Route, error) { return &route, nil },
		deleteRouteFn:       func(ctx context.Context, routeID string) error { return nil },
		updateRouteOrderFn:  func(ctx context.Context, routeID string, order int64) error { return nil },
		updateRouteOrdersFn: func(ctx context.Context, items []domain.RouteOrderItem) error { return nil },
		getRouteStatsFn:     func(ctx context.Context, gameID string) ([]domain.RouteStat, error) { return nil, nil },
		getGameByIDFn:       func(ctx context.Context, gameID string) (*domain.Game, error) { return nil, nil },
		updateGameFn:        func(ctx context.Context, game domain.Game) (*domain.Game, error) { return &game, nil },
	}
}

func TestRouteServiceSetCurrentRouteUsesRepositoryBoundary(t *testing.T) {
	t.Parallel()

	repo := newFullFakeRouteRepository()
	repo.getGameByIDFn = func(ctx context.Context, gameID string) (*domain.Game, error) {
		return &domain.Game{ID: gameID, Title: "Game"}, nil
	}
	repo.updateGameFn = func(ctx context.Context, game domain.Game) (*domain.Game, error) {
		return &game, nil
	}
	service := NewRouteService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := service.SetCurrentRoute(context.Background(), "game-1", "route-1"); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestRouteServiceListCreateUpdateDeleteUseRepositoryBoundary(t *testing.T) {
	t.Parallel()

	route := domain.Route{ID: "route-1", Name: "Route 1", Order: 1, GameID: "game-1"}
	repo := newFullFakeRouteRepository()
	repo.listRoutesByGameFn = func(ctx context.Context, gameID string) ([]domain.Route, error) {
		return []domain.Route{route}, nil
	}
	repo.createRouteFn = func(ctx context.Context, created domain.Route) (*domain.Route, error) {
		created.ID = "route-1"
		return &created, nil
	}
	repo.getRouteByIDFn = func(ctx context.Context, routeID string) (*domain.Route, error) {
		return &route, nil
	}
	repo.updateRouteFn = func(ctx context.Context, updated domain.Route) (*domain.Route, error) {
		return &updated, nil
	}
	service := NewRouteService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	listed, err := service.ListRoutesByGame(context.Background(), "game-1")
	if err != nil || len(listed) != 1 || listed[0].ID != "route-1" {
		t.Fatalf("unexpected listed routes: %#v", listed)
	}

	created, err := service.CreateRoute(context.Background(), RouteInput{Name: " Route 1 ", Order: 1, GameID: "game-1"})
	if err != nil || created == nil || created.Name != "Route 1" {
		t.Fatalf("unexpected create result: %#v", created)
	}

	updated, err := service.UpdateRoute(context.Background(), "route-1", RouteUpdateInput{Name: " Route X ", Order: 2})
	if err != nil || updated == nil || updated.Name != "Route X" || updated.Order != 2 {
		t.Fatalf("unexpected update result: %#v", updated)
	}

	if err := service.DeleteRoute(context.Background(), "route-1"); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
}

func TestRouteServiceGetRouteStatsHandlesRepositoryError(t *testing.T) {
	t.Parallel()

	repo := newFullFakeRouteRepository()
	repo.getRouteStatsFn = func(ctx context.Context, gameID string) ([]domain.RouteStat, error) {
		return nil, errors.New("db down")
	}
	service := NewRouteService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.GetRouteStats(context.Background(), "game-1")
	if err == nil {
		t.Fatalf("expected failure")
	}
}

func TestRouteServiceSetCurrentRouteReturnsNotFoundWhenGameMissing(t *testing.T) {
	t.Parallel()

	repo := newFullFakeRouteRepository()
	service := NewRouteService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := service.SetCurrentRoute(context.Background(), "game-1", "route-1"); err == nil {
		t.Fatalf("expected missing game to fail")
	}
}

func TestRouteServiceUpdateRouteOrdersRejectsNegativeOrder(t *testing.T) {
	t.Parallel()

	repo := newFullFakeRouteRepository()
	service := NewRouteService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := service.UpdateRouteOrders(context.Background(), "game-1", []RouteOrderUpdate{{ID: "route-1", Order: -1}}); err == nil {
		t.Fatalf("expected invalid order to fail")
	}
}
