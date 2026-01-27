// @fileoverview SQL ファイルを使ったシンプルなマイグレーションを実行する。
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// ApplyMigrations は未適用の SQL マイグレーションを順に実行する。
func ApplyMigrations(connection *sql.DB) error {
	if error := ensureSchemaTable(connection); error != nil {
		return error
	}

	entries, error := migrationFiles.ReadDir("migrations")
	if error != nil {
		return error
	}

	fileNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileNames = append(fileNames, entry.Name())
	}
	sort.Strings(fileNames)

	for _, fileName := range fileNames {
		applied, error := isMigrationApplied(connection, fileName)
		if error != nil {
			return error
		}
		if applied {
			continue
		}

		sqlBytes, error := migrationFiles.ReadFile(fmt.Sprintf("migrations/%s", fileName))
		if error != nil {
			return error
		}

		if error := applyMigration(connection, fileName, string(sqlBytes)); error != nil {
			return error
		}
	}

	return nil
}

// ensureSchemaTable はマイグレーション管理テーブルを作成する。
func ensureSchemaTable(connection *sql.DB) error {
	_, error := connection.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id TEXT NOT NULL PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return error
}

// isMigrationApplied は指定マイグレーションが適用済みかを返す。
func isMigrationApplied(connection *sql.DB, fileName string) (bool, error) {
	var count int
	error := connection.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE id = ?`, fileName).Scan(&count)
	if error != nil {
		return false, error
	}
	return count > 0, nil
}

// applyMigration は単一マイグレーションをトランザクションで適用する。
func applyMigration(connection *sql.DB, fileName string, sqlText string) error {
	statements := splitSQLStatements(sqlText)
	transaction, error := connection.Begin()
	if error != nil {
		return error
	}

	for _, statement := range statements {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, error := transaction.Exec(statement); error != nil {
			_ = transaction.Rollback()
			return error
		}
	}

	if _, error := transaction.Exec(`INSERT INTO schema_migrations (id) VALUES (?)`, fileName); error != nil {
		_ = transaction.Rollback()
		return error
	}

	return transaction.Commit()
}

// splitSQLStatements はSQLテキストを簡易的に分割する。
func splitSQLStatements(sqlText string) []string {
	statements := []string{}
	buffer := strings.Builder{}
	inTrigger := false

	for _, r := range sqlText {
		buffer.WriteRune(r)
		if r != ';' {
			continue
		}

		current := strings.TrimSpace(buffer.String())
		if current == "" {
			buffer.Reset()
			continue
		}

		if !inTrigger {
			if strings.Contains(strings.ToUpper(current), "CREATE TRIGGER") {
				inTrigger = true
			}
		}

		if !inTrigger {
			statements = append(statements, current)
			buffer.Reset()
			continue
		}

		// Trigger内はEND;で終了させる
		if strings.HasSuffix(strings.TrimSpace(strings.ToUpper(current)), "END;") {
			statements = append(statements, current)
			buffer.Reset()
			inTrigger = false
			continue
		}

		// 末尾が;でもtrigger内なので継続
	}

	rest := strings.TrimSpace(buffer.String())
	if rest != "" {
		statements = append(statements, rest)
	}

	return statements
}
