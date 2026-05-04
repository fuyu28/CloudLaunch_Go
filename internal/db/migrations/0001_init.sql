PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS "Game" (
  "id" TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
  "title" TEXT NOT NULL,
  "publisher" TEXT NOT NULL,
  "imagePath" TEXT,
  "exePath" TEXT NOT NULL,
  "saveFolderPath" TEXT,
  "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "totalPlayTime" INTEGER NOT NULL DEFAULT 0,
  "lastPlayed" DATETIME,
  "clearedAt" DATETIME,
  CHECK ("title" != ''),
  CHECK ("publisher" != ''),
  CHECK ("exePath" != ''),
  CHECK ("totalPlayTime" >= 0)
);

CREATE TABLE IF NOT EXISTS "PlaySession" (
  "id" TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
  "gameId" TEXT NOT NULL,
  "playedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "duration" INTEGER NOT NULL,
  FOREIGN KEY ("gameId") REFERENCES "Game"("id") ON DELETE CASCADE ON UPDATE CASCADE,
  CHECK ("duration" >= 0)
);

CREATE TABLE IF NOT EXISTS "Memo" (
  "id" TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
  "title" TEXT NOT NULL,
  "content" TEXT NOT NULL,
  "gameId" TEXT NOT NULL,
  "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY ("gameId") REFERENCES "Game"("id") ON DELETE CASCADE ON UPDATE CASCADE,
  CHECK ("title" != ''),
  CHECK ("content" != '')
);

CREATE INDEX IF NOT EXISTS "idx_games_title" ON "Game"("title");
CREATE INDEX IF NOT EXISTS "idx_games_publisher" ON "Game"("publisher");
CREATE INDEX IF NOT EXISTS "idx_games_last_played_desc" ON "Game"("lastPlayed" DESC);
CREATE INDEX IF NOT EXISTS "idx_games_total_play_time_desc" ON "Game"("totalPlayTime" DESC);

CREATE INDEX IF NOT EXISTS "idx_playsessions_gameid_played_at" ON "PlaySession"("gameId", "playedAt");
CREATE INDEX IF NOT EXISTS "idx_playsessions_played_at_desc" ON "PlaySession"("playedAt" DESC);

CREATE INDEX IF NOT EXISTS "idx_memos_gameid" ON "Memo"("gameId");
CREATE INDEX IF NOT EXISTS "idx_memos_title" ON "Memo"("title");
CREATE INDEX IF NOT EXISTS "idx_memos_created_at" ON "Memo"("createdAt");
CREATE INDEX IF NOT EXISTS "idx_memos_updated_at" ON "Memo"("updatedAt");

CREATE TRIGGER IF NOT EXISTS "trigger_memo_updated_at"
AFTER UPDATE ON "Memo"
FOR EACH ROW
BEGIN
  UPDATE "Memo" SET "updatedAt" = CURRENT_TIMESTAMP WHERE "id" = OLD."id";
END;
