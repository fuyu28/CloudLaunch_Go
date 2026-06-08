-- playStatus 列を手動管理方式として復元する。
--
-- 【既知の制限】
-- 0006 で playStatus 列を削除済みのため、元の値を完全には復元できない。
-- 旧DBで playStatus = 'playing' かつ lastPlayed IS NULL だった行（手動設定状態）は
-- 復元不能であり、本マイグレーション適用後は 'unplayed' になる。
-- アプリ未配布のため、本制限は許容する。

CREATE TABLE "Game_new" (
  "id" TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
  "title" TEXT NOT NULL,
  "publisher" TEXT NOT NULL,
  "imagePath" TEXT,
  "exePath" TEXT NOT NULL,
  "saveFolderPath" TEXT,
  "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "localSaveHash" TEXT,
  "localSaveHashUpdatedAt" DATETIME,
  "totalPlayTime" INTEGER NOT NULL DEFAULT 0,
  "lastPlayed" DATETIME,
  "clearedAt" DATETIME,
  "playStatus" TEXT NOT NULL DEFAULT 'unplayed'
    CHECK ("playStatus" IN ('unplayed', 'playing', 'played')),
  "currentRouteId" TEXT,
  FOREIGN KEY ("currentRouteId") REFERENCES "Route"("id") ON DELETE SET NULL ON UPDATE CASCADE,
  CHECK ("title" != ''),
  CHECK ("publisher" != ''),
  CHECK ("exePath" != ''),
  CHECK ("totalPlayTime" >= 0)
);

INSERT INTO "Game_new" (id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
  localSaveHash, localSaveHashUpdatedAt, totalPlayTime, lastPlayed, clearedAt, playStatus, currentRouteId)
SELECT id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
  localSaveHash, localSaveHashUpdatedAt, totalPlayTime, lastPlayed, clearedAt,
  CASE
    WHEN clearedAt IS NOT NULL THEN 'played'
    WHEN lastPlayed IS NOT NULL THEN 'playing'
    ELSE 'unplayed'
  END,
  currentRouteId
FROM "Game";

DROP TABLE "Game";

ALTER TABLE "Game_new" RENAME TO "Game";

CREATE INDEX IF NOT EXISTS "idx_games_title" ON "Game"("title");
CREATE INDEX IF NOT EXISTS "idx_games_publisher" ON "Game"("publisher");
CREATE INDEX IF NOT EXISTS "idx_games_last_played_desc" ON "Game"("lastPlayed" DESC);
CREATE INDEX IF NOT EXISTS "idx_games_total_play_time_desc" ON "Game"("totalPlayTime" DESC);
CREATE INDEX IF NOT EXISTS "idx_games_play_status" ON "Game"("playStatus");
