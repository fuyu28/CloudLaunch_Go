// Package db は SQLite 接続ヘルパーとマイグレーションを提供する。
package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open は必要なプラグマを設定して SQLite データベースを開く。
func Open(databasePath string) (*sql.DB, error) {
	connection, error := sql.Open("sqlite", fmt.Sprintf("file:%s?_pragma=foreign_keys(1)", databasePath))
	if error != nil {
		return nil, error
	}

	if error := connection.Ping(); error != nil {
		return nil, error
	}

	return connection, nil
}
