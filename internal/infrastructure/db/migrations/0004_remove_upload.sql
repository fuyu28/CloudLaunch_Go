CREATE TABLE "PlaySession_new" (
  "id" TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
  "gameId" TEXT NOT NULL,
  "playedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "duration" INTEGER NOT NULL,
  "sessionName" TEXT,
  "chapterId" TEXT,
  "updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY ("gameId") REFERENCES "Game"("id") ON DELETE CASCADE ON UPDATE CASCADE,
  FOREIGN KEY ("chapterId") REFERENCES "Chapter"("id") ON DELETE SET NULL ON UPDATE CASCADE,
  CHECK ("duration" >= 0)
);

INSERT INTO "PlaySession_new" (id, gameId, playedAt, duration, sessionName, chapterId, updatedAt)
SELECT id, gameId, playedAt, duration, sessionName, chapterId, updatedAt FROM "PlaySession";

DROP TABLE "PlaySession";
ALTER TABLE "PlaySession_new" RENAME TO "PlaySession";

DROP TABLE IF EXISTS "Upload";

CREATE INDEX IF NOT EXISTS "idx_playsessions_gameid_played_at" ON "PlaySession"("gameId", "playedAt");
CREATE INDEX IF NOT EXISTS "idx_playsessions_gameid_chapterid" ON "PlaySession"("gameId", "chapterId");
CREATE INDEX IF NOT EXISTS "idx_playsessions_played_at_desc" ON "PlaySession"("playedAt" DESC);
