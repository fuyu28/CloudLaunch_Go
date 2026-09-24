-- Game 削除後にローカルメモディレクトリの削除が完了していないゲームIDを保持する。
-- Game 本体は同一トランザクションで削除されるため、外部キーは設定しない。
CREATE TABLE IF NOT EXISTS "PendingMemoCleanup" (
  "gameId" TEXT NOT NULL PRIMARY KEY,
  "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
