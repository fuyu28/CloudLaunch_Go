// @fileoverview データベース操作の基本CRUDを提供する。
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"CloudLaunch_Go/internal/models"
)

// Repository は主要テーブルへのCRUDを提供する。
type Repository struct {
	connection *sql.DB
}

// NewRepository は Repository を初期化する。
func NewRepository(connection *sql.DB) *Repository {
	return &Repository{connection: connection}
}

// GetGameByID はID指定でゲームを取得する。
func (repository *Repository) GetGameByID(ctx context.Context, gameID string) (*models.Game, error) {
	row := repository.connection.QueryRowContext(ctx, `
		SELECT id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
		       localSaveHash, localSaveHashUpdatedAt,
		       totalPlayTime, lastPlayed, clearedAt, currentRouteId
		FROM "Game" WHERE id = ?
	`, gameID)

	game, error := scanGame(row)
	if error == sql.ErrNoRows {
		return nil, nil
	}
	if error != nil {
		return nil, error
	}
	return game, nil
}

// GetGameByExePath は実行ファイルパスに一致するゲームを取得する。
func (repository *Repository) GetGameByExePath(ctx context.Context, exePath string) (*models.Game, error) {
	trimmed := strings.TrimSpace(exePath)
	if trimmed == "" {
		return nil, errors.New("exePath is empty")
	}
	row := repository.connection.QueryRowContext(ctx, `
		SELECT id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
		       localSaveHash, localSaveHashUpdatedAt,
		       totalPlayTime, lastPlayed, clearedAt, currentRouteId
		FROM "Game" WHERE lower(exePath) = lower(?)
	`, trimmed)

	game, error := scanGame(row)
	if error == sql.ErrNoRows {
		return nil, nil
	}
	if error != nil {
		return nil, error
	}
	return game, nil
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
		       totalPlayTime, lastPlayed, clearedAt, currentRouteId
		FROM "Game"
	`)

	whereClauses := make([]string, 0, 2)
	args := make([]any, 0, 2)
	if searchText != "" {
		whereClauses = append(whereClauses, "(title LIKE ? OR publisher LIKE ?)")
		pattern := fmt.Sprintf("%%%s%%", searchText)
		args = append(args, pattern, pattern)
	}
	switch filter {
	case models.PlayStatusPlayed:
		whereClauses = append(whereClauses, "clearedAt IS NOT NULL")
	case models.PlayStatusPlaying:
		whereClauses = append(whereClauses, "lastPlayed IS NOT NULL AND clearedAt IS NULL")
	case models.PlayStatusUnplayed:
		whereClauses = append(whereClauses, "lastPlayed IS NULL AND clearedAt IS NULL")
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
	_, error := repository.connection.ExecContext(ctx, `
		INSERT INTO "Game" (title, publisher, imagePath, exePath, saveFolderPath, localSaveHash, localSaveHashUpdatedAt,
			totalPlayTime, lastPlayed, clearedAt, currentRouteId)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, game.Title, game.Publisher, game.ImagePath, game.ExePath, game.SaveFolderPath,
		game.LocalSaveHash, game.LocalSaveHashUpdatedAt,
		game.TotalPlayTime, game.LastPlayed, game.ClearedAt, game.CurrentRouteID)
	if error != nil {
		return nil, error
	}

	return repository.findLatestGame(ctx, game.Title, game.ExePath)
}

