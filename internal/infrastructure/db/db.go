// Package db provides SQLite connection helpers and migrations.
package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open opens a SQLite database with required pragmas.
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
