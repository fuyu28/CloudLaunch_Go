// @fileoverview データベース操作の基本CRUDを提供する。
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	sqlcgen "CloudLaunch_Go/internal/db/sqlc"
	"CloudLaunch_Go/internal/models"
)

const derivedPlayStatusSQL = `
CASE
  WHEN clearedAt IS NOT NULL THEN 'cleared'
  WHEN lastPlayed IS NOT NULL OR totalPlayTime > 0 THEN 'playing'
  ELSE 'unplayed'
END
`

// Repository は主要テーブルへのCRUDを提供する。
type Repository struct {
	connection *sql.DB
	queries    *sqlcgen.Queries
}

// NewRepository は Repository を初期化する。
func NewRepository(connection *sql.DB) *Repository {
	return &Repository{
		connection: connection,
		queries:    sqlcgen.New(connection),
	}
}

// GetGameByID はID指定でゲームを取得する。
func (repository *Repository) GetGameByID(ctx context.Context, gameID string) (*models.Game, error) {
	gameRow, error := repository.queries.GetGameByID(ctx, gameID)
	if error == sql.ErrNoRows {
		return nil, nil
	}
	if error != nil {
		return nil, error
	}
	return modelGameFromSQLC(gameRow), nil
}

// GetGameByExePath は実行ファイルパスに一致するゲームを取得する。
func (repository *Repository) GetGameByExePath(ctx context.Context, exePath string) (*models.Game, error) {
	trimmed := strings.TrimSpace(exePath)
	if trimmed == "" {
		return nil, errors.New("exePath is empty")
	}
	gameRow, error := repository.queries.GetGameByExePath(ctx, trimmed)
	if error == sql.ErrNoRows {
		return nil, nil
	}
	if error != nil {
		return nil, error
	}
	return modelGameFromSQLC(gameRow), nil
}