// UpdateGame はゲームを更新して返す。
func (repository *Repository) UpdateGame(ctx context.Context, game models.Game) (*models.Game, error) {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "Game" SET title = ?, publisher = ?, imagePath = ?, exePath = ?, saveFolderPath = ?,
			localSaveHash = ?, localSaveHashUpdatedAt = ?,
			totalPlayTime = ?, lastPlayed = ?, clearedAt = ?, currentRouteId = ?
		WHERE id = ?
	`, game.Title, game.Publisher, game.ImagePath, game.ExePath, game.SaveFolderPath,
		game.LocalSaveHash, game.LocalSaveHashUpdatedAt,
		game.TotalPlayTime, game.LastPlayed, game.ClearedAt, game.CurrentRouteID, game.ID)
	if error != nil {
		return nil, error
	}

	return repository.GetGameByID(ctx, game.ID)
}

// UpsertGameSync はID指定でゲームを追加/更新する。
func (repository *Repository) UpsertGameSync(ctx context.Context, game models.Game) error {
	_, error := repository.connection.ExecContext(ctx, `
		INSERT INTO "Game" (
			id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
			localSaveHash, localSaveHashUpdatedAt,
			totalPlayTime, lastPlayed, clearedAt, currentRouteId
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			publisher = excluded.publisher,
			imagePath = excluded.imagePath,
			exePath = excluded.exePath,
			saveFolderPath = excluded.saveFolderPath,
			createdAt = excluded.createdAt,
			updatedAt = excluded.updatedAt,
			localSaveHash = excluded.localSaveHash,
			localSaveHashUpdatedAt = excluded.localSaveHashUpdatedAt,
			totalPlayTime = excluded.totalPlayTime,
			lastPlayed = excluded.lastPlayed,
			clearedAt = excluded.clearedAt,
			currentRouteId = excluded.currentRouteId
	`, game.ID, game.Title, game.Publisher, game.ImagePath, game.ExePath, game.SaveFolderPath,
		game.CreatedAt, game.UpdatedAt, game.LocalSaveHash, game.LocalSaveHashUpdatedAt,
		game.TotalPlayTime, game.LastPlayed, game.ClearedAt, game.CurrentRouteID)
	return error
}

// TouchGameUpdatedAt はゲームのupdatedAtを現在時刻に更新する。
func (repository *Repository) TouchGameUpdatedAt(ctx context.Context, gameID string) error {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "Game" SET updatedAt = CURRENT_TIMESTAMP WHERE id = ?
	`, gameID)
	return error
}

// DeleteGame はゲームを削除する。
func (repository *Repository) DeleteGame(ctx context.Context, gameID string) error {
	_, error := repository.connection.ExecContext(ctx, `DELETE FROM "Game" WHERE id = ?`, gameID)
	return error
}

// CreateRoute はルートを作成して返す。
func (repository *Repository) CreateRoute(ctx context.Context, route models.Route) (*models.Route, error) {
	_, error := repository.connection.ExecContext(ctx, `
		INSERT INTO "Route" (name, "order", gameId)
		VALUES (?, ?, ?)
	`, route.Name, route.Order, route.GameID)
	if error != nil {
		return nil, error
	}

	return repository.findLatestRoute(ctx, route.GameID, route.Name)
}

