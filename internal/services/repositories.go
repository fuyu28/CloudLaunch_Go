package services

import (
	"context"
	"time"

	"CloudLaunch_Go/internal/models"
)

// GameRepository defines the persistence boundary required by GameService.
type GameRepository interface {
	ListGames(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error)
	GetGameByID(ctx context.Context, gameID string) (*models.Game, error)
	CreateGame(ctx context.Context, game models.Game) (*models.Game, error)
	UpdateGame(ctx context.Context, game models.Game) (*models.Game, error)
	DeleteGame(ctx context.Context, gameID string) error
}

// SessionRepository defines the persistence boundary required by SessionService.
type SessionRepository interface {
	CreatePlaySession(ctx context.Context, session models.PlaySession) (*models.PlaySession, error)
	ListPlaySessionsByGame(ctx context.Context, gameID string) ([]models.PlaySession, error)
	GetPlaySessionByID(ctx context.Context, sessionID string) (*models.PlaySession, error)
	DeletePlaySession(ctx context.Context, sessionID string) error
	TouchGameUpdatedAt(ctx context.Context, gameID string) error
	SumPlaySessionDurationsByGame(ctx context.Context, gameID string) (int64, error)
	UpdateGameTotalPlayTime(ctx context.Context, gameID string, totalPlayTime int64) error
	UpdateGameTotalPlayTimeWithLastPlayed(ctx context.Context, gameID string, totalPlayTime int64, playedAt time.Time) error
}

// MemoRepository defines the persistence boundary required by MemoService.
type MemoRepository interface {
	CreateMemo(ctx context.Context, memo models.Memo) (*models.Memo, error)
	UpdateMemo(ctx context.Context, memo models.Memo) (*models.Memo, error)
	GetMemoByID(ctx context.Context, memoID string) (*models.Memo, error)
	FindMemoByTitle(ctx context.Context, gameID string, title string) (*models.Memo, error)
	ListMemosByGame(ctx context.Context, gameID string) ([]models.Memo, error)
	ListAllMemos(ctx context.Context) ([]models.Memo, error)
	DeleteMemo(ctx context.Context, memoID string) error
}

// PlayRouteRepository defines the persistence boundary required by PlayRouteService.
type PlayRouteRepository interface {
	CreatePlayRoute(ctx context.Context, route models.PlayRoute) (*models.PlayRoute, error)
	ListPlayRoutesByGame(ctx context.Context, gameID string) ([]models.PlayRoute, error)
	DeletePlayRoute(ctx context.Context, routeID string) error
}

// CloudSyncRepository defines the persistence boundary required by CloudSyncService.
type CloudSyncRepository interface {
	GetGameByID(ctx context.Context, gameID string) (*models.Game, error)
	ListGames(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error)
	ListPlaySessionsByGame(ctx context.Context, gameID string) ([]models.PlaySession, error)
	ListPlayRoutesByGame(ctx context.Context, gameID string) ([]models.PlayRoute, error)
	UpsertGameSync(ctx context.Context, game models.Game) error
	DeletePlaySessionsByGame(ctx context.Context, gameID string) error
	DeletePlayRoutesByGame(ctx context.Context, gameID string) error
	UpsertPlaySessionSync(ctx context.Context, session models.PlaySession) error
	UpsertPlayRouteSync(ctx context.Context, route models.PlayRoute) error
	SumPlaySessionDurationsByGame(ctx context.Context, gameID string) (int64, error)
	UpdateGameTotalPlayTime(ctx context.Context, gameID string, totalPlayTime int64) error
	UpdateGameTotalPlayTimeWithLastPlayed(ctx context.Context, gameID string, totalPlayTime int64, playedAt time.Time) error
}

// ScreenshotRepository defines the persistence boundary required by ScreenshotService.
type ScreenshotRepository interface {
	GetGameByID(ctx context.Context, gameID string) (*models.Game, error)
}

// ProcessMonitorRepository defines the persistence boundary required by ProcessMonitorService.
type ProcessMonitorRepository interface {
	CreatePlaySession(ctx context.Context, session models.PlaySession) (*models.PlaySession, error)
	GetGameByID(ctx context.Context, gameID string) (*models.Game, error)
	UpdateGame(ctx context.Context, game models.Game) (*models.Game, error)
	ListGames(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error)
}
