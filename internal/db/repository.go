// @fileoverview データベース操作の基本CRUDを提供する。
package db

import (
	"context"
	"database/sql"
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
		SELECT id, title, publisher, imagePath, exePath, saveFolderPath, createdAt,
		       playStatus, totalPlayTime, lastPlayed, clearedAt, currentChapter
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

// ListGames は検索・フィルタ・ソート付きでゲームを取得する。
func (repository *Repository) ListGames(
	ctx context.Context,
	searchText string,
	filter models.PlayStatus,
	sortBy string,
	sortDirection string,
) ([]models.Game, error) {
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT id, title, publisher, imagePath, exePath, saveFolderPath, createdAt,
		       playStatus, totalPlayTime, lastPlayed, clearedAt, currentChapter
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
		whereClauses = append(whereClauses, "playStatus = ?")
		args = append(args, filter)
	}
	if len(whereClauses) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(whereClauses, " AND "))
	}

	orderBy := normalizeSortColumn(sortBy)
	direction := normalizeSortDirection(sortDirection)
	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY %s %s", orderBy, direction))

	rows, error := repository.connection.QueryContext(ctx, queryBuilder.String(), args...)
	if error != nil {
		return nil, error
	}
	defer rows.Close()

	games := make([]models.Game, 0)
	for rows.Next() {
		game, error := scanGame(rows)
		if error != nil {
			return nil, error
		}
		games = append(games, *game)
	}
	if error := rows.Err(); error != nil {
		return nil, error
	}

	return games, nil
}

// CreateGame はゲームを作成して返す。
func (repository *Repository) CreateGame(ctx context.Context, game models.Game) (*models.Game, error) {
	_, error := repository.connection.ExecContext(ctx, `
		INSERT INTO "Game" (title, publisher, imagePath, exePath, saveFolderPath, playStatus, totalPlayTime, lastPlayed, clearedAt, currentChapter)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, game.Title, game.Publisher, game.ImagePath, game.ExePath, game.SaveFolderPath,
		game.PlayStatus, game.TotalPlayTime, game.LastPlayed, game.ClearedAt, game.CurrentChapter)
	if error != nil {
		return nil, error
	}

	return repository.findLatestGame(ctx, game.Title, game.ExePath)
}

// UpdateGame はゲームを更新して返す。
func (repository *Repository) UpdateGame(ctx context.Context, game models.Game) (*models.Game, error) {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "Game" SET title = ?, publisher = ?, imagePath = ?, exePath = ?, saveFolderPath = ?,
			playStatus = ?, totalPlayTime = ?, lastPlayed = ?, clearedAt = ?, currentChapter = ?
		WHERE id = ?
	`, game.Title, game.Publisher, game.ImagePath, game.ExePath, game.SaveFolderPath,
		game.PlayStatus, game.TotalPlayTime, game.LastPlayed, game.ClearedAt, game.CurrentChapter, game.ID)
	if error != nil {
		return nil, error
	}

	return repository.GetGameByID(ctx, game.ID)
}

// DeleteGame はゲームを削除する。
func (repository *Repository) DeleteGame(ctx context.Context, gameID string) error {
	_, error := repository.connection.ExecContext(ctx, `DELETE FROM "Game" WHERE id = ?`, gameID)
	return error
}

// CreateChapter は章を作成して返す。
func (repository *Repository) CreateChapter(ctx context.Context, chapter models.Chapter) (*models.Chapter, error) {
	_, error := repository.connection.ExecContext(ctx, `
		INSERT INTO "Chapter" (name, "order", gameId)
		VALUES (?, ?, ?)
	`, chapter.Name, chapter.Order, chapter.GameID)
	if error != nil {
		return nil, error
	}

	return repository.findLatestChapter(ctx, chapter.GameID, chapter.Name)
}

