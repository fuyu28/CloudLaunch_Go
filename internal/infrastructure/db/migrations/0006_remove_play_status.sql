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
  "currentRouteId" TEXT,
  FOREIGN KEY ("currentRouteId") REFERENCES "Route"("id") ON DELETE SET NULL ON UPDATE CASCADE,
  CHECK ("title" != ''),
  CHECK ("publisher" != ''),
  CHECK ("exePath" != ''),
  CHECK ("totalPlayTime" >= 0)
);

-- playStatus = 'played' だが clearedAt が未設定の行は updatedAt で補完する
INSERT INTO "Game_new" (id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
  localSaveHash, localSaveHashUpdatedAt, totalPlayTime, lastPlayed, clearedAt, currentRouteId)
SELECT id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
  localSaveHash, localSaveHashUpdatedAt, totalPlayTime, lastPlayed,
  CASE WHEN playStatus = 'played' AND clearedAt IS NULL THEN updatedAt ELSE clearedAt END,
  currentRouteId
FROM "Game";

DROP TABLE "Game";

ALTER TABLE "Game_new" RENAME TO "Game";

CREATE INDEX IF NOT EXISTS "idx_games_title" ON "Game"("title");
CREATE INDEX IF NOT EXISTS "idx_games_publisher" ON "Game"("publisher");
CREATE INDEX IF NOT EXISTS "idx_games_last_played_desc" ON "Game"("lastPlayed" DESC);
CREATE INDEX IF NOT EXISTS "idx_games_total_play_time_desc" ON "Game"("totalPlayTime" DESC);