// ListGames は検索・フィルタ・ソート付きでゲームを取得する。
func (repository *Repository) ListGames(
	ctx context.Context,
	searchText string,
	filter models.PlayStatus,
	sortBy string,
	sortDirection string,
) (games []models.Game, err error) {
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
		       localSaveHash, localSaveHashUpdatedAt,
		       totalPlayTime, lastPlayed, clearedAt
		FROM "Game"
	`)

	whereClauses := make([]string, 0, 2)
	args := make([]any, 0, 2)
	if searchText != "" {
		whereClauses = append(whereClauses, "(title LIKE ? OR publisher LIKE ?)")
		pattern := fmt.Sprintf("%%%s%%", searchText)
		args = append(args, pattern, pattern)
	}
	if filter != "" {
		whereClauses = append(whereClauses, derivedPlayStatusSQL+" = ?")
		args = append(args, filter)
	}
	if len(whereClauses) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(whereClauses, " AND "))
	}

	orderBy := normalizeSortColumn(sortBy)
	direction := normalizeSortDirection(sortDirection)
	_, _ = fmt.Fprintf(&queryBuilder, " ORDER BY %s %s", orderBy, direction)

	rows, err := repository.connection.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	games = make([]models.Game, 0)
	for rows.Next() {
		game, err := scanGame(rows)
		if err != nil {
			return nil, err
		}
		games = append(games, *game)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return games, nil
}

// CreateGame はゲームを作成して返す。
func (repository *Repository) CreateGame(ctx context.Context, game models.Game) (*models.Game, error) {
	created, error := repository.queries.CreateGame(ctx, sqlcgen.CreateGameParams{
		Title:                  game.Title,
		Publisher:              game.Publisher,
		ImagePath:              toNullString(game.ImagePath),
		ExePath:                game.ExePath,
		SaveFolderPath:         toNullString(game.SaveFolderPath),
		LocalSaveHash:          toNullString(game.LocalSaveHash),
		LocalSaveHashUpdatedAt: toNullTime(game.LocalSaveHashUpdatedAt),
		TotalPlayTime:          game.TotalPlayTime,
		LastPlayed:             toNullTime(game.LastPlayed),
		ClearedAt:              toNullTime(game.ClearedAt),
	})
	if error != nil {
		return nil, error
	}
	return modelGameFromSQLC(created), nil
}

// UpdateGame はゲームを更新して返す。
func (repository *Repository) UpdateGame(ctx context.Context, game models.Game) (*models.Game, error) {
	updated, error := repository.queries.UpdateGame(ctx, sqlcgen.UpdateGameParams{
		Title:                  game.Title,
		Publisher:              game.Publisher,
		ImagePath:              toNullString(game.ImagePath),
		ExePath:                game.ExePath,
		SaveFolderPath:         toNullString(game.SaveFolderPath),
		LocalSaveHash:          toNullString(game.LocalSaveHash),
		LocalSaveHashUpdatedAt: toNullTime(game.LocalSaveHashUpdatedAt),
		TotalPlayTime:          game.TotalPlayTime,
		LastPlayed:             toNullTime(game.LastPlayed),
		ClearedAt:              toNullTime(game.ClearedAt),
		ID:                     game.ID,
	})
	if error != nil {
		return nil, error
	}
	return modelGameFromSQLC(updated), nil
}

// UpsertGameSync はID指定でゲームを追加/更新する。
func (repository *Repository) UpsertGameSync(ctx context.Context, game models.Game) error {
	return repository.queries.UpsertGameSync(ctx, sqlcgen.UpsertGameSyncParams{
		ID:                     game.ID,
		Title:                  game.Title,
		Publisher:              game.Publisher,
		ImagePath:              toNullString(game.ImagePath),
		ExePath:                game.ExePath,
		SaveFolderPath:         toNullString(game.SaveFolderPath),
		CreatedAt:              game.CreatedAt,
		UpdatedAt:              game.UpdatedAt,
		LocalSaveHash:          toNullString(game.LocalSaveHash),
		LocalSaveHashUpdatedAt: toNullTime(game.LocalSaveHashUpdatedAt),
		TotalPlayTime:          game.TotalPlayTime,
		LastPlayed:             toNullTime(game.LastPlayed),
		ClearedAt:              toNullTime(game.ClearedAt),
	})
}

// TouchGameUpdatedAt はゲームのupdatedAtを現在時刻に更新する。
func (repository *Repository) TouchGameUpdatedAt(ctx context.Context, gameID string) error {
	return repository.queries.TouchGameUpdatedAt(ctx, gameID)
}

// DeleteGame はゲームを削除する。
func (repository *Repository) DeleteGame(ctx context.Context, gameID string) error {
	return repository.queries.DeleteGame(ctx, gameID)
}

// CreatePlayRoute はプレイルートを作成して返す。
func (repository *Repository) CreatePlayRoute(ctx context.Context, route models.PlayRoute) (*models.PlayRoute, error) {
	created, err := repository.queries.CreatePlayRoute(ctx, sqlcgen.CreatePlayRouteParams{
		GameID:    route.GameID,
		Name:      route.Name,
		SortOrder: int64(route.SortOrder),
	})
	if err != nil {
		return nil, err
	}
	return modelPlayRouteFromSQLC(created), nil
}

// ListPlayRoutesByGame はゲームID配下のプレイルート一覧を取得する。
func (repository *Repository) ListPlayRoutesByGame(ctx context.Context, gameID string) (routes []models.PlayRoute, err error) {
	items, err := repository.queries.ListPlayRoutesByGame(ctx, gameID)
	if err != nil {
		return nil, err
	}
	routes = make([]models.PlayRoute, 0, len(items))
	for _, item := range items {
		routes = append(routes, *modelPlayRouteFromSQLC(item))
	}
	return routes, nil
}

// DeletePlayRoute はプレイルートを削除する。
func (repository *Repository) DeletePlayRoute(ctx context.Context, routeID string) error {
	return repository.queries.DeletePlayRoute(ctx, routeID)
}

// DeletePlayRoutesByGame はゲームID配下のプレイルートを削除する。
func (repository *Repository) DeletePlayRoutesByGame(ctx context.Context, gameID string) error {
	return repository.queries.DeletePlayRoutesByGame(ctx, gameID)
}

// CreatePlaySession はプレイセッションを作成して返す。
func (repository *Repository) CreatePlaySession(ctx context.Context, session models.PlaySession) (*models.PlaySession, error) {
	created, error := repository.queries.CreatePlaySession(ctx, sqlcgen.CreatePlaySessionParams{
		GameID:      session.GameID,
		PlayRouteID: toNullString(session.PlayRouteID),
		PlayedAt:    session.PlayedAt,
		Duration:    session.Duration,
	})
	if error != nil {
		return nil, error
	}
	return modelPlaySessionFromSQLC(created), nil
}

// GetPlaySessionByID はID指定でセッションを取得する。
func (repository *Repository) GetPlaySessionByID(ctx context.Context, sessionID string) (*models.PlaySession, error) {
	sessionRow, error := repository.queries.GetPlaySessionByID(ctx, sessionID)
	if error == sql.ErrNoRows {
		return nil, nil
	}
	if error != nil {
		return nil, error
	}
	return modelPlaySessionFromSQLC(sessionRow), nil
}

// ListPlaySessionsByGame はゲームIDでセッション一覧を取得する。
func (repository *Repository) ListPlaySessionsByGame(ctx context.Context, gameID string) (sessions []models.PlaySession, err error) {
	items, err := repository.queries.ListPlaySessionsByGame(ctx, gameID)
	if err != nil {
		return nil, err
	}
	sessions = make([]models.PlaySession, 0, len(items))
	for _, item := range items {
		sessions = append(sessions, *modelPlaySessionFromSQLC(item))
	}
	return sessions, nil
}

// DeletePlaySession はセッションを削除する。
func (repository *Repository) DeletePlaySession(ctx context.Context, sessionID string) error {
	return repository.queries.DeletePlaySession(ctx, sessionID)
}

// DeletePlaySessionsByGame はゲームID配下のセッションを削除する。
func (repository *Repository) DeletePlaySessionsByGame(ctx context.Context, gameID string) error {
	return repository.queries.DeletePlaySessionsByGame(ctx, gameID)
}

// SumPlaySessionDurationsByGame はゲームIDのセッション合計時間を取得する。
func (repository *Repository) SumPlaySessionDurationsByGame(ctx context.Context, gameID string) (int64, error) {
	row := repository.connection.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(duration), 0) FROM "PlaySession" WHERE gameId = ?
	`, gameID)
	var total int64
	if err := row.Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// UpdateGameTotalPlayTime はゲームの総プレイ時間のみ更新する。