// ListChaptersByGame はゲームIDで章一覧を取得する。
func (repository *Repository) ListChaptersByGame(ctx context.Context, gameID string) ([]models.Chapter, error) {
	rows, error := repository.connection.QueryContext(ctx, `
		SELECT id, name, "order", gameId, createdAt
		FROM "Chapter" WHERE gameId = ? ORDER BY "order" ASC
	`, gameID)
	if error != nil {
		return nil, error
	}
	defer rows.Close()

	chapters := make([]models.Chapter, 0)
	for rows.Next() {
		chapter, error := scanChapter(rows)
		if error != nil {
			return nil, error
		}
		chapters = append(chapters, *chapter)
	}
	if error := rows.Err(); error != nil {
		return nil, error
	}

	return chapters, nil
}

// UpdateChapter は章を更新して返す。
func (repository *Repository) UpdateChapter(ctx context.Context, chapter models.Chapter) (*models.Chapter, error) {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "Chapter" SET name = ?, "order" = ? WHERE id = ?
	`, chapter.Name, chapter.Order, chapter.ID)
	if error != nil {
		return nil, error
	}
	return repository.GetChapterByID(ctx, chapter.ID)
}

// UpdateChapterOrder は章の順序を更新する。
func (repository *Repository) UpdateChapterOrder(ctx context.Context, chapterID string, order int64) error {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "Chapter" SET "order" = ? WHERE id = ?
	`, order, chapterID)
	return error
}

// GetChapterByID は章IDで章を取得する。
func (repository *Repository) GetChapterByID(ctx context.Context, chapterID string) (*models.Chapter, error) {
	row := repository.connection.QueryRowContext(ctx, `
		SELECT id, name, "order", gameId, createdAt FROM "Chapter" WHERE id = ?
	`, chapterID)

	chapter, error := scanChapter(row)
	if error == sql.ErrNoRows {
		return nil, nil
	}
	if error != nil {
		return nil, error
	}
	return chapter, nil
}

// DeleteChapter は章を削除する。
func (repository *Repository) DeleteChapter(ctx context.Context, chapterID string) error {
	_, error := repository.connection.ExecContext(ctx, `DELETE FROM "Chapter" WHERE id = ?`, chapterID)
	return error
}

// CreatePlaySession はプレイセッションを作成して返す。
func (repository *Repository) CreatePlaySession(ctx context.Context, session models.PlaySession) (*models.PlaySession, error) {
	_, error := repository.connection.ExecContext(ctx, `
		INSERT INTO "PlaySession" (gameId, playedAt, duration, sessionName, chapterId, uploadId)
		VALUES (?, ?, ?, ?, ?, ?)
	`, session.GameID, session.PlayedAt, session.Duration, session.SessionName, session.ChapterID, session.UploadID)
	if error != nil {
		return nil, error
	}

	return repository.findLatestPlaySession(ctx, session.GameID)
}

// ListPlaySessionsByGame はゲームIDでセッション一覧を取得する。
func (repository *Repository) ListPlaySessionsByGame(ctx context.Context, gameID string) ([]models.PlaySession, error) {
	rows, error := repository.connection.QueryContext(ctx, `
		SELECT id, gameId, playedAt, duration, sessionName, chapterId, uploadId
		FROM "PlaySession" WHERE gameId = ? ORDER BY playedAt DESC
	`, gameID)
	if error != nil {
		return nil, error
	}
	defer rows.Close()

	sessions := make([]models.PlaySession, 0)
	for rows.Next() {
		session, error := scanPlaySession(rows)
		if error != nil {
			return nil, error
		}
		sessions = append(sessions, *session)
	}
	if error := rows.Err(); error != nil {
		return nil, error
	}

	return sessions, nil
}

// DeletePlaySession はセッションを削除する。
func (repository *Repository) DeletePlaySession(ctx context.Context, sessionID string) error {
	_, error := repository.connection.ExecContext(ctx, `DELETE FROM "PlaySession" WHERE id = ?`, sessionID)
	return error
}

