CREATE TABLE "PlaySession_new" (
  "id" TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
  "gameId" TEXT NOT NULL,
  "playedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "duration" INTEGER NOT NULL,
  "routeId" TEXT,
  "updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY ("gameId") REFERENCES "Game"("id") ON DELETE CASCADE ON UPDATE CASCADE,
  FOREIGN KEY ("routeId") REFERENCES "Route"("id") ON DELETE SET NULL ON UPDATE CASCADE,
  CHECK ("duration" >= 0)
);

INSERT INTO "PlaySession_new" (id, gameId, playedAt, duration, routeId, updatedAt)
SELECT id, gameId, playedAt, duration, routeId, updatedAt FROM "PlaySession";

DROP TABLE "PlaySession";
ALTER TABLE "PlaySession_new" RENAME TO "PlaySession";

CREATE INDEX IF NOT EXISTS "idx_playsessions_gameid_played_at" ON "PlaySession"("gameId", "playedAt");
CREATE INDEX IF NOT EXISTS "idx_playsessions_gameid_routeid" ON "PlaySession"("gameId", "routeId");
CREATE INDEX IF NOT EXISTS "idx_playsessions_played_at_desc" ON "PlaySession"("playedAt" DESC);
