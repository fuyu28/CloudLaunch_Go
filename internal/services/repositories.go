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
	CreateChapter(ctx context.Context, chapter models.Chapter) (*models.Chapter, error)
}

// SessionRepository defines the persistence boundary required by SessionService.
type SessionRepository interface {
	CreatePlaySession(ctx context.Context, session models.PlaySession) (*models.PlaySession, error)
	ListPlaySessionsByGame(ctx context.Context, gameID string) ([]models.PlaySession, error)
	GetPlaySessionByID(ctx context.Context, sessionID string) (*models.PlaySession, error)
	DeletePlaySession(ctx context.Context, sessionID string) error
	UpdatePlaySessionChapter(ctx context.Context, sessionID string, chapterID *string) error
	UpdatePlaySessionName(ctx context.Context, sessionID string, sessionName string) error
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

// ChapterRepository defines the persistence boundary required by ChapterService.
type ChapterRepository interface {
	ListChaptersByGame(ctx context.Context, gameID string) ([]models.Chapter, error)
	CreateChapter(ctx context.Context, chapter models.Chapter) (*models.Chapter, error)
	GetChapterByID(ctx context.Context, chapterID string) (*models.Chapter, error)
	UpdateChapter(ctx context.Context, chapter models.Chapter) (*models.Chapter, error)
	DeleteChapter(ctx context.Context, chapterID string) error
	UpdateChapterOrder(ctx context.Context, chapterID string, order int64) error
	GetChapterStats(ctx context.Context, gameID string) ([]models.ChapterStat, error)
	GetGameByID(ctx context.Context, gameID string) (*models.Game, error)
	UpdateGame(ctx context.Context, game models.Game) (*models.Game, error)
}

// UploadRepository defines the persistence boundary required by UploadService.
type UploadRepository interface {
	CreateUpload(ctx context.Context, upload models.Upload) (*models.Upload, error)
	ListUploadsByGame(ctx context.Context, gameID string) ([]models.Upload, error)
}

// CloudSyncRepository defines the persistence boundary required by CloudSyncService.
type CloudSyncRepository interface {
	GetGameByID(ctx context.Context, gameID string) (*models.Game, error)
	ListGames(ctx context.Context, searchText string, filter models.PlayStatus, sortBy string, sortDirection string) ([]models.Game, error)
	ListPlaySessionsByGame(ctx context.Context, gameID string) ([]models.PlaySession, error)
	UpsertGameSync(ctx context.Context, game models.Game) error
	DeletePlaySessionsByGame(ctx context.Context, gameID string) error
	UpsertPlaySessionSync(ctx context.Context, session models.PlaySession) error
	SumPlaySessionDurationsByGame(ctx context.Context, gameID string) (int64, error)
	UpdateGameTotalPlayTime(ctx context.Context, gameID string, totalPlayTime int64) error
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
