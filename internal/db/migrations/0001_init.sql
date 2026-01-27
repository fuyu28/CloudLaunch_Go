PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS "Game" (
  "id" TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
  "title" TEXT NOT NULL,
  "publisher" TEXT NOT NULL,
  "imagePath" TEXT,
  "exePath" TEXT NOT NULL,
  "saveFolderPath" TEXT,
  "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "playStatus" TEXT NOT NULL DEFAULT 'unplayed',
  "totalPlayTime" INTEGER NOT NULL DEFAULT 0,
  "lastPlayed" DATETIME,
  "clearedAt" DATETIME,
  "currentChapter" TEXT,
  CHECK ("title" != ''),
  CHECK ("publisher" != ''),
  CHECK ("exePath" != ''),
  CHECK ("totalPlayTime" >= 0),
  CHECK ("playStatus" IN ('unplayed', 'playing', 'played'))
);

CREATE TABLE IF NOT EXISTS "Chapter" (
  "id" TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
  "name" TEXT NOT NULL,
  "order" INTEGER NOT NULL,
  "gameId" TEXT NOT NULL,
  "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY ("gameId") REFERENCES "Game"("id") ON DELETE CASCADE ON UPDATE CASCADE,
  UNIQUE ("gameId", "order"),
  UNIQUE ("gameId", "name"),
  CHECK ("name" != ''),
  CHECK ("order" >= 0)
);

CREATE TABLE IF NOT EXISTS "Upload" (
  "id" TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
  "clientId" TEXT,
  "comment" TEXT NOT NULL,
  "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "gameId" TEXT NOT NULL,
  FOREIGN KEY ("gameId") REFERENCES "Game"("id") ON DELETE CASCADE ON UPDATE CASCADE,
  CHECK ("comment" != '')
);

CREATE TABLE IF NOT EXISTS "PlaySession" (
  "id" TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
  "gameId" TEXT NOT NULL,
  "playedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "duration" INTEGER NOT NULL,
  "sessionName" TEXT,
  "chapterId" TEXT,
  "uploadId" TEXT,
  FOREIGN KEY ("gameId") REFERENCES "Game"("id") ON DELETE CASCADE ON UPDATE CASCADE,
  FOREIGN KEY ("chapterId") REFERENCES "Chapter"("id") ON DELETE SET NULL ON UPDATE CASCADE,
  FOREIGN KEY ("uploadId") REFERENCES "Upload"("id") ON DELETE SET NULL ON UPDATE CASCADE,
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
CREATE INDEX IF NOT EXISTS "idx_games_play_status" ON "Game"("playStatus");
CREATE INDEX IF NOT EXISTS "idx_games_last_played_desc" ON "Game"("lastPlayed" DESC);
CREATE INDEX IF NOT EXISTS "idx_games_total_play_time_desc" ON "Game"("totalPlayTime" DESC);

CREATE INDEX IF NOT EXISTS "idx_chapters_gameid_order" ON "Chapter"("gameId", "order");
CREATE INDEX IF NOT EXISTS "idx_chapters_name" ON "Chapter"("name");

CREATE INDEX IF NOT EXISTS "idx_uploads_gameid" ON "Upload"("gameId");
CREATE INDEX IF NOT EXISTS "idx_uploads_created_at" ON "Upload"("createdAt");
CREATE INDEX IF NOT EXISTS "idx_uploads_client_id" ON "Upload"("clientId");

CREATE INDEX IF NOT EXISTS "idx_playsessions_gameid_played_at" ON "PlaySession"("gameId", "playedAt");
CREATE INDEX IF NOT EXISTS "idx_playsessions_gameid_chapterid" ON "PlaySession"("gameId", "chapterId");
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