// UpdatePlaySessionChapter はセッションの章を更新する。
func (repository *Repository) UpdatePlaySessionChapter(ctx context.Context, sessionID string, chapterID *string) error {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "PlaySession" SET chapterId = ? WHERE id = ?
	`, chapterID, sessionID)
	return error
}

// UpdatePlaySessionName はセッション名を更新する。
func (repository *Repository) UpdatePlaySessionName(ctx context.Context, sessionID string, sessionName string) error {
	_, error := repository.connection.ExecContext(ctx, `
		UPDATE "PlaySession" SET sessionName = ? WHERE id = ?
	`, sessionName, sessionID)
	return error
}

// CreateUpload はアップロード履歴を作成して返す。
func (repository *Repository) CreateUpload(ctx context.Context, upload models.Upload) (*models.Upload, error) {
	_, error := repository.connection.ExecContext(ctx, `
		INSERT INTO "Upload" (clientId, comment, gameId)
		VALUES (?, ?, ?)
	`, upload.ClientID, upload.Comment, upload.GameID)
	if error != nil {
		return nil, error
	}

	return repository.findLatestUpload(ctx, upload.GameID)
}

// ListUploadsByGame はゲームIDでアップロード一覧を取得する。
func (repository *Repository) ListUploadsByGame(ctx context.Context, gameID string) ([]models.Upload, error) {
	rows, error := repository.connection.QueryContext(ctx, `
		SELECT id, clientId, comment, createdAt, gameId
		FROM "Upload" WHERE gameId = ? ORDER BY createdAt DESC
	`, gameID)
	if error != nil {
		return nil, error
	}
	defer rows.Close()

	uploads := make([]models.Upload, 0)
	for rows.Next() {
		upload, error := scanUpload(rows)
		if error != nil {
			return nil, error
		}
		uploads = append(uploads, *upload)
	}
	if error := rows.Err(); error != nil {
		return nil, error
	}

	return uploads, nil
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
func (repository *Repository) ListMemosByGame(ctx context.Context, gameID string) ([]models.Memo, error) {
	rows, error := repository.connection.QueryContext(ctx, `
		SELECT id, title, content, gameId, createdAt, updatedAt
		FROM "Memo" WHERE gameId = ? ORDER BY updatedAt DESC
	`, gameID)
	if error != nil {
		return nil, error
	}
	defer rows.Close()

	memos := make([]models.Memo, 0)
	for rows.Next() {
		memo, error := scanMemo(rows)
		if error != nil {
			return nil, error
		}
		memos = append(memos, *memo)
	}
	if error := rows.Err(); error != nil {
		return nil, error
	}

	return memos, nil
}

// GetChapterStats は章ごとの統計を取得する。
func (repository *Repository) GetChapterStats(ctx context.Context, gameID string) ([]models.ChapterStat, error) {
	rows, error := repository.connection.QueryContext(ctx, `
		SELECT c.id, c.name, c."order",
		       COALESCE(SUM(ps.duration), 0) as total_time,
		       COUNT(ps.id) as session_count
		FROM "Chapter" c
		LEFT JOIN "PlaySession" ps ON ps.chapterId = c.id
		WHERE c.gameId = ?
		GROUP BY c.id, c.name, c."order"
		ORDER BY c."order" ASC
	`, gameID)
	if error != nil {
		return nil, error
	}
	defer rows.Close()

	stats := make([]models.ChapterStat, 0)
	for rows.Next() {
		var (
			chapterID    string
			chapterName  string
			orderValue   int64
			totalTime    int64
			sessionCount int64
		)
		if error := rows.Scan(&chapterID, &chapterName, &orderValue, &totalTime, &sessionCount); error != nil {
			return nil, error
		}
		average := float64(0)
		if sessionCount > 0 {
			average = float64(totalTime) / float64(sessionCount)
		}
		stats = append(stats, models.ChapterStat{
			ChapterID:    chapterID,
			ChapterName:  chapterName,
			TotalTime:    totalTime,
			SessionCount: sessionCount,
			AverageTime:  average,
			Order:        orderValue,
		})
	}
	if error := rows.Err(); error != nil {
		return nil, error
	}
	return stats, nil
}

// ListAllMemos は全メモを取得する。
func (repository *Repository) ListAllMemos(ctx context.Context) ([]models.Memo, error) {
	rows, error := repository.connection.QueryContext(ctx, `
		SELECT id, title, content, gameId, createdAt, updatedAt
		FROM "Memo" ORDER BY updatedAt DESC
	`)
	if error != nil {
		return nil, error
	}
	defer rows.Close()

	memos := make([]models.Memo, 0)
	for rows.Next() {
		memo, error := scanMemo(rows)
		if error != nil {
			return nil, error
		}
		memos = append(memos, *memo)
	}
	if error := rows.Err(); error != nil {
		return nil, error
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
		imagePath      sql.NullString
		saveFolderPath sql.NullString
		lastPlayed     sql.NullTime
		clearedAt      sql.NullTime
		currentChapter sql.NullString
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
		&game.PlayStatus,
		&game.TotalPlayTime,
		&lastPlayed,
		&clearedAt,
		&currentChapter,
	)
	if error != nil {
		return nil, error
	}

	game.ImagePath = nullStringPtr(imagePath)
	game.SaveFolderPath = nullStringPtr(saveFolderPath)
	game.LastPlayed = nullTimePtr(lastPlayed)
	game.ClearedAt = nullTimePtr(clearedAt)
	game.CurrentChapter = nullStringPtr(currentChapter)

	return &game, nil
}

// scanChapter は1行分の章データを読み取る。
func scanChapter(row scanner) (*models.Chapter, error) {
	chapter := models.Chapter{}
	error := row.Scan(&chapter.ID, &chapter.Name, &chapter.Order, &chapter.GameID, &chapter.CreatedAt)
	if error != nil {
		return nil, error
	}
	return &chapter, nil
}

// scanPlaySession は1行分のセッションデータを読み取る。
func scanPlaySession(row scanner) (*models.PlaySession, error) {
	var (
		sessionName sql.NullString
		chapterID   sql.NullString
		uploadID    sql.NullString
	)

	session := models.PlaySession{}
	error := row.Scan(
		&session.ID,
		&session.GameID,
		&session.PlayedAt,
		&session.Duration,
		&sessionName,
		&chapterID,
		&uploadID,
	)
	if error != nil {
		return nil, error
	}

	session.SessionName = nullStringPtr(sessionName)
	session.ChapterID = nullStringPtr(chapterID)
	session.UploadID = nullStringPtr(uploadID)

	return &session, nil
}

// scanUpload は1行分のアップロードデータを読み取る。
func scanUpload(row scanner) (*models.Upload, error) {
	var clientID sql.NullString

	upload := models.Upload{}
	error := row.Scan(&upload.ID, &clientID, &upload.Comment, &upload.CreatedAt, &upload.GameID)
	if error != nil {
		return nil, error
	}

	upload.ClientID = nullStringPtr(clientID)
	return &upload, nil
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
		SELECT id, title, publisher, imagePath, exePath, saveFolderPath, createdAt,
		       playStatus, totalPlayTime, lastPlayed, clearedAt, currentChapter
		FROM "Game" WHERE title = ? AND exePath = ? ORDER BY createdAt DESC LIMIT 1
	`, title, exePath)
	return scanGame(row)
}