func (repository *Repository) UpdateGameTotalPlayTime(ctx context.Context, gameID string, totalPlayTime int64) error {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "Game"
		SET totalPlayTime = ?,
		    lastPlayed = (
		      SELECT MAX(datetime(playedAt, '+' || duration || ' seconds'))
		      FROM "PlaySession"
		      WHERE gameId = ?
		    )
		WHERE id = ?
	`, totalPlayTime, gameID, gameID)
	return error
}

// UpdateGameTotalPlayTimeWithLastPlayed は総プレイ時間と最終プレイ日時を更新する。
func (repository *Repository) UpdateGameTotalPlayTimeWithLastPlayed(
	ctx context.Context,
	gameID string,
	totalPlayTime int64,
	lastPlayed time.Time,
) error {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "Game"
		SET totalPlayTime = ?,
		    lastPlayed = CASE
		      WHEN lastPlayed IS NULL OR lastPlayed < ? THEN ?
		      ELSE lastPlayed
		    END
		WHERE id = ?
	`, totalPlayTime, lastPlayed, lastPlayed, gameID)
	return error
}

// UpsertPlaySessionSync はID指定でセッションを追加/更新する。
func (repository *Repository) UpsertPlaySessionSync(ctx context.Context, session models.PlaySession) error {
	return repository.queries.UpsertPlaySessionSync(ctx, sqlcgen.UpsertPlaySessionSyncParams{
		ID:          session.ID,
		GameID:      session.GameID,
		PlayRouteID: toNullString(session.PlayRouteID),
		PlayedAt:    session.PlayedAt,
		Duration:    session.Duration,
		UpdatedAt:   session.UpdatedAt,
	})
}

// UpsertPlayRouteSync はID指定でプレイルートを追加/更新する。
func (repository *Repository) UpsertPlayRouteSync(ctx context.Context, route models.PlayRoute) error {
	return repository.queries.UpsertPlayRouteSync(ctx, sqlcgen.UpsertPlayRouteSyncParams{
		ID:        route.ID,
		GameID:    route.GameID,
		Name:      route.Name,
		SortOrder: int64(route.SortOrder),
		CreatedAt: route.CreatedAt,
	})
}

