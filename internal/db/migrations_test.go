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
	assertTableExists(t, connection, "Memo")

	assertTableNotExists(t, connection, "Upload")
	assertTableNotExists(t, connection, "Chapter")

	assertTableColumns(t, connection, "Game", []string{
		"id", "title", "publisher", "imagePath", "exePath", "saveFolderPath",
		"createdAt", "updatedAt", "localSaveHash", "localSaveHashUpdatedAt",
		"totalPlayTime", "lastPlayed", "clearedAt",
	})
	assertTableColumns(t, connection, "PlaySession", []string{
		"id", "gameId", "playedAt", "duration", "updatedAt",
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