// findLatestChapter は直近作成の章を取得する。
func (repository *Repository) findLatestChapter(ctx context.Context, gameID string, name string) (*models.Chapter, error) {
	row := repository.connection.QueryRowContext(ctx, `
		SELECT id, name, "order", gameId, createdAt
		FROM "Chapter" WHERE gameId = ? AND name = ? ORDER BY createdAt DESC LIMIT 1
	`, gameID, name)
	return scanChapter(row)
}

// findLatestPlaySession は直近のプレイセッションを取得する。
func (repository *Repository) findLatestPlaySession(ctx context.Context, gameID string) (*models.PlaySession, error) {
	row := repository.connection.QueryRowContext(ctx, `
		SELECT id, gameId, playedAt, duration, sessionName, chapterId, uploadId
		FROM "PlaySession" WHERE gameId = ? ORDER BY playedAt DESC LIMIT 1
	`, gameID)
	return scanPlaySession(row)
}

// findLatestUpload は直近のアップロード履歴を取得する。
func (repository *Repository) findLatestUpload(ctx context.Context, gameID string) (*models.Upload, error) {
	row := repository.connection.QueryRowContext(ctx, `
		SELECT id, clientId, comment, createdAt, gameId
		FROM "Upload" WHERE gameId = ? ORDER BY createdAt DESC LIMIT 1
	`, gameID)
	return scanUpload(row)
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
