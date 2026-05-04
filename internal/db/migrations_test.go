package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestApplyMigrationsBuildsCurrentSchema(t *testing.T) {
	t.Parallel()

	connection := openMigratedTestDB(t)
	defer func() { _ = connection.Close() }()

	assertTableExists(t, connection, "Game")
	assertTableExists(t, connection, "PlaySession")
	assertTableExists(t, connection, "PlayRoute")
	assertTableExists(t, connection, "Memo")

	assertTableNotExists(t, connection, "Upload")
	assertTableNotExists(t, connection, "Chapter")

	assertTableColumns(t, connection, "Game", []string{
		"id", "title", "publisher", "imagePath", "exePath", "saveFolderPath",
		"createdAt", "updatedAt", "localSaveHash", "localSaveHashUpdatedAt",
		"totalPlayTime", "lastPlayed", "clearedAt",
	})
	assertTableColumns(t, connection, "PlaySession", []string{
		"id", "gameId", "playRouteId", "playedAt", "duration", "updatedAt",
	})
	assertTableColumns(t, connection, "PlayRoute", []string{
		"id", "gameId", "name", "sortOrder", "createdAt",
	})

	assertColumnMissing(t, connection, "Game", "playStatus")
	assertColumnMissing(t, connection, "Game", "currentChapter")
	assertColumnMissing(t, connection, "PlaySession", "sessionName")
	assertColumnMissing(t, connection, "PlaySession", "chapterId")
	assertColumnMissing(t, connection, "PlaySession", "uploadId")

	assertPlaySessionForeignKey(t, connection, "Game")
}

func TestApplyMigrationsIsIdempotent(t *testing.T) {
	t.Parallel()

	connection := openMigratedTestDB(t)
	defer func() { _ = connection.Close() }()

	if err := ApplyMigrations(connection); err != nil {
		t.Fatalf("expected migrations to be idempotent, got %v", err)
	}
}

func TestApplyMigrationsUpgradesLegacySchema(t *testing.T) {
	t.Parallel()

	connection, err := Open(filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatalf("failed to open legacy test database: %v", err)
	}
	defer func() { _ = connection.Close() }()

	seedLegacySchema(t, connection)

	if err := ApplyMigrations(connection); err != nil {
		t.Fatalf("expected legacy schema upgrade to succeed, got %v", err)
	}

	assertTableExists(t, connection, "Game")
	assertTableExists(t, connection, "PlaySession")
	assertTableExists(t, connection, "PlayRoute")
	assertTableExists(t, connection, "Memo")
	assertTableNotExists(t, connection, "Upload")
	assertTableNotExists(t, connection, "Chapter")

	assertColumnMissing(t, connection, "Game", "playStatus")
	assertColumnMissing(t, connection, "Game", "currentChapter")
	assertColumnMissing(t, connection, "PlaySession", "sessionName")
	assertColumnMissing(t, connection, "PlaySession", "chapterId")
	assertColumnMissing(t, connection, "PlaySession", "uploadId")
	assertPlaySessionForeignKey(t, connection, "Game")
}

func openMigratedTestDB(t *testing.T) *sql.DB {
	t.Helper()

	connection, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := ApplyMigrations(connection); err != nil {
		_ = connection.Close()
		t.Fatalf("failed to apply migrations: %v", err)
	}
	return connection
}

func assertTableExists(t *testing.T, connection *sql.DB, name string) {
	t.Helper()
	if !tableExists(t, connection, name) {
		t.Fatalf("expected table %q to exist", name)
	}
}

func assertTableNotExists(t *testing.T, connection *sql.DB, name string) {
	t.Helper()
	if tableExists(t, connection, name) {
		t.Fatalf("expected table %q to be absent", name)
	}
}

func tableExists(t *testing.T, connection *sql.DB, name string) bool {
	t.Helper()

	var count int
	if err := connection.QueryRow(
		`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`,
		name,
	).Scan(&count); err != nil {
		t.Fatalf("failed to inspect sqlite_master: %v", err)
	}
	return count > 0
}

func assertTableColumns(t *testing.T, connection *sql.DB, table string, expected []string) {
	t.Helper()

	actual := readColumns(t, connection, table)
	if len(actual) != len(expected) {
		t.Fatalf("expected columns %v for %s, got %v", expected, table, actual)
	}
	for index, column := range expected {
		if actual[index] != column {
			t.Fatalf("expected columns %v for %s, got %v", expected, table, actual)
		}
	}
}

func assertColumnMissing(t *testing.T, connection *sql.DB, table string, column string) {
	t.Helper()

	for _, current := range readColumns(t, connection, table) {
		if current == column {
			t.Fatalf("expected column %q to be absent from %s", column, table)
		}
	}
}

