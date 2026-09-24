// 各サービスが依存する永続化リポジトリのインターフェース境界を定義する。
package services

import (
	"context"

	"CloudLaunch_Go/internal/domain"
)

// GameRepository は GameService が必要とする永続化境界を定義する。
type GameRepository interface {
	ListGames(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error)
	GetGameByID(ctx context.Context, gameID string) (*domain.Game, error)
	CreateGameWithInitialRoute(ctx context.Context, game domain.Game, initialRoute domain.Route) (*domain.Game, error)
	UpdateGame(ctx context.Context, game domain.Game) (*domain.Game, error)
	DeleteGameAndQueueMemoCleanup(ctx context.Context, gameID string) error
	ListPendingMemoCleanup(ctx context.Context) ([]string, error)
	ClearPendingMemoCleanup(ctx context.Context, gameID string) error
	RefreshGamePlayTimeFromSessions(ctx context.Context, gameID string) error
}

// SessionRepository は SessionService が必要とする永続化境界を定義する。
type SessionRepository interface {
	CreatePlaySessionAndRefreshGame(ctx context.Context, session domain.PlaySession) (*domain.PlaySession, error)
	ListPlaySessionsByGame(ctx context.Context, gameID string) ([]domain.PlaySession, error)
	GetPlaySessionByID(ctx context.Context, sessionID string) (*domain.PlaySession, error)
	DeletePlaySessionAndRefreshGame(ctx context.Context, sessionID string) (gameID string, err error)
	UpdatePlaySessionRoute(ctx context.Context, sessionID string, routeID *string) error
	UpdatePlaySessionName(ctx context.Context, sessionID string, sessionName string) error
	TouchGameUpdatedAt(ctx context.Context, gameID string) error
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
	// UpdateRouteOrders は gameID 配下のルートのみを対象に順序を一括更新する。
	// gameID を指定外の Route ID は無視する（更新行数 0）。
	UpdateRouteOrders(ctx context.Context, gameID string, items []domain.RouteOrderItem) error
	GetRouteStats(ctx context.Context, gameID string) ([]domain.RouteStat, error)
	GetGameByID(ctx context.Context, gameID string) (*domain.Game, error)
	UpdateGame(ctx context.Context, game domain.Game) (*domain.Game, error)
}

// ContentSyncRepository は ContentSyncService が必要とする永続化境界を定義する。
type ContentSyncRepository interface {
	GetGameByID(ctx context.Context, gameID string) (*domain.Game, error)
	ListPlaySessionsByGame(ctx context.Context, gameID string) ([]domain.PlaySession, error)
	ListRoutesByGame(ctx context.Context, gameID string) ([]domain.Route, error)
	GetLocalSaveTree(ctx context.Context, gameID string) (string, error)
	// SetLocalSyncState は localSyncHead と localSaveTree を単一トランザクションで更新する。
	SetLocalSyncState(ctx context.Context, gameID, syncHead, saveTree string) error
	// BeginPendingPush はリモート HEAD 更新前に pending Push を永続化する（UPSERT）。
	BeginPendingPush(ctx context.Context, pending domain.PendingPush) error
	// FinalizePendingPush は local baseline 更新と pending 削除を単一トランザクションで行う。
	FinalizePendingPush(ctx context.Context, gameID, syncHead, saveTree string) error
	// ClearPendingPush は baseline を変えずに pending だけ削除する（自動確定できない場合）。
	ClearPendingPush(ctx context.Context, gameID string) error
	ListPendingPushes(ctx context.Context) ([]domain.PendingPush, error)
	// BeginPullOperation はセーブ交換直前に PREPARED ジャーナルを永続化する。
	BeginPullOperation(ctx context.Context, op domain.PullOperation) error
	// ClearPullOperation は指定ジャーナルを削除する（backup 掃除後、または PREPARED 復旧後）。
	ClearPullOperation(ctx context.Context, operationID string) error
	ListPullOperations(ctx context.Context) ([]domain.PullOperation, error)
	// ApplyPullResult は v1 Pull のローカル反映（単一トランザクション）。
	// Route は置換せず、存在しない Route 参照は NULL に正規化する。
	// pullOperationID が非空なら同一 TX でジャーナルを APPLIED にする。
	ApplyPullResult(ctx context.Context, game domain.Game, sessions []domain.PlaySession, syncHead, saveTree, pullOperationID string) error
	// ApplyPullResultV2 は v2 Pull のローカル反映（単一トランザクション）。
	// Route を ID 保持で置換し、不正・重複・参照欠落はエラーで全体 rollback する。
	// pullOperationID が非空なら同一 TX でジャーナルを APPLIED にする。
	ApplyPullResultV2(ctx context.Context, game domain.Game, routes []domain.Route, sessions []domain.PlaySession, syncHead, saveTree, pullOperationID string) error
	GetSetting(ctx context.Context, key string) (string, error)
	UpsertSetting(ctx context.Context, key, value string) error
}

// MaintenanceRepository は MaintenanceService が必要とする永続化境界を定義する。
type MaintenanceRepository interface {
	ListGames(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error)
	ListPlaySessionsByGames(ctx context.Context, gameIDs []string) (map[string][]domain.PlaySession, error)
}

// ScreenshotRepository は ScreenshotService が必要とする永続化境界を定義する。
type ScreenshotRepository interface {
	GetGameByID(ctx context.Context, gameID string) (*domain.Game, error)
}

// ProcessIDResolver は実行ファイルパスから稼働中プロセスIDを引く境界。
type ProcessIDResolver interface {
	FindProcessIDsByExe(exePath string) ([]int, error)
}

// ProcessMonitorRepository は ProcessMonitorService が必要とする永続化境界を定義する。
type ProcessMonitorRepository interface {
	CreatePlaySessionAndRefreshGame(ctx context.Context, session domain.PlaySession) (*domain.PlaySession, error)
	GetGameByID(ctx context.Context, gameID string) (*domain.Game, error)
	UpdateGame(ctx context.Context, game domain.Game) (*domain.Game, error)
	ListGames(ctx context.Context, searchText string, filter domain.PlayStatus, sortBy string, sortDirection string) ([]domain.Game, error)
}
