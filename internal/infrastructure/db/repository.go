package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"CloudLaunch_Go/internal/domain"
)

// Repository は主要テーブルへのCRUDを提供する。
type Repository struct {
	connection *sql.DB
}

// NewRepository は Repository を初期化する。
func NewRepository(connection *sql.DB) *Repository {
	return &Repository{connection: connection}
}

// 同じカラム並びで SELECT する箇所をまとめ、列追加時の更新漏れを防ぐ。
const (
	gameSelectCols = `id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
		       localSaveHash, localSaveHashUpdatedAt, localSyncHead,
		       totalPlayTime, lastPlayed, clearedAt, playStatus, currentRouteId`
	routeSelectCols       = `id, name, "order", gameId, createdAt`
	playSessionSelectCols = `id, gameId, playedAt, duration, sessionName, routeId, updatedAt`
	memoSelectCols        = `id, title, content, gameId, createdAt, updatedAt`
)

// queryAll は QueryContext → 行ごとの scan → defer Close をまとめる。
// scan は1行ぶんを domain 型に変換する関数。
func queryAll[T any](
	ctx context.Context,
	conn *sql.DB,
	query string,
	scan func(scanner) (*T, error),
	args ...any,
) (results []T, err error) {
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	results = make([]T, 0)
	for rows.Next() {
		item, scanErr := scan(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		results = append(results, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// GetGameByID はID指定でゲームを取得する。
func (repository *Repository) GetGameByID(ctx context.Context, gameID string) (*domain.Game, error) {
	row := repository.connection.QueryRowContext(ctx, `SELECT `+gameSelectCols+` FROM "Game" WHERE id = ?`, gameID)
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
func (repository *Repository) GetGameByExePath(ctx context.Context, exePath string) (*domain.Game, error) {
	trimmed := strings.TrimSpace(exePath)
	if trimmed == "" {
		return nil, errors.New("exePath is empty")
	}
	row := repository.connection.QueryRowContext(ctx, `SELECT `+gameSelectCols+` FROM "Game" WHERE lower(exePath) = lower(?)`, trimmed)
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
	filter domain.PlayStatus,
	sortBy string,
	sortDirection string,
) ([]domain.Game, error) {
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`SELECT ` + gameSelectCols + ` FROM "Game"`)

	whereClauses := make([]string, 0, 2)
	args := make([]any, 0, 2)
	if searchText != "" {
		whereClauses = append(whereClauses, "(title LIKE ? OR publisher LIKE ?)")
		pattern := fmt.Sprintf("%%%s%%", searchText)
		args = append(args, pattern, pattern)
	}
	switch filter {
	case domain.PlayStatusPlayed, domain.PlayStatusPlaying, domain.PlayStatusUnplayed:
		whereClauses = append(whereClauses, "playStatus = ?")
		args = append(args, string(filter))
	}
	if len(whereClauses) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(whereClauses, " AND "))
	}

	_, _ = fmt.Fprintf(&queryBuilder, " ORDER BY %s %s", normalizeSortColumn(sortBy), normalizeSortDirection(sortDirection))
	return queryAll(ctx, repository.connection, queryBuilder.String(), scanGame, args...)
}

// CreateGame はゲームを作成して返す。
func (repository *Repository) CreateGame(ctx context.Context, game domain.Game) (*domain.Game, error) {
	_, error := repository.connection.ExecContext(ctx, `
		INSERT INTO "Game" (title, publisher, imagePath, exePath, saveFolderPath, localSaveHash, localSaveHashUpdatedAt,
			totalPlayTime, lastPlayed, clearedAt, playStatus, currentRouteId)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, game.Title, game.Publisher, game.ImagePath, game.ExePath, game.SaveFolderPath,
		game.LocalSaveHash, game.LocalSaveHashUpdatedAt,
		game.TotalPlayTime, game.LastPlayed, game.ClearedAt, game.PlayStatus, game.CurrentRouteID)
	if error != nil {
		return nil, error
	}

	return repository.findLatestGame(ctx, game.Title, game.ExePath)
}

// CreateGameWithInitialRoute はゲームと初期ルートを単一トランザクションで作成する。
func (repository *Repository) CreateGameWithInitialRoute(
	ctx context.Context,
	game domain.Game,
	initialRoute domain.Route,
) (created *domain.Game, err error) {
	tx, err := repository.connection.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var gameID string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO "Game" (title, publisher, imagePath, exePath, saveFolderPath, localSaveHash, localSaveHashUpdatedAt,
			totalPlayTime, lastPlayed, clearedAt, playStatus, currentRouteId)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`, game.Title, game.Publisher, game.ImagePath, game.ExePath, game.SaveFolderPath,
		game.LocalSaveHash, game.LocalSaveHashUpdatedAt,
		game.TotalPlayTime, game.LastPlayed, game.ClearedAt, game.PlayStatus, game.CurrentRouteID).Scan(&gameID)
	if err != nil {
		return nil, err
	}

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO "Route" (name, "order", gameId)
		VALUES (?, ?, ?)
	`, initialRoute.Name, initialRoute.Order, gameID); err != nil {
		return nil, err
	}

	created, err = scanGame(tx.QueryRowContext(ctx, `SELECT `+gameSelectCols+` FROM "Game" WHERE id = ?`, gameID))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
}

// UpdateGame はゲームを更新して返す。
func (repository *Repository) UpdateGame(ctx context.Context, game domain.Game) (*domain.Game, error) {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "Game" SET title = ?, publisher = ?, imagePath = ?, exePath = ?, saveFolderPath = ?,
			localSaveHash = ?, localSaveHashUpdatedAt = ?,
			totalPlayTime = ?, lastPlayed = ?, clearedAt = ?, playStatus = ?, currentRouteId = ?
		WHERE id = ?
	`, game.Title, game.Publisher, game.ImagePath, game.ExePath, game.SaveFolderPath,
		game.LocalSaveHash, game.LocalSaveHashUpdatedAt,
		game.TotalPlayTime, game.LastPlayed, game.ClearedAt, game.PlayStatus, game.CurrentRouteID, game.ID)
	if error != nil {
		return nil, error
	}

	return repository.GetGameByID(ctx, game.ID)
}

// UpsertGameSync はID指定でゲームを追加/更新する。
func (repository *Repository) UpsertGameSync(ctx context.Context, game domain.Game) error {
	_, error := repository.connection.ExecContext(ctx, `
		INSERT INTO "Game" (
			id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
			localSaveHash, localSaveHashUpdatedAt,
			totalPlayTime, lastPlayed, clearedAt, playStatus, currentRouteId
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			playStatus = excluded.playStatus,
			currentRouteId = excluded.currentRouteId
	`, game.ID, game.Title, game.Publisher, game.ImagePath, game.ExePath, game.SaveFolderPath,
		game.CreatedAt, game.UpdatedAt, game.LocalSaveHash, game.LocalSaveHashUpdatedAt,
		game.TotalPlayTime, game.LastPlayed, game.ClearedAt, game.PlayStatus, game.CurrentRouteID)
	return error
}

// TouchGameUpdatedAt はゲームのupdatedAtを現在時刻に更新する。
func (repository *Repository) TouchGameUpdatedAt(ctx context.Context, gameID string) error {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "Game" SET updatedAt = CURRENT_TIMESTAMP WHERE id = ?
	`, gameID)
	return error
}

// DeleteGameAndQueueMemoCleanup はメモ削除保留の記録とゲーム削除を単一トランザクションで行う。
func (repository *Repository) DeleteGameAndQueueMemoCleanup(ctx context.Context, gameID string) (err error) {
	tx, err := repository.connection.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO "PendingMemoCleanup" (gameId) VALUES (?)
		ON CONFLICT(gameId) DO NOTHING
	`, gameID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM "Game" WHERE id = ?`, gameID); err != nil {
		return err
	}
	return tx.Commit()
}

// ListPendingMemoCleanup はローカルメモ削除が保留中のゲームIDを返す。
func (repository *Repository) ListPendingMemoCleanup(ctx context.Context) ([]string, error) {
	rows, err := repository.connection.QueryContext(ctx, `
		SELECT gameId FROM "PendingMemoCleanup" ORDER BY createdAt, gameId
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	gameIDs := make([]string, 0)
	for rows.Next() {
		var gameID string
		if err := rows.Scan(&gameID); err != nil {
			return nil, err
		}
		gameIDs = append(gameIDs, gameID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return gameIDs, nil
}

// ClearPendingMemoCleanup は完了したローカルメモ削除の保留記録を消す。
func (repository *Repository) ClearPendingMemoCleanup(ctx context.Context, gameID string) error {
	_, err := repository.connection.ExecContext(ctx, `
		DELETE FROM "PendingMemoCleanup" WHERE gameId = ?
	`, gameID)
	return err
}

// CreateRoute はルートを作成して返す。
func (repository *Repository) CreateRoute(ctx context.Context, route domain.Route) (*domain.Route, error) {
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
func (repository *Repository) ListRoutesByGame(ctx context.Context, gameID string) ([]domain.Route, error) {
	return queryAll(ctx, repository.connection,
		`SELECT `+routeSelectCols+` FROM "Route" WHERE gameId = ? ORDER BY "order" ASC`,
		scanRoute, gameID)
}

// UpdateRoute はルートを更新して返す。
func (repository *Repository) UpdateRoute(ctx context.Context, route domain.Route) (*domain.Route, error) {
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

// UpdateRouteOrders は gameID 配下のルートの順序更新を単一トランザクションで実行する。
// 部分失敗を防ぎ、行数ぶんのラウンドトリップを1トランザクションに収める。
// WHERE 句に gameId 条件を含むため、誤って他ゲームの Route ID を渡しても無視される。
func (repository *Repository) UpdateRouteOrders(ctx context.Context, gameID string, items []domain.RouteOrderItem) (err error) {
	if len(items) == 0 {
		return nil
	}
	tx, err := repository.connection.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	for _, item := range items {
		if _, execErr := tx.ExecContext(ctx, `UPDATE "Route" SET "order" = ? WHERE id = ? AND gameId = ?`, item.Order, item.ID, gameID); execErr != nil {
			return execErr
		}
	}
	return tx.Commit()
}

// GetRouteByID はルートIDでルートを取得する。
func (repository *Repository) GetRouteByID(ctx context.Context, routeID string) (*domain.Route, error) {
	row := repository.connection.QueryRowContext(ctx, `SELECT `+routeSelectCols+` FROM "Route" WHERE id = ?`, routeID)
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

// CreatePlaySession はプレイセッションを作成して返す（Game 派生キャッシュは更新しない）。
// 通常のユースケースは CreatePlaySessionAndRefreshGame を使う。
func (repository *Repository) CreatePlaySession(ctx context.Context, session domain.PlaySession) (*domain.PlaySession, error) {
	var id string
	error := repository.connection.QueryRowContext(ctx, `
		INSERT INTO "PlaySession" (gameId, playedAt, duration, sessionName, routeId)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id
	`, session.GameID, session.PlayedAt, session.Duration, session.SessionName, session.RouteID).Scan(&id)
	if error != nil {
		return nil, error
	}

	return repository.GetPlaySessionByID(ctx, id)
}

// CreatePlaySessionAndRefreshGame はセッション作成と Game.totalPlayTime / lastPlayed の再計算を
// 単一トランザクションで行う。PlaySession が正本、Game 列は派生キャッシュ。
func (repository *Repository) CreatePlaySessionAndRefreshGame(
	ctx context.Context,
	session domain.PlaySession,
) (created *domain.PlaySession, err error) {
	tx, err := repository.connection.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var id string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO "PlaySession" (gameId, playedAt, duration, sessionName, routeId)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id
	`, session.GameID, session.PlayedAt, session.Duration, session.SessionName, session.RouteID).Scan(&id)
	if err != nil {
		return nil, err
	}

	if err = refreshGamePlayTimeFromSessionsTx(ctx, tx, session.GameID); err != nil {
		return nil, err
	}

	created, err = scanPlaySession(tx.QueryRowContext(ctx, `SELECT `+playSessionSelectCols+` FROM "PlaySession" WHERE id = ?`, id))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
}

// GetPlaySessionByID はID指定でセッションを取得する。
func (repository *Repository) GetPlaySessionByID(ctx context.Context, sessionID string) (*domain.PlaySession, error) {
	row := repository.connection.QueryRowContext(ctx, `SELECT `+playSessionSelectCols+` FROM "PlaySession" WHERE id = ?`, sessionID)
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
// playedAt が同値のときも順序が安定するよう id を第2ソートキーにしている。
// （sessions.json のシリアライズ結果が環境ごとにブレてハッシュが変わるのを防ぐ）
func (repository *Repository) ListPlaySessionsByGame(ctx context.Context, gameID string) ([]domain.PlaySession, error) {
	return queryAll(ctx, repository.connection,
		`SELECT `+playSessionSelectCols+` FROM "PlaySession" WHERE gameId = ? ORDER BY playedAt DESC, id`,
		scanPlaySession, gameID)
}

// DeletePlaySession はセッションを削除する（Game 派生キャッシュは更新しない）。
// 通常のユースケースは DeletePlaySessionAndRefreshGame を使う。
func (repository *Repository) DeletePlaySession(ctx context.Context, sessionID string) error {
	_, error := repository.connection.ExecContext(ctx, `DELETE FROM "PlaySession" WHERE id = ?`, sessionID)
	return error
}

// DeletePlaySessionAndRefreshGame はセッション削除と Game 派生キャッシュ再計算を単一トランザクションで行う。
// 削除したセッションの gameID を返す（存在しなければ空文字）。
func (repository *Repository) DeletePlaySessionAndRefreshGame(
	ctx context.Context,
	sessionID string,
) (gameID string, err error) {
	tx, err := repository.connection.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	err = tx.QueryRowContext(ctx, `SELECT gameId FROM "PlaySession" WHERE id = ?`, sessionID).Scan(&gameID)
	if errors.Is(err, sql.ErrNoRows) {
		if err = tx.Commit(); err != nil {
			return "", err
		}
		return "", nil
	}
	if err != nil {
		return "", err
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM "PlaySession" WHERE id = ?`, sessionID); err != nil {
		return "", err
	}
	if err = refreshGamePlayTimeFromSessionsTx(ctx, tx, gameID); err != nil {
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	return gameID, nil
}

// RefreshGamePlayTimeFromSessions は Game.totalPlayTime / lastPlayed をセッション集計から再構築する。
func (repository *Repository) RefreshGamePlayTimeFromSessions(ctx context.Context, gameID string) (err error) {
	tx, err := repository.connection.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if err = refreshGamePlayTimeFromSessionsTx(ctx, tx, gameID); err != nil {
		return err
	}
	err = tx.Commit()
	return err
}

// refreshGamePlayTimeFromSessionsTx は tx 内で SUM(duration) / MAX(playedAt) を Game へ反映する。
func refreshGamePlayTimeFromSessionsTx(ctx context.Context, tx *sql.Tx, gameID string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE "Game"
		SET totalPlayTime = (
			SELECT COALESCE(SUM(duration), 0) FROM "PlaySession" WHERE gameId = ?
		),
		lastPlayed = (
			SELECT MAX(playedAt) FROM "PlaySession" WHERE gameId = ?
		),
		updatedAt = CURRENT_TIMESTAMP
		WHERE id = ?
	`, gameID, gameID, gameID)
	return err
}

// ListPlaySessionsByGames は複数ゲームのセッションを一括取得し、gameID→sessions の map を返す。
// N+1 を避けたい一括処理（エクスポート等）で使う。空入力なら空 map を返す。
func (repository *Repository) ListPlaySessionsByGames(ctx context.Context, gameIDs []string) (map[string][]domain.PlaySession, error) {
	result := make(map[string][]domain.PlaySession, len(gameIDs))
	for _, id := range gameIDs {
		result[id] = nil
	}
	if len(gameIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(gameIDs))
	args := make([]any, len(gameIDs))
	for i, id := range gameIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := `SELECT ` + playSessionSelectCols + ` FROM "PlaySession" WHERE gameId IN (` +
		strings.Join(placeholders, ",") + `) ORDER BY playedAt DESC, id`

	sessions, err := queryAll(ctx, repository.connection, query, scanPlaySession, args...)
	if err != nil {
		return nil, err
	}
	for _, s := range sessions {
		result[s.GameID] = append(result[s.GameID], s)
	}
	return result, nil
}

// DeletePlaySessionsByGame はゲームID配下のセッションを削除する。
func (repository *Repository) DeletePlaySessionsByGame(ctx context.Context, gameID string) error {
	_, error := repository.connection.ExecContext(ctx, `DELETE FROM "PlaySession" WHERE gameId = ?`, gameID)
	return error
}

// GetLocalSaveTree はゲームの localSaveTree（前回同期した SaveSnapshot JSON）を取得する。
// 未設定の場合は "" を返す。
func (repository *Repository) GetLocalSaveTree(ctx context.Context, gameID string) (string, error) {
	var value sql.NullString
	err := repository.connection.QueryRowContext(ctx, `
		SELECT localSaveTree FROM "Game" WHERE id = ?
	`, gameID).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value.String, nil
}

// SetLocalSyncState は localSyncHead と localSaveTree を単一トランザクションで更新する。
func (repository *Repository) SetLocalSyncState(ctx context.Context, gameID, syncHead, saveTree string) (err error) {
	tx, err := repository.connection.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if err = setLocalSyncStateTx(ctx, tx, gameID, syncHead, saveTree); err != nil {
		return err
	}
	return tx.Commit()
}

func setLocalSyncStateTx(ctx context.Context, tx *sql.Tx, gameID, syncHead, saveTree string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE "Game" SET localSyncHead = ?, localSaveTree = ? WHERE id = ?
	`, syncHead, saveTree, gameID)
	return err
}

// BeginPendingPush はリモート HEAD 更新前に pending Push を永続化する。
func (repository *Repository) BeginPendingPush(ctx context.Context, pending domain.PendingPush) error {
	_, err := repository.connection.ExecContext(ctx, `
		INSERT INTO "PendingPush" (gameId, expectedRemoteHead, newCommitHash, contentFingerprint, saveTree)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(gameId) DO UPDATE SET
			expectedRemoteHead = excluded.expectedRemoteHead,
			newCommitHash = excluded.newCommitHash,
			contentFingerprint = excluded.contentFingerprint,
			saveTree = excluded.saveTree,
			createdAt = CURRENT_TIMESTAMP
	`, pending.GameID, pending.ExpectedRemoteHead, pending.NewCommitHash,
		pending.ContentFingerprint, pending.SaveTree)
	return err
}

// FinalizePendingPush は local baseline 更新と pending 削除を単一トランザクションで行う。
func (repository *Repository) FinalizePendingPush(ctx context.Context, gameID, syncHead, saveTree string) (err error) {
	tx, err := repository.connection.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if err = setLocalSyncStateTx(ctx, tx, gameID, syncHead, saveTree); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM "PendingPush" WHERE gameId = ?`, gameID); err != nil {
		return err
	}
	return tx.Commit()
}

// ClearPendingPush は baseline を変えずに pending だけ削除する。
func (repository *Repository) ClearPendingPush(ctx context.Context, gameID string) error {
	_, err := repository.connection.ExecContext(ctx, `
		DELETE FROM "PendingPush" WHERE gameId = ?
	`, gameID)
	return err
}

// ListPendingPushes は未確定の pending Push を返す。
func (repository *Repository) ListPendingPushes(ctx context.Context) ([]domain.PendingPush, error) {
	rows, err := repository.connection.QueryContext(ctx, `
		SELECT gameId, expectedRemoteHead, newCommitHash, contentFingerprint, saveTree
		FROM "PendingPush"
		ORDER BY createdAt, gameId
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	pending := make([]domain.PendingPush, 0)
	for rows.Next() {
		var item domain.PendingPush
		if err := rows.Scan(
			&item.GameID,
			&item.ExpectedRemoteHead,
			&item.NewCommitHash,
			&item.ContentFingerprint,
			&item.SaveTree,
		); err != nil {
			return nil, err
		}
		pending = append(pending, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return pending, nil
}

// BeginPullOperation はセーブ交換直前に PREPARED ジャーナルを永続化する。
func (repository *Repository) BeginPullOperation(ctx context.Context, op domain.PullOperation) error {
	if op.Status == "" {
		op.Status = domain.PullOperationPrepared
	}
	hadLive := 0
	if op.HadLive {
		hadLive = 1
	}
	_, err := repository.connection.ExecContext(ctx, `
		INSERT INTO "PullOperation" (operationId, gameId, livePath, stagePath, backupPath, commitHash, status, hadLive)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, op.OperationID, op.GameID, op.LivePath, op.StagePath, op.BackupPath, op.CommitHash, string(op.Status), hadLive)
	return err
}

// ClearPullOperation は指定ジャーナルを削除する。
func (repository *Repository) ClearPullOperation(ctx context.Context, operationID string) error {
	_, err := repository.connection.ExecContext(ctx, `
		DELETE FROM "PullOperation" WHERE operationId = ?
	`, operationID)
	return err
}

// ListPullOperations は未完了の Pull ジャーナルを返す。
func (repository *Repository) ListPullOperations(ctx context.Context) ([]domain.PullOperation, error) {
	rows, err := repository.connection.QueryContext(ctx, `
		SELECT operationId, gameId, livePath, stagePath, backupPath, commitHash, status, hadLive
		FROM "PullOperation"
		ORDER BY createdAt, operationId
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	ops := make([]domain.PullOperation, 0)
	for rows.Next() {
		var item domain.PullOperation
		var status string
		var hadLive int
		if err := rows.Scan(
			&item.OperationID,
			&item.GameID,
			&item.LivePath,
			&item.StagePath,
			&item.BackupPath,
			&item.CommitHash,
			&status,
			&hadLive,
		); err != nil {
			return nil, err
		}
		item.Status = domain.PullOperationStatus(status)
		item.HadLive = hadLive != 0
		ops = append(ops, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ops, nil
}

func markPullOperationAppliedTx(ctx context.Context, tx *sql.Tx, operationID string) error {
	if operationID == "" {
		return nil
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE "PullOperation" SET status = ? WHERE operationId = ? AND status = ?
	`, string(domain.PullOperationApplied), operationID, string(domain.PullOperationPrepared))
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("pull operation not found or not PREPARED: %s", operationID)
	}
	return nil
}

// GetSetting は Settings テーブルから値を取得する。存在しない場合は "" を返す。
func (repository *Repository) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := repository.connection.QueryRowContext(ctx, `
		SELECT value FROM "Settings" WHERE key = ?
	`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// UpsertSetting は Settings テーブルに値を追加または更新する。
func (repository *Repository) UpsertSetting(ctx context.Context, key, value string) error {
	_, err := repository.connection.ExecContext(ctx, `
		INSERT INTO "Settings" (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}

// UpsertPlaySessionSync はID指定でセッションを追加/更新する。
func (repository *Repository) UpsertPlaySessionSync(ctx context.Context, session domain.PlaySession) error {
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

// routeExistsTx は tx 内で Route が存在するか確認する。存在しなければ NULL 正規化のため nil を返す。
func routeExistsTx(ctx context.Context, tx *sql.Tx, routeID *string) (*string, error) {
	if routeID == nil || *routeID == "" {
		return nil, nil
	}
	var dummy int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM "Route" WHERE id = ?`, *routeID).Scan(&dummy)
	if err == sql.ErrNoRows {
		return nil, nil // ローカルに該当 Route が無い → NULL に正規化
	}
	if err != nil {
		return nil, err
	}
	return routeID, nil
}

// ApplyPullResult は v1 Pull のローカル反映を単一トランザクションで実行する。
// ローカル Route は削除せず、存在しない Route 参照は NULL に正規化して FK 違反を防ぐ。
// totalPlayTime / lastPlayed は game.json を信用せず、投入したセッションから SUM/MAX で導出する。
func (repository *Repository) ApplyPullResult(
	ctx context.Context,
	game domain.Game,
	sessions []domain.PlaySession,
	syncHead, saveTree, pullOperationID string,
) (err error) {
	tx, err := repository.connection.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	game.CurrentRouteID, err = routeExistsTx(ctx, tx, game.CurrentRouteID)
	if err != nil {
		return err
	}

	if err = upsertGameForPullTx(ctx, tx, game, game.CurrentRouteID); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM "PlaySession" WHERE gameId = ?`, game.ID); err != nil {
		return err
	}

	for _, session := range sessions {
		var routeID *string
		routeID, err = routeExistsTx(ctx, tx, session.RouteID)
		if err != nil {
			return err
		}
		if err = insertPlaySessionForPullTx(ctx, tx, game.ID, session, routeID); err != nil {
			return err
		}
	}

	if err = refreshGamePlayTimeFromSessionsTx(ctx, tx, game.ID); err != nil {
		return err
	}
	if err = setLocalSyncStateTx(ctx, tx, game.ID, syncHead, saveTree); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM "PendingPush" WHERE gameId = ?`, game.ID); err != nil {
		return err
	}
	if err = markPullOperationAppliedTx(ctx, tx, pullOperationID); err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

// ApplyPullResultV2 は v2 Pull のローカル反映を単一トランザクションで実行する。
//
// 適用順（FK 制約のため）:
// 1. Game を currentRouteId=NULL で upsert
// 2. 旧 Session 削除
// 3. 旧 Route 削除
// 4. クラウド Route を ID 保持で insert
// 5. currentRouteId を設定
// 6. Session を insert
// 7. playtime をセッションから導出
// 8. baseline / pending を更新
//
// 不正・重複・他ゲーム混入・参照欠落はサイレント正規化せずエラー（全体 rollback）。
func (repository *Repository) ApplyPullResultV2(
	ctx context.Context,
	game domain.Game,
	routes []domain.Route,
	sessions []domain.PlaySession,
	syncHead, saveTree, pullOperationID string,
) (err error) {
	if err = validatePullRoutesV2(game.ID, routes, game.CurrentRouteID, sessions); err != nil {
		return err
	}

	tx, err := repository.connection.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 1. currentRoute はまだ無いので NULL で upsert
	if err = upsertGameForPullTx(ctx, tx, game, nil); err != nil {
		return err
	}

	// 2–3. Session → Route の順で削除（Session が Route を参照するため）
	if _, err = tx.ExecContext(ctx, `DELETE FROM "PlaySession" WHERE gameId = ?`, game.ID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM "Route" WHERE gameId = ?`, game.ID); err != nil {
		return err
	}

	// 4. Route を ID 保持で insert
	for _, route := range routes {
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO "Route" (id, name, "order", gameId, createdAt)
			VALUES (?, ?, ?, ?, ?)
		`, route.ID, route.Name, route.Order, game.ID, route.CreatedAt); err != nil {
			return fmt.Errorf("route insert failed: %w", err)
		}
	}

	// 5. currentRouteId を設定
	if _, err = tx.ExecContext(ctx, `UPDATE "Game" SET currentRouteId = ? WHERE id = ?`, game.CurrentRouteID, game.ID); err != nil {
		return err
	}

	// 6. Session insert（参照は validate 済み）
	for _, session := range sessions {
		if err = insertPlaySessionForPullTx(ctx, tx, game.ID, session, session.RouteID); err != nil {
			return err
		}
	}

	// 7–8. playtime / baseline / pending / pull journal
	if err = refreshGamePlayTimeFromSessionsTx(ctx, tx, game.ID); err != nil {
		return err
	}
	if err = setLocalSyncStateTx(ctx, tx, game.ID, syncHead, saveTree); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM "PendingPush" WHERE gameId = ?`, game.ID); err != nil {
		return err
	}
	if err = markPullOperationAppliedTx(ctx, tx, pullOperationID); err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

// validatePullRoutesV2 は v2 Pull 入力の Route / 参照整合を検証する。
func validatePullRoutesV2(gameID string, routes []domain.Route, currentRouteID *string, sessions []domain.PlaySession) error {
	seenID := make(map[string]struct{}, len(routes))
	seenOrder := make(map[int64]struct{}, len(routes))
	seenName := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		id := strings.TrimSpace(route.ID)
		name := strings.TrimSpace(route.Name)
		if id == "" {
			return fmt.Errorf("malformed route: empty id")
		}
		if name == "" {
			return fmt.Errorf("malformed route: empty name")
		}
		if route.Order < 0 {
			return fmt.Errorf("malformed route: negative order")
		}
		if route.GameID != "" && route.GameID != gameID {
			return fmt.Errorf("route %s belongs to wrong game", id)
		}
		if _, ok := seenID[id]; ok {
			return fmt.Errorf("duplicate route id: %s", id)
		}
		if _, ok := seenOrder[route.Order]; ok {
			return fmt.Errorf("duplicate route order: %d", route.Order)
		}
		if _, ok := seenName[name]; ok {
			return fmt.Errorf("duplicate route name: %s", name)
		}
		seenID[id] = struct{}{}
		seenOrder[route.Order] = struct{}{}
		seenName[name] = struct{}{}
	}

	if currentRouteID != nil && *currentRouteID != "" {
		if _, ok := seenID[*currentRouteID]; !ok {
			return fmt.Errorf("missing route reference: currentRouteId=%s", *currentRouteID)
		}
	}
	for _, session := range sessions {
		if session.RouteID != nil && *session.RouteID != "" {
			if _, ok := seenID[*session.RouteID]; !ok {
				return fmt.Errorf("missing route reference: session %s routeId=%s", session.ID, *session.RouteID)
			}
		}
	}
	return nil
}

func upsertGameForPullTx(ctx context.Context, tx *sql.Tx, game domain.Game, currentRouteID *string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO "Game" (
			id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
			localSaveHash, localSaveHashUpdatedAt,
			totalPlayTime, lastPlayed, clearedAt, playStatus, currentRouteId
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, NULL, ?, ?, ?)
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
			totalPlayTime = 0,
			lastPlayed = NULL,
			clearedAt = excluded.clearedAt,
			playStatus = excluded.playStatus,
			currentRouteId = excluded.currentRouteId
	`, game.ID, game.Title, game.Publisher, game.ImagePath, game.ExePath, game.SaveFolderPath,
		game.CreatedAt, game.UpdatedAt, game.LocalSaveHash, game.LocalSaveHashUpdatedAt,
		game.ClearedAt, game.PlayStatus, currentRouteID)
	return err
}

func insertPlaySessionForPullTx(ctx context.Context, tx *sql.Tx, gameID string, session domain.PlaySession, routeID *string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO "PlaySession" (id, gameId, playedAt, duration, sessionName, routeId, updatedAt)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			gameId = excluded.gameId,
			playedAt = excluded.playedAt,
			duration = excluded.duration,
			sessionName = excluded.sessionName,
			routeId = excluded.routeId,
			updatedAt = excluded.updatedAt
	`, session.ID, gameID, session.PlayedAt, session.Duration, session.SessionName,
		routeID, session.UpdatedAt)
	return err
}

// UpdatePlaySessionRoute はセッションのルートを更新する。
func (repository *Repository) UpdatePlaySessionRoute(ctx context.Context, sessionID string, routeID *string) error {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "PlaySession" SET routeId = ? WHERE id = ?
	`, routeID, sessionID)
	return error
}

// UpdatePlaySessionName はセッション名を更新する。
// 空文字は NULL に丸めることで、フロントエンドからのクリア要求（"未設定"に戻す）を実現する。
func (repository *Repository) UpdatePlaySessionName(ctx context.Context, sessionID string, sessionName string) error {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "PlaySession" SET sessionName = NULLIF(?, '') WHERE id = ?
	`, sessionName, sessionID)
	return error
}

// CreateMemo はメモを作成して返す。
// memo.ID が空でなければその ID で挿入する（クラウド→ローカル同期で ID を保持するため）。
// 空なら SQLite の DEFAULT（randomblob）に任せる。
func (repository *Repository) CreateMemo(ctx context.Context, memo domain.Memo) (*domain.Memo, error) {
	if strings.TrimSpace(memo.ID) != "" {
		_, error := repository.connection.ExecContext(ctx, `
			INSERT INTO "Memo" (id, title, content, gameId)
			VALUES (?, ?, ?, ?)
		`, memo.ID, memo.Title, memo.Content, memo.GameID)
		if error != nil {
			return nil, error
		}
		return repository.GetMemoByID(ctx, memo.ID)
	}

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
func (repository *Repository) UpdateMemo(ctx context.Context, memo domain.Memo) (*domain.Memo, error) {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "Memo" SET title = ?, content = ? WHERE id = ?
	`, memo.Title, memo.Content, memo.ID)
	if error != nil {
		return nil, error
	}
	return repository.GetMemoByID(ctx, memo.ID)
}

// GetMemoByID はメモIDでメモを取得する。
func (repository *Repository) GetMemoByID(ctx context.Context, memoID string) (*domain.Memo, error) {
	row := repository.connection.QueryRowContext(ctx, `SELECT `+memoSelectCols+` FROM "Memo" WHERE id = ?`, memoID)
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
func (repository *Repository) FindMemoByTitle(ctx context.Context, gameID string, title string) (*domain.Memo, error) {
	row := repository.connection.QueryRowContext(ctx, `SELECT `+memoSelectCols+` FROM "Memo" WHERE gameId = ? AND title = ? LIMIT 1`, gameID, title)
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
func (repository *Repository) ListMemosByGame(ctx context.Context, gameID string) ([]domain.Memo, error) {
	return queryAll(ctx, repository.connection,
		`SELECT `+memoSelectCols+` FROM "Memo" WHERE gameId = ? ORDER BY updatedAt DESC`,
		scanMemo, gameID)
}

// GetRouteStats はルートごとの統計を取得する。
func (repository *Repository) GetRouteStats(ctx context.Context, gameID string) (stats []domain.RouteStat, err error) {
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

	stats = make([]domain.RouteStat, 0)
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
		stats = append(stats, domain.RouteStat{
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
func (repository *Repository) ListAllMemos(ctx context.Context) ([]domain.Memo, error) {
	return queryAll(ctx, repository.connection,
		`SELECT `+memoSelectCols+` FROM "Memo" ORDER BY updatedAt DESC`,
		scanMemo)
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
	if strings.EqualFold(direction, "desc") {
		return "DESC"
	}
	return "ASC"
}

// scanGame は1行分のゲームデータを読み取る。
func scanGame(row scanner) (*domain.Game, error) {
	var (
		imagePath              sql.NullString
		saveFolderPath         sql.NullString
		localSaveHash          sql.NullString
		localSaveHashUpdatedAt sql.NullTime
		localSyncHead          sql.NullString
		lastPlayed             sql.NullTime
		clearedAt              sql.NullTime
		currentRouteId         sql.NullString
	)

	game := domain.Game{}
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
		&localSyncHead,
		&game.TotalPlayTime,
		&lastPlayed,
		&clearedAt,
		&game.PlayStatus,
		&currentRouteId,
	)
	if error != nil {
		return nil, error
	}

	game.ImagePath = nullStringPtr(imagePath)
	game.SaveFolderPath = nullStringPtr(saveFolderPath)
	game.LocalSaveHash = nullStringPtr(localSaveHash)
	game.LocalSaveHashUpdatedAt = nullTimePtr(localSaveHashUpdatedAt)
	game.LocalSyncHead = nullStringPtr(localSyncHead)
	game.LastPlayed = nullTimePtr(lastPlayed)
	game.ClearedAt = nullTimePtr(clearedAt)
	game.CurrentRouteID = nullStringPtr(currentRouteId)

	return &game, nil
}

// scanRoute は1行分のルートデータを読み取る。
func scanRoute(row scanner) (*domain.Route, error) {
	route := domain.Route{}
	error := row.Scan(&route.ID, &route.Name, &route.Order, &route.GameID, &route.CreatedAt)
	if error != nil {
		return nil, error
	}
	return &route, nil
}

// scanPlaySession は1行分のセッションデータを読み取る。
func scanPlaySession(row scanner) (*domain.PlaySession, error) {
	var (
		sessionName sql.NullString
		routeID     sql.NullString
	)

	session := domain.PlaySession{}
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
func scanMemo(row scanner) (*domain.Memo, error) {
	memo := domain.Memo{}
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
func (repository *Repository) findLatestGame(ctx context.Context, title string, exePath string) (*domain.Game, error) {
	row := repository.connection.QueryRowContext(ctx,
		`SELECT `+gameSelectCols+` FROM "Game" WHERE title = ? AND exePath = ? ORDER BY createdAt DESC LIMIT 1`,
		title, exePath)
	return scanGame(row)
}

// findLatestRoute は直近作成のルートを取得する。
func (repository *Repository) findLatestRoute(ctx context.Context, gameID string, name string) (*domain.Route, error) {
	row := repository.connection.QueryRowContext(ctx,
		`SELECT `+routeSelectCols+` FROM "Route" WHERE gameId = ? AND name = ? ORDER BY createdAt DESC LIMIT 1`,
		gameID, name)
	return scanRoute(row)
}

// findLatestMemo は直近のメモを取得する。
func (repository *Repository) findLatestMemo(ctx context.Context, gameID string, title string) (*domain.Memo, error) {
	row := repository.connection.QueryRowContext(ctx,
		`SELECT `+memoSelectCols+` FROM "Memo" WHERE gameId = ? AND title = ? ORDER BY createdAt DESC LIMIT 1`,
		gameID, title)
	return scanMemo(row)
}

// scanner はScanだけを要求する簡易インターフェース。
type scanner interface {
	Scan(dest ...any) error
}
