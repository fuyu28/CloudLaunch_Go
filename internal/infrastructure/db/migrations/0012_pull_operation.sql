-- Pull のセーブディレクトリ交換（stage/backup rename）と DB 反映をまたぐ障害向けジャーナル。
-- PREPARED: ディスク交換直前〜DB 未反映。起動回復では旧 live を復元する。
-- APPLIED: DB 反映済み。backup 削除と journal 消去が残作業。
-- hadLive: 交換前に live が存在したか。無かった場合の PREPARED 復旧で新 live を捨てる判定に使う。
-- Game 削除時は一緒に消す（FK CASCADE）。
CREATE TABLE IF NOT EXISTS "PullOperation" (
  "operationId" TEXT NOT NULL PRIMARY KEY,
  "gameId" TEXT NOT NULL,
  "livePath" TEXT NOT NULL,
  "stagePath" TEXT NOT NULL,
  "backupPath" TEXT NOT NULL,
  "commitHash" TEXT NOT NULL,
  "status" TEXT NOT NULL,
  "hadLive" INTEGER NOT NULL DEFAULT 0,
  "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY ("gameId") REFERENCES "Game"("id") ON DELETE CASCADE ON UPDATE CASCADE,
  CHECK ("operationId" != ''),
  CHECK ("livePath" != ''),
  CHECK ("stagePath" != ''),
  CHECK ("backupPath" != ''),
  CHECK ("commitHash" != ''),
  CHECK ("status" IN ('PREPARED', 'APPLIED')),
  CHECK ("hadLive" IN (0, 1))
);

CREATE INDEX IF NOT EXISTS "idx_pull_operation_gameid" ON "PullOperation"("gameId");
CREATE INDEX IF NOT EXISTS "idx_pull_operation_status" ON "PullOperation"("status");
