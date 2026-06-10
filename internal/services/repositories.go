package services

import (
	"context"
	"time"

	"CloudLaunch_Go/internal/domain"
)

// GameRepository は GameService が必要とする永続化境界を定義する。
type GameRepository interface {
	ListGames(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error)
	GetGameByID(ctx context.Context, gameID string) (*domain.Game, error)
	CreateGame(ctx context.Context, game domain.Game) (*domain.Game, error)
	UpdateGame(ctx context.Context, game domain.Game) (*domain.Game, error)
	DeleteGame(ctx context.Context, gameID string) error
	CreateRoute(ctx context.Context, route domain.Route) (*domain.Route, error)
}

// SessionRepository は SessionService が必要とする永続化境界を定義する。
type SessionRepository interface {
	CreatePlaySession(ctx context.Context, session domain.PlaySession) (*domain.PlaySession, error)
	ListPlaySessionsByGame(ctx context.Context, gameID string) ([]domain.PlaySession, error)
	GetPlaySessionByID(ctx context.Context, sessionID string) (*domain.PlaySession, error)
	DeletePlaySession(ctx context.Context, sessionID string) error
	UpdatePlaySessionRoute(ctx context.Context, sessionID string, routeID *string) error
	UpdatePlaySessionName(ctx context.Context, sessionID string, sessionName string) error
	TouchGameUpdatedAt(ctx context.Context, gameID string) error
	SumPlaySessionDurationsByGame(ctx context.Context, gameID string) (int64, error)
	UpdateGameTotalPlayTime(ctx context.Context, gameID string, totalPlayTime int64) error
	UpdateGameTotalPlayTimeWithLastPlayed(ctx context.Context, gameID string, totalPlayTime int64, playedAt time.Time) error
}

// MemoRepository は MemoService が必要とする永続化境界を定義する。
type MemoRepository interface {
	CreateMemo(ctx context.Context, memo domain.Memo) (*domain.Memo, error)
	UpdateMemo(ctx context.Context, memo domain.Memo) (*domain.Memo, error)
	GetMemoByID(ctx context.Context, memoID string) (*domain.Memo, error)
	FindMemoByTitle(ctx context.Context, gameID string, title string) (*domain.Memo, error)
	ListMemosByGame(ctx context.Context, gameID string) ([]domain.Memo, error)
	ListAllMemos(ctx context.Context) ([]domain.Memo, error)
	DeleteMemo(ctx context.Context, memoID string) error
}

// RouteRepository は RouteService が必要とする永続化境界を定義する。
type RouteRepository interface {
	ListRoutesByGame(ctx context.Context, gameID string) ([]domain.Route, error)
	CreateRoute(ctx context.Context, route domain.Route) (*domain.Route, error)
	GetRouteByID(ctx context.Context, routeID string) (*domain.Route, error)
	UpdateRoute(ctx context.Context, route domain.Route) (*domain.Route, error)
	DeleteRoute(ctx context.Context, routeID string) error
	UpdateRouteOrder(ctx context.Context, routeID string, order int64) error
	GetRouteStats(ctx context.Context, gameID string) ([]domain.RouteStat, error)
	GetGameByID(ctx context.Context, gameID string) (*domain.Game, error)
	UpdateGame(ctx context.Context, game domain.Game) (*domain.Game, error)
}

// ContentSyncRepository は ContentSyncService が必要とする永続化境界を定義する。
type ContentSyncRepository interface {
	GetGameByID(ctx context.Context, gameID string) (*domain.Game, error)
	ListPlaySessionsByGame(ctx context.Context, gameID string) ([]domain.PlaySession, error)
	SetLocalSyncHead(ctx context.Context, gameID, hash string) error
	UpsertGameSync(ctx context.Context, game domain.Game) error
	DeletePlaySessionsByGame(ctx context.Context, gameID string) error
	UpsertPlaySessionSync(ctx context.Context, session domain.PlaySession) error
	GetSetting(ctx context.Context, key string) (string, error)
	UpsertSetting(ctx context.Context, key, value string) error
}

// ScreenshotRepository は ScreenshotService が必要とする永続化境界を定義する。
type ScreenshotRepository interface {
	GetGameByID(ctx context.Context, gameID string) (*domain.Game, error)
}

// ProcessMonitorRepository は ProcessMonitorService が必要とする永続化境界を定義する。
type ProcessMonitorRepository interface {
	CreatePlaySession(ctx context.Context, session domain.PlaySession) (*domain.PlaySession, error)
	GetGameByID(ctx context.Context, gameID string) (*domain.Game, error)
	UpdateGame(ctx context.Context, game domain.Game) (*domain.Game, error)
	ListGames(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error)
}
