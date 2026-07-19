-- Push のリモート HEAD 更新とローカル baseline 確定の間をまたぐ障害向けの保留記録。
-- HEAD 書き換え前に書き込み、baseline 確定と同時に削除する。
-- Game 削除時は一緒に消す（FK CASCADE）。Game が無い孤立 pending は意味を持たない。
CREATE TABLE IF NOT EXISTS "PendingPush" (
  "gameId" TEXT NOT NULL PRIMARY KEY,
  "expectedRemoteHead" TEXT NOT NULL DEFAULT '',
  "newCommitHash" TEXT NOT NULL,
  "contentFingerprint" TEXT NOT NULL,
  "saveTree" TEXT NOT NULL,
  "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY ("gameId") REFERENCES "Game"("id") ON DELETE CASCADE ON UPDATE CASCADE,
  CHECK ("newCommitHash" != ''),
  CHECK ("contentFingerprint" != '')
);