// ListRoutesByGame はゲームIDでルート一覧を取得する。
func (repository *Repository) ListRoutesByGame(ctx context.Context, gameID string) (routes []models.Route, err error) {
	rows, err := repository.connection.QueryContext(ctx, `
		SELECT id, name, "order", gameId, createdAt
		FROM "Route" WHERE gameId = ? ORDER BY "order" ASC
	`, gameID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	routes = make([]models.Route, 0)
	for rows.Next() {
		route, err := scanRoute(rows)
		if err != nil {
			return nil, err
		}
		routes = append(routes, *route)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return routes, nil
}

// UpdateRoute はルートを更新して返す。
func (repository *Repository) UpdateRoute(ctx context.Context, route models.Route) (*models.Route, error) {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "Route" SET name = ?, "order" = ? WHERE id = ?
	`, route.Name, route.Order, route.ID)
	if error != nil {
		return nil, error
	}
	return repository.GetRouteByID(ctx, route.ID)
}

// UpdateRouteOrder はルートの順序を更新する。
func (repository *Repository) UpdateRouteOrder(ctx context.Context, routeID string, order int64) error {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "Route" SET "order" = ? WHERE id = ?
	`, order, routeID)
	return error
}

// GetRouteByID はルートIDでルートを取得する。
func (repository *Repository) GetRouteByID(ctx context.Context, routeID string) (*models.Route, error) {
	row := repository.connection.QueryRowContext(ctx, `
		SELECT id, name, "order", gameId, createdAt FROM "Route" WHERE id = ?
	`, routeID)

	route, error := scanRoute(row)
	if error == sql.ErrNoRows {
		return nil, nil
	}
	if error != nil {
		return nil, error
	}
	return route, nil
}

// DeleteRoute はルートを削除する。
func (repository *Repository) DeleteRoute(ctx context.Context, routeID string) error {
	_, error := repository.connection.ExecContext(ctx, `DELETE FROM "Route" WHERE id = ?`, routeID)
	return error
}

// CreatePlaySession はプレイセッションを作成して返す。
func (repository *Repository) CreatePlaySession(ctx context.Context, session models.PlaySession) (*models.PlaySession, error) {
	_, error := repository.connection.ExecContext(ctx, `
		INSERT INTO "PlaySession" (gameId, playedAt, duration, sessionName, routeId)
		VALUES (?, ?, ?, ?, ?)
	`, session.GameID, session.PlayedAt, session.Duration, session.SessionName, session.RouteID)
	if error != nil {
		return nil, error
	}

	return repository.findLatestPlaySession(ctx, session.GameID)
}

// GetPlaySessionByID はID指定でセッションを取得する。
func (repository *Repository) GetPlaySessionByID(ctx context.Context, sessionID string) (*models.PlaySession, error) {
	row := repository.connection.QueryRowContext(ctx, `
		SELECT id, gameId, playedAt, duration, sessionName, routeId, updatedAt
		FROM "PlaySession" WHERE id = ?
	`, sessionID)
	session, error := scanPlaySession(row)
	if error == sql.ErrNoRows {
		return nil, nil
	}
	if error != nil {
		return nil, error
	}
	return session, nil
}

// ListPlaySessionsByGame はゲームIDでセッション一覧を取得する。
func (repository *Repository) ListPlaySessionsByGame(ctx context.Context, gameID string) (sessions []models.PlaySession, err error) {
	rows, err := repository.connection.QueryContext(ctx, `
		SELECT id, gameId, playedAt, duration, sessionName, routeId, updatedAt
		FROM "PlaySession" WHERE gameId = ? ORDER BY playedAt DESC
	`, gameID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	sessions = make([]models.PlaySession, 0)
	for rows.Next() {
		session, err := scanPlaySession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *session)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

// DeletePlaySession はセッションを削除する。
func (repository *Repository) DeletePlaySession(ctx context.Context, sessionID string) error {
	_, error := repository.connection.ExecContext(ctx, `DELETE FROM "PlaySession" WHERE id = ?`, sessionID)
	return error
}

// DeletePlaySessionsByGame はゲームID配下のセッションを削除する。
func (repository *Repository) DeletePlaySessionsByGame(ctx context.Context, gameID string) error {
	_, error := repository.connection.ExecContext(ctx, `DELETE FROM "PlaySession" WHERE gameId = ?`, gameID)
	return error
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
		UPDATE "Game" SET totalPlayTime = ? WHERE id = ?
	`, totalPlayTime, gameID)
	return error
}

// UpdateGameTotalPlayTimeWithLastPlayed は総プレイ時間と最終プレイ日時を更新する。
func (repository *Repository) UpdateGameTotalPlayTimeWithLastPlayed(
	ctx context.Context,
	gameID string,
	totalPlayTime int64,
	playedAt time.Time,
) error {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "Game"
		SET totalPlayTime = ?,
		    lastPlayed = CASE
		      WHEN lastPlayed IS NULL OR lastPlayed < ? THEN ?
		      ELSE lastPlayed
		    END
		WHERE id = ?
	`, totalPlayTime, playedAt, playedAt, gameID)
	return error
}

// UpsertPlaySessionSync はID指定でセッションを追加/更新する。
func (repository *Repository) UpsertPlaySessionSync(ctx context.Context, session models.PlaySession) error {
	_, error := repository.connection.ExecContext(ctx, `
		INSERT INTO "PlaySession" (id, gameId, playedAt, duration, sessionName, routeId, updatedAt)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			gameId = excluded.gameId,
			playedAt = excluded.playedAt,
			duration = excluded.duration,
			sessionName = excluded.sessionName,
			routeId = excluded.routeId,
			updatedAt = excluded.updatedAt
	`, session.ID, session.GameID, session.PlayedAt, session.Duration, session.SessionName,
		session.RouteID, session.UpdatedAt)
	return error
}

// UpdatePlaySessionRoute はセッションのルートを更新する。
func (repository *Repository) UpdatePlaySessionRoute(ctx context.Context, sessionID string, routeID *string) error {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "PlaySession" SET routeId = ? WHERE id = ?
	`, routeID, sessionID)
	return error
}

// UpdatePlaySessionName はセッション名を更新する。
func (repository *Repository) UpdatePlaySessionName(ctx context.Context, sessionID string, sessionName string) error {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "PlaySession" SET sessionName = ? WHERE id = ?
	`, sessionName, sessionID)
	return error
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

// GetRouteStats はルートごとの統計を取得する。
func (repository *Repository) GetRouteStats(ctx context.Context, gameID string) (stats []models.RouteStat, err error) {
	rows, err := repository.connection.QueryContext(ctx, `
		SELECT r.id, r.name, r."order",
		       COALESCE(SUM(ps.duration), 0) as total_time,
		       COUNT(ps.id) as session_count
		FROM "Route" r
		LEFT JOIN "PlaySession" ps ON ps.routeId = r.id
		WHERE r.gameId = ?
		GROUP BY r.id, r.name, r."order"
		ORDER BY r."order" ASC
	`, gameID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	stats = make([]models.RouteStat, 0)
	for rows.Next() {
		var (
			routeID      string
			routeName    string
			orderValue   int64
			totalTime    int64
			sessionCount int64
		)
		if err := rows.Scan(&routeID, &routeName, &orderValue, &totalTime, &sessionCount); err != nil {
			return nil, err
		}
		average := float64(0)
		if sessionCount > 0 {
			average = float64(totalTime) / float64(sessionCount)
		}
		stats = append(stats, models.RouteStat{
			RouteID:      routeID,
			RouteName:    routeName,
			TotalTime:    totalTime,
			SessionCount: sessionCount,
			AverageTime:  average,
			Order:        orderValue,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return stats, nil
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
		currentRouteId         sql.NullString
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
		&currentRouteId,
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
	game.CurrentRouteID = nullStringPtr(currentRouteId)
	game.PlayStatus = models.ComputePlayStatus(game.ClearedAt, game.LastPlayed)

	return &game, nil
}

// scanRoute は1行分のルートデータを読み取る。
func scanRoute(row scanner) (*models.Route, error) {
	route := models.Route{}
	error := row.Scan(&route.ID, &route.Name, &route.Order, &route.GameID, &route.CreatedAt)
	if error != nil {
		return nil, error
	}
	return &route, nil
}

// scanPlaySession は1行分のセッションデータを読み取る。
func scanPlaySession(row scanner) (*models.PlaySession, error) {
	var (
		sessionName sql.NullString
		routeID     sql.NullString
	)

	session := models.PlaySession{}
	error := row.Scan(
		&session.ID,
		&session.GameID,
		&session.PlayedAt,
		&session.Duration,
		&sessionName,
		&routeID,
		&session.UpdatedAt,
	)
	if error != nil {
		return nil, error
	}

	session.SessionName = nullStringPtr(sessionName)
	session.RouteID = nullStringPtr(routeID)

	return &session, nil
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

// findLatestGame は直近作成のゲームを取得する。
func (repository *Repository) findLatestGame(ctx context.Context, title string, exePath string) (*models.Game, error) {
	row := repository.connection.QueryRowContext(ctx, `
		SELECT id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
		       localSaveHash, localSaveHashUpdatedAt,
		       totalPlayTime, lastPlayed, clearedAt, currentRouteId
		FROM "Game" WHERE title = ? AND exePath = ? ORDER BY createdAt DESC LIMIT 1
	`, title, exePath)
	return scanGame(row)
}

// findLatestRoute は直近作成のルートを取得する。
func (repository *Repository) findLatestRoute(ctx context.Context, gameID string, name string) (*models.Route, error) {
	row := repository.connection.QueryRowContext(ctx, `
		SELECT id, name, "order", gameId, createdAt
		FROM "Route" WHERE gameId = ? AND name = ? ORDER BY createdAt DESC LIMIT 1
	`, gameID, name)
	return scanRoute(row)
}

// findLatestPlaySession は直近のプレイセッションを取得する。
func (repository *Repository) findLatestPlaySession(ctx context.Context, gameID string) (*models.PlaySession, error) {
	row := repository.connection.QueryRowContext(ctx, `
		SELECT id, gameId, playedAt, duration, sessionName, routeId, updatedAt
		FROM "PlaySession" WHERE gameId = ? ORDER BY playedAt DESC LIMIT 1
	`, gameID)
	return scanPlaySession(row)
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
