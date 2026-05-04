package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"CloudLaunch_Go/internal/models"
)

type fakePlayRouteRepository struct {
	createdRoute  *models.PlayRoute
	listedRoutes  []models.PlayRoute
	deleteRouteID string
	listErr       error
	createErr     error
	deleteErr     error
}

func (repository *fakePlayRouteRepository) CreatePlayRoute(ctx context.Context, route models.PlayRoute) (*models.PlayRoute, error) {
	if repository.createErr != nil {
		return nil, repository.createErr
	}
	route.ID = "route-1"
	repository.createdRoute = &route
	return &route, nil
}

func (repository *fakePlayRouteRepository) ListPlayRoutesByGame(ctx context.Context, gameID string) ([]models.PlayRoute, error) {
	if repository.listErr != nil {
		return nil, repository.listErr
	}
	return repository.listedRoutes, nil
}

func (repository *fakePlayRouteRepository) DeletePlayRoute(ctx context.Context, routeID string) error {
	repository.deleteRouteID = routeID
	return repository.deleteErr
}

func TestPlayRouteServiceCreatePlayRouteUsesRepositoryBoundary(t *testing.T) {
	t.Parallel()

	repository := &fakePlayRouteRepository{}
	service := NewPlayRouteService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	created, err := service.CreatePlayRoute(context.Background(), PlayRouteInput{
		GameID:    " game-1 ",
		Name:      " heroine route ",
		SortOrder: 2,
	})

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if created == nil || created.ID != "route-1" {
		t.Fatalf("expected created route, got %#v", created)
	}
	if repository.createdRoute == nil || repository.createdRoute.GameID != "game-1" || repository.createdRoute.Name != "heroine route" || repository.createdRoute.SortOrder != 2 {
		t.Fatalf("expected trimmed route to be stored, got %#v", repository.createdRoute)
	}
}

func TestPlayRouteServiceListAndDeleteUseRepositoryBoundary(t *testing.T) {
	t.Parallel()

	repository := &fakePlayRouteRepository{
		listedRoutes: []models.PlayRoute{{ID: "route-1", GameID: "game-1", Name: "Common", SortOrder: 0}},
	}
	service := NewPlayRouteService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))

	routes, err := service.ListPlayRoutesByGame(context.Background(), " game-1 ")
	if err != nil || len(routes) != 1 || routes[0].ID != "route-1" {
		t.Fatalf("unexpected list result: %#v, err=%v", routes, err)
	}

	if err := service.DeletePlayRoute(context.Background(), " route-1 "); err != nil {
		t.Fatalf("expected delete success, got %v", err)
	}
	if repository.deleteRouteID != "route-1" {
		t.Fatalf("expected delete target to be trimmed")
	}
}

func TestPlayRouteServiceCreatePlayRouteRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	service := NewPlayRouteService(&fakePlayRouteRepository{}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := service.CreatePlayRoute(context.Background(), PlayRouteInput{
		GameID:    "game-1",
		Name:      "",
		SortOrder: 0,
	})
	if err == nil {
		t.Fatal("expected invalid input to fail")
	}
}

func TestPlayRouteServiceDeletePlayRouteReturnsRepositoryError(t *testing.T) {
	t.Parallel()

	service := NewPlayRouteService(&fakePlayRouteRepository{deleteErr: errors.New("db down")}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := service.DeletePlayRoute(context.Background(), "route-1"); err == nil {
		t.Fatal("expected delete error")
	}
}
