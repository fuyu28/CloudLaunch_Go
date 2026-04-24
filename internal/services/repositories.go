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