// CreateMemo はメモを作成して返す。
func (repository *Repository) CreateMemo(ctx context.Context, memo models.Memo) (*models.Memo, error) {
	_, error := repository.connection.ExecContext(ctx, `
		INSERT INTO "Memo" (title, content, gameId)
		VALUES (?, ?, ?)
	`, memo.Title, memo.Content, memo.GameID)
	if error != nil {
		return nil, error
	}

	return repository.findLatestMemo(ctx, memo.GameID, memo.Title)
}

// UpdateMemo はメモを更新して返す。
func (repository *Repository) UpdateMemo(ctx context.Context, memo models.Memo) (*models.Memo, error) {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "Memo" SET title = ?, content = ? WHERE id = ?
	`, memo.Title, memo.Content, memo.ID)
	if error != nil {
		return nil, error
	}
	return repository.GetMemoByID(ctx, memo.ID)
}

// GetMemoByID はメモIDでメモを取得する。
func (repository *Repository) GetMemoByID(ctx context.Context, memoID string) (*models.Memo, error) {
	row := repository.connection.QueryRowContext(ctx, `
		SELECT id, title, content, gameId, createdAt, updatedAt
		FROM "Memo" WHERE id = ?
	`, memoID)

	memo, error := scanMemo(row)
	if error == sql.ErrNoRows {
		return nil, nil
	}
	if error != nil {
		return nil, error
	}
	return memo, nil
}

// FindMemoByTitle はゲームIDとタイトルでメモを取得する。
func (repository *Repository) FindMemoByTitle(ctx context.Context, gameID string, title string) (*models.Memo, error) {
	row := repository.connection.QueryRowContext(ctx, `
		SELECT id, title, content, gameId, createdAt, updatedAt
		FROM "Memo" WHERE gameId = ? AND title = ? LIMIT 1
	`, gameID, title)

	memo, error := scanMemo(row)
	if error == sql.ErrNoRows {
		return nil, nil
	}
	if error != nil {
		return nil, error
	}
	return memo, nil
}

// ListMemosByGame はゲームIDでメモ一覧を取得する。
func (repository *Repository) ListMemosByGame(ctx context.Context, gameID string) (memos []models.Memo, err error) {
	rows, err := repository.connection.QueryContext(ctx, `
		SELECT id, title, content, gameId, createdAt, updatedAt
		FROM "Memo" WHERE gameId = ? ORDER BY updatedAt DESC
	`, gameID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	memos = make([]models.Memo, 0)
	for rows.Next() {
		memo, err := scanMemo(rows)
		if err != nil {
			return nil, err
		}
		memos = append(memos, *memo)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return memos, nil
}

// ListAllMemos は全メモを取得する。
func (repository *Repository) ListAllMemos(ctx context.Context) (memos []models.Memo, err error) {
	rows, err := repository.connection.QueryContext(ctx, `
		SELECT id, title, content, gameId, createdAt, updatedAt
		FROM "Memo" ORDER BY updatedAt DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	memos = make([]models.Memo, 0)
	for rows.Next() {
		memo, err := scanMemo(rows)
		if err != nil {
			return nil, err
		}
		memos = append(memos, *memo)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return memos, nil
}

// DeleteMemo はメモを削除する。
func (repository *Repository) DeleteMemo(ctx context.Context, memoID string) error {
	_, error := repository.connection.ExecContext(ctx, `DELETE FROM "Memo" WHERE id = ?`, memoID)
	return error
}

// normalizeSortColumn は許可されたソート対象に変換する。
func normalizeSortColumn(sortBy string) string {
	switch sortBy {
	case "title", "publisher", "lastPlayed", "totalPlayTime", "createdAt":
		return sortBy
	default:
		return "title"
	}
}

// normalizeSortDirection はソート方向をSQL用に整形する。
func normalizeSortDirection(direction string) string {
	switch strings.ToLower(direction) {
	case "desc":
		return "DESC"
	default:
		return "ASC"
	}
}

// scanGame は1行分のゲームデータを読み取る。
func scanGame(row scanner) (*models.Game, error) {
	var (
		imagePath              sql.NullString
		saveFolderPath         sql.NullString
		localSaveHash          sql.NullString
		localSaveHashUpdatedAt sql.NullTime
		lastPlayed             sql.NullTime
		clearedAt              sql.NullTime
	)

	game := models.Game{}
	error := row.Scan(
		&game.ID,
		&game.Title,
		&game.Publisher,
		&imagePath,
		&game.ExePath,
		&saveFolderPath,
		&game.CreatedAt,
		&game.UpdatedAt,
		&localSaveHash,
		&localSaveHashUpdatedAt,
		&game.TotalPlayTime,
		&lastPlayed,
		&clearedAt,
	)
	if error != nil {
		return nil, error
	}

	game.ImagePath = nullStringPtr(imagePath)
	game.SaveFolderPath = nullStringPtr(saveFolderPath)
	game.LocalSaveHash = nullStringPtr(localSaveHash)
	game.LocalSaveHashUpdatedAt = nullTimePtr(localSaveHashUpdatedAt)
	game.LastPlayed = nullTimePtr(lastPlayed)
	game.ClearedAt = nullTimePtr(clearedAt)
	game.PlayStatus = normalizeProgressPlayStatus(game.LastPlayed, game.ClearedAt, game.TotalPlayTime)
	return &game, nil
}

func normalizeProgressPlayStatus(
	lastPlayed *time.Time,
	clearedAt *time.Time,
	totalPlayTime int64,
) models.PlayStatus {
	if clearedAt != nil {
		return models.PlayStatusCleared
	}
	if lastPlayed != nil || totalPlayTime > 0 {
		return models.PlayStatusPlaying
	}
	return models.PlayStatusUnplayed
}

// scanMemo は1行分のメモデータを読み取る。
func scanMemo(row scanner) (*models.Memo, error) {
	memo := models.Memo{}
	error := row.Scan(&memo.ID, &memo.Title, &memo.Content, &memo.GameID, &memo.CreatedAt, &memo.UpdatedAt)
	if error != nil {
		return nil, error
	}
	return &memo, nil
}

// nullStringPtr は NULL 文字列をポインタに変換する。
func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

// nullTimePtr は NULL 時刻をポインタに変換する。
func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func toNullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func toNullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func modelGameFromSQLC(game sqlcgen.Game) *models.Game {
	return &models.Game{
		ID:                     game.ID,
		Title:                  game.Title,
		Publisher:              game.Publisher,
		ImagePath:              nullStringPtr(game.ImagePath),
		ExePath:                game.ExePath,
		SaveFolderPath:         nullStringPtr(game.SaveFolderPath),
		CreatedAt:              game.CreatedAt,
		UpdatedAt:              game.UpdatedAt,
		LocalSaveHash:          nullStringPtr(game.LocalSaveHash),
		LocalSaveHashUpdatedAt: nullTimePtr(game.LocalSaveHashUpdatedAt),
		TotalPlayTime:          game.TotalPlayTime,
		LastPlayed:             nullTimePtr(game.LastPlayed),
		ClearedAt:              nullTimePtr(game.ClearedAt),
		PlayStatus:             normalizeProgressPlayStatus(nullTimePtr(game.LastPlayed), nullTimePtr(game.ClearedAt), game.TotalPlayTime),
	}
}

func modelPlaySessionFromSQLC(session sqlcgen.PlaySession) *models.PlaySession {
	return &models.PlaySession{
		ID:          session.ID,
		GameID:      session.GameId,
		PlayRouteID: nullStringPtr(session.PlayRouteId),
		PlayedAt:    session.PlayedAt,
		Duration:    session.Duration,
		UpdatedAt:   session.UpdatedAt,
	}
}

func modelPlayRouteFromSQLC(route sqlcgen.PlayRoute) *models.PlayRoute {
	return &models.PlayRoute{
		ID:        route.ID,
		GameID:    route.GameId,
		Name:      route.Name,
		SortOrder: int(route.SortOrder),
		CreatedAt: route.CreatedAt,
	}
}

// findLatestMemo は直近のメモを取得する。
func (repository *Repository) findLatestMemo(ctx context.Context, gameID string, title string) (*models.Memo, error) {
	row := repository.connection.QueryRowContext(ctx, `
		SELECT id, title, content, gameId, createdAt, updatedAt
		FROM "Memo" WHERE gameId = ? AND title = ? ORDER BY createdAt DESC LIMIT 1
	`, gameID, title)
	return scanMemo(row)
}

// scanner はScanだけを要求する簡易インターフェース。
type scanner interface {
	Scan(dest ...any) error
}