func readColumns(t *testing.T, connection *sql.DB, table string) []string {
	t.Helper()

	rows, err := connection.Query(`PRAGMA table_info("` + table + `")`)
	if err != nil {
		t.Fatalf("failed to inspect columns for %s: %v", table, err)
	}
	defer func() { _ = rows.Close() }()

	columns := make([]string, 0)
	for rows.Next() {
		var (
			cid        int
			name       string
			dataType   string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &primaryKey); err != nil {
			t.Fatalf("failed to read column metadata for %s: %v", table, err)
		}
		columns = append(columns, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate column metadata for %s: %v", table, err)
	}
	return columns
}

func assertPlaySessionForeignKey(t *testing.T, connection *sql.DB, expectedTable string) {
	t.Helper()

	rows, err := connection.Query(`PRAGMA foreign_key_list("PlaySession")`)
	if err != nil {
		t.Fatalf("failed to inspect foreign keys: %v", err)
	}
	defer func() { _ = rows.Close() }()

	found := false
	for rows.Next() {
		var (
			id       int
			seq      int
			table    string
			from     string
			to       string
			onUpdate string
			onDelete string
			match    string
		)
		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			t.Fatalf("failed to read foreign key metadata: %v", err)
		}
		if from == "gameId" {
			found = true
			if table != expectedTable {
				t.Fatalf("expected PlaySession.gameId to reference %q, got %q", expectedTable, table)
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate foreign key metadata: %v", err)
	}
	if !found {
		t.Fatal("expected PlaySession.gameId foreign key to exist")
	}
}

func seedLegacySchema(t *testing.T, connection *sql.DB) {
	t.Helper()

	statements := []string{
		`CREATE TABLE schema_migrations (
			id TEXT NOT NULL PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`INSERT INTO schema_migrations (id) VALUES
			('0001_init.sql'),
			('0002_add_updated_at.sql'),
			('0003_add_local_save_hash.sql'),
			('0004_remove_upload.sql'),
			('0005_remove_session_name.sql');`,
		`CREATE TABLE "Game" (
			"id" TEXT NOT NULL PRIMARY KEY,
			"title" TEXT NOT NULL,
			"publisher" TEXT NOT NULL,
			"imagePath" TEXT,
			"exePath" TEXT NOT NULL UNIQUE,
			"saveFolderPath" TEXT,
			"createdAt" DATETIME NOT NULL,
			"updatedAt" DATETIME NOT NULL,
			"localSaveHash" TEXT,
			"localSaveHashUpdatedAt" DATETIME,
			"playStatus" TEXT NOT NULL DEFAULT 'unplayed',
			"currentChapter" TEXT,
			"totalPlayTime" INTEGER NOT NULL DEFAULT 0,
			"lastPlayed" DATETIME,
			"clearedAt" DATETIME
		);`,
		`CREATE TABLE "Upload" (
			"id" TEXT NOT NULL PRIMARY KEY,
			"title" TEXT NOT NULL,
			"createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE "Chapter" (
			"id" TEXT NOT NULL PRIMARY KEY,
			"gameId" TEXT NOT NULL,
			"name" TEXT NOT NULL,
			"sortOrder" INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE "PlaySession" (
			"id" TEXT NOT NULL PRIMARY KEY,
			"gameId" TEXT NOT NULL,
			"sessionName" TEXT,
			"chapterId" TEXT,
			"uploadId" TEXT,
			"playedAt" DATETIME NOT NULL,
			"duration" INTEGER NOT NULL,
			"updatedAt" DATETIME NOT NULL
		);`,
		`CREATE TABLE "Memo" (
			"id" TEXT NOT NULL PRIMARY KEY,
			"title" TEXT NOT NULL,
			"content" TEXT NOT NULL,
			"gameId" TEXT NOT NULL,
			"createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			"updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX "idx_chapters_gameid_order" ON "Chapter"("gameId", "sortOrder");`,
		`CREATE INDEX "idx_chapters_name" ON "Chapter"("name");`,
		`INSERT INTO "Game" (
			id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
			localSaveHash, localSaveHashUpdatedAt, playStatus, currentChapter, totalPlayTime, lastPlayed, clearedAt
		) VALUES (
			'game-1', 'Legacy Game', 'Legacy Brand', NULL, '/games/legacy.exe', NULL,
			'2026-05-01 10:00:00', '2026-05-02 11:00:00', NULL, NULL, 'played', 'chapter-1', 3600,
			'2026-05-02 12:00:00', '2026-05-03 00:00:00'
		);`,
		`INSERT INTO "Upload" (id, title, createdAt) VALUES ('upload-1', 'old upload', '2026-05-01 10:00:00');`,
		`INSERT INTO "Chapter" (id, gameId, name, sortOrder) VALUES ('chapter-1', 'game-1', 'Common', 0);`,
		`INSERT INTO "PlaySession" (
			id, gameId, sessionName, chapterId, uploadId, playedAt, duration, updatedAt
		) VALUES (
			'session-1', 'game-1', 'legacy session', 'chapter-1', 'upload-1',
			'2026-05-02 11:00:00', 1800, '2026-05-02 11:30:00'
		);`,
		`INSERT INTO "Memo" (id, title, content, gameId, createdAt, updatedAt) VALUES (
			'memo-1', 'Legacy Memo', 'memo body', 'game-1', '2026-05-01 10:00:00', '2026-05-01 10:00:00'
		);`,
	}

	for _, statement := range statements {
		if _, err := connection.Exec(statement); err != nil {
			t.Fatalf("failed to seed legacy schema: %v\nstatement: %s", err, statement)
		}
	}
}
