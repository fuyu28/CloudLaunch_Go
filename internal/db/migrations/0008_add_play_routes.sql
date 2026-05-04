DROP TRIGGER IF EXISTS "trigger_play_session_updated_at";
ALTER TABLE "PlaySession" RENAME TO "PlaySession_old";

CREATE TABLE "PlayRoute" (
  "id" TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
  "gameId" TEXT NOT NULL,
  "name" TEXT NOT NULL,
  "sortOrder" INTEGER NOT NULL DEFAULT 0,
  "createdAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY ("gameId") REFERENCES "Game"("id") ON DELETE CASCADE ON UPDATE CASCADE,
  UNIQUE ("gameId", "name"),
  UNIQUE ("gameId", "sortOrder"),
  CHECK ("name" != ''),
  CHECK ("sortOrder" >= 0)
);

CREATE TABLE "PlaySession" (
  "id" TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
  "gameId" TEXT NOT NULL,
  "playRouteId" TEXT,
  "playedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "duration" INTEGER NOT NULL,
  "updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY ("gameId") REFERENCES "Game"("id") ON DELETE CASCADE ON UPDATE CASCADE,
  FOREIGN KEY ("playRouteId") REFERENCES "PlayRoute"("id") ON DELETE SET NULL ON UPDATE CASCADE,
  CHECK ("duration" >= 0)
);

INSERT INTO "PlaySession" (
  id, gameId, playedAt, duration, updatedAt
)
SELECT
  id, gameId, playedAt, duration, updatedAt
FROM "PlaySession_old";

DROP TABLE "PlaySession_old";

CREATE INDEX IF NOT EXISTS "idx_play_routes_gameid_sort_order" ON "PlayRoute"("gameId", "sortOrder");
CREATE INDEX IF NOT EXISTS "idx_play_routes_gameid_name" ON "PlayRoute"("gameId", "name");
CREATE INDEX IF NOT EXISTS "idx_playsessions_gameid_played_at" ON "PlaySession"("gameId", "playedAt");
CREATE INDEX IF NOT EXISTS "idx_playsessions_gameid_play_route_id" ON "PlaySession"("gameId", "playRouteId");
CREATE INDEX IF NOT EXISTS "idx_playsessions_played_at_desc" ON "PlaySession"("playedAt" DESC);

CREATE TRIGGER IF NOT EXISTS "trigger_play_session_updated_at"
AFTER UPDATE ON "PlaySession"
FOR EACH ROW
BEGIN
  UPDATE "PlaySession" SET "updatedAt" = CURRENT_TIMESTAMP WHERE "id" = OLD."id";
END;
