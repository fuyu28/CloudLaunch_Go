ALTER TABLE "PlaySession" RENAME TO "PlaySession_old";

CREATE TABLE "PlaySession" (
  "id" TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
  "gameId" TEXT NOT NULL,
  "playedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "duration" INTEGER NOT NULL,
  "chapterId" TEXT,
  "updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY ("gameId") REFERENCES "Game"("id") ON DELETE CASCADE ON UPDATE CASCADE,
  FOREIGN KEY ("chapterId") REFERENCES "Chapter"("id") ON DELETE SET NULL ON UPDATE CASCADE,
  CHECK ("duration" >= 0)
);

INSERT INTO "PlaySession" (
  id, gameId, playedAt, duration, chapterId, updatedAt
)
SELECT
  id, gameId, playedAt, duration, chapterId, updatedAt
FROM "PlaySession_old";

DROP TABLE "PlaySession_old";

CREATE INDEX IF NOT EXISTS "idx_playsessions_gameid_played_at" ON "PlaySession"("gameId", "playedAt");
CREATE INDEX IF NOT EXISTS "idx_playsessions_gameid_chapterid" ON "PlaySession"("gameId", "chapterId");
CREATE INDEX IF NOT EXISTS "idx_playsessions_played_at_desc" ON "PlaySession"("playedAt" DESC);

DROP TRIGGER IF EXISTS "trigger_play_session_updated_at";

CREATE TRIGGER IF NOT EXISTS "trigger_play_session_updated_at"
AFTER UPDATE ON "PlaySession"
FOR EACH ROW
BEGIN
  UPDATE "PlaySession" SET "updatedAt" = CURRENT_TIMESTAMP WHERE "id" = OLD."id";
END;
