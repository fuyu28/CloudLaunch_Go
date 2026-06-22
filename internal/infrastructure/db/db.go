// Package db は SQLite 接続ヘルパーとマイグレーションを提供する。
package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open は必要なプラグマを設定して SQLite データベースを開く。
//
// busy_timeout: ApplyPullResult 等の複数文の書き込みトランザクションと、バックグラウンドの
// 自動 Push・別 Pull が別コネクションで競合したとき、即 SQLITE_BUSY（"database is
// locked"）で失敗せず一定時間（5秒）待機・リトライさせる。
// （journal_mode=WAL は読み書き並行性を高められるが、フルバックアップがDBファイルの
//  コピー方式のため WAL 化するとチェックポイント未済データを取りこぼす。ここでは採用しない。）
func Open(databasePath string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)",
		databasePath,
	)
	connection, error := sql.Open("sqlite", dsn)
	if error != nil {
		return nil, error
	}

	if error := connection.Ping(); error != nil {
		return nil, error
	}

	return connection, nil
}
