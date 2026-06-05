CREATE TABLE "Route" (
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

INSERT INTO "Route" (id, name, "order", gameId, createdAt)
SELECT id, name, "order", gameId, createdAt FROM "Chapter";

CREATE TABLE "PlaySession_new" (
  "id" TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
  "gameId" TEXT NOT NULL,
  "playedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "duration" INTEGER NOT NULL,
  "sessionName" TEXT,
  "routeId" TEXT,
  "updatedAt" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY ("gameId") REFERENCES "Game"("id") ON DELETE CASCADE ON UPDATE CASCADE,
  FOREIGN KEY ("routeId") REFERENCES "Route"("id") ON DELETE SET NULL ON UPDATE CASCADE,
  CHECK ("duration" >= 0)
);

INSERT INTO "PlaySession_new" (id, gameId, playedAt, duration, sessionName, routeId, updatedAt)
SELECT id, gameId, playedAt, duration, sessionName, chapterId, updatedAt FROM "PlaySession";

DROP TABLE "PlaySession";

ALTER TABLE "PlaySession_new" RENAME TO "PlaySession";

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
  "playStatus" TEXT NOT NULL DEFAULT 'unplayed',
  "totalPlayTime" INTEGER NOT NULL DEFAULT 0,
  "lastPlayed" DATETIME,
  "clearedAt" DATETIME,
  "currentRouteId" TEXT,
  FOREIGN KEY ("currentRouteId") REFERENCES "Route"("id") ON DELETE SET NULL ON UPDATE CASCADE,
  CHECK ("title" != ''),
  CHECK ("publisher" != ''),
  CHECK ("exePath" != ''),
  CHECK ("totalPlayTime" >= 0),
  CHECK ("playStatus" IN ('unplayed', 'playing', 'played'))
);

INSERT INTO "Game_new" (id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
  localSaveHash, localSaveHashUpdatedAt, playStatus, totalPlayTime, lastPlayed, clearedAt, currentRouteId)
SELECT id, title, publisher, imagePath, exePath, saveFolderPath, createdAt, updatedAt,
  localSaveHash, localSaveHashUpdatedAt, playStatus, totalPlayTime, lastPlayed, clearedAt, NULL
FROM "Game";

DROP TABLE "Game";

ALTER TABLE "Game_new" RENAME TO "Game";

DROP TABLE "Chapter";

CREATE INDEX IF NOT EXISTS "idx_games_title" ON "Game"("title");

CREATE INDEX IF NOT EXISTS "idx_games_publisher" ON "Game"("publisher");

CREATE INDEX IF NOT EXISTS "idx_games_play_status" ON "Game"("playStatus");

CREATE INDEX IF NOT EXISTS "idx_games_last_played_desc" ON "Game"("lastPlayed" DESC);

CREATE INDEX IF NOT EXISTS "idx_games_total_play_time_desc" ON "Game"("totalPlayTime" DESC);

CREATE INDEX IF NOT EXISTS "idx_routes_gameid_order" ON "Route"("gameId", "order");

CREATE INDEX IF NOT EXISTS "idx_routes_name" ON "Route"("name");

CREATE INDEX IF NOT EXISTS "idx_playsessions_gameid_played_at" ON "PlaySession"("gameId", "playedAt");

CREATE INDEX IF NOT EXISTS "idx_playsessions_gameid_routeid" ON "PlaySession"("gameId", "routeId");

CREATE INDEX IF NOT EXISTS "idx_playsessions_played_at_desc" ON "PlaySession"("playedAt" DESC);
