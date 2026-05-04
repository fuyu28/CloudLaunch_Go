ALTER TABLE "Game" RENAME TO "Game_old";

CREATE TABLE "Game" (
  "id" TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
  "title" TEXT NOT NULL,
  "publisher" TEXT NOT NULL,
  "imagePath" TEXT,
  "exePath" TEXT NOT NULL UNIQUE,
  "saveFolderPath" TEXT,
  "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "localSaveHash" TEXT,
  "localSaveHashUpdatedAt" DATETIME,
  "playStatus" TEXT NOT NULL DEFAULT 'unplayed',
  "totalPlayTime" INTEGER NOT NULL DEFAULT 0,
  "lastPlayed" DATETIME,
  "clearedAt" DATETIME,
  CHECK ("playStatus" IN ('unplayed', 'playing', 'played')),
  CHECK ("totalPlayTime" >= 0)
);

INSERT INTO "Game" (
  id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
  localSaveHash, localSaveHashUpdatedAt, playStatus, totalPlayTime, lastPlayed, clearedAt
)
SELECT
  id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
  localSaveHash, localSaveHashUpdatedAt, playStatus, totalPlayTime, lastPlayed, clearedAt
FROM "Game_old";

DROP TABLE "Game_old";

CREATE INDEX IF NOT EXISTS "idx_games_title" ON "Game"("title");
CREATE INDEX IF NOT EXISTS "idx_games_publisher" ON "Game"("publisher");
CREATE INDEX IF NOT EXISTS "idx_games_play_status" ON "Game"("playStatus");
CREATE INDEX IF NOT EXISTS "idx_games_last_played" ON "Game"("lastPlayed");
CREATE INDEX IF NOT EXISTS "idx_games_created_at" ON "Game"("createdAt");

DROP TRIGGER IF EXISTS "trigger_game_updated_at";

CREATE TRIGGER IF NOT EXISTS "trigger_game_updated_at"
AFTER UPDATE ON "Game"
FOR EACH ROW
BEGIN
  UPDATE "Game" SET "updatedAt" = CURRENT_TIMESTAMP WHERE "id" = OLD."id";
END;

ALTER TABLE "PlaySession" RENAME TO "PlaySession_old";

CREATE TABLE "PlaySession" (
  "id" TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
  "gameId" TEXT NOT NULL,
  "playedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "duration" INTEGER NOT NULL,
  "updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY ("gameId") REFERENCES "Game"("id") ON DELETE CASCADE ON UPDATE CASCADE,
  CHECK ("duration" >= 0)
);

INSERT INTO "PlaySession" (
  id, gameId, playedAt, duration, updatedAt
)
SELECT
  id, gameId, playedAt, duration, updatedAt
FROM "PlaySession_old";

DROP TABLE "PlaySession_old";

CREATE INDEX IF NOT EXISTS "idx_playsessions_gameid_played_at" ON "PlaySession"("gameId", "playedAt");
CREATE INDEX IF NOT EXISTS "idx_playsessions_played_at_desc" ON "PlaySession"("playedAt" DESC);

DROP TRIGGER IF EXISTS "trigger_play_session_updated_at";

CREATE TRIGGER IF NOT EXISTS "trigger_play_session_updated_at"
AFTER UPDATE ON "PlaySession"
FOR EACH ROW
BEGIN
  UPDATE "PlaySession" SET "updatedAt" = CURRENT_TIMESTAMP WHERE "id" = OLD."id";
END;

DROP INDEX IF EXISTS "idx_chapters_gameid_order";
DROP INDEX IF EXISTS "idx_chapters_name";
DROP TABLE IF EXISTS "Chapter";
