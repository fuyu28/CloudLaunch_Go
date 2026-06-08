-- Game テーブルを再作成して以下を適用する:
-- 1. playStatus に CHECK 制約とインデックスを追加
-- 2. playStatus = 'played' かつ clearedAt が未設定の行は updatedAt で補完する
--
-- 【既知の制限】
-- 元の playStatus が 'playing' かつ lastPlayed IS NULL の場合（手動設定状態）は
-- 復元不能なため 'unplayed' になる。アプリ未配布のため許容する。

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
  localSaveHash, localSaveHashUpdatedAt, totalPlayTime, lastPlayed,
  CASE WHEN playStatus = 'played' AND clearedAt IS NULL THEN updatedAt ELSE clearedAt END,
  CASE
    WHEN playStatus IN ('unplayed', 'playing', 'played') THEN playStatus
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
