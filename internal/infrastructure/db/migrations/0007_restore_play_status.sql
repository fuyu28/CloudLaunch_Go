ALTER TABLE "Game" ADD COLUMN "playStatus" TEXT NOT NULL DEFAULT 'unplayed';

UPDATE "Game" SET "playStatus" = 'played'  WHERE "clearedAt" IS NOT NULL;
UPDATE "Game" SET "playStatus" = 'playing' WHERE "clearedAt" IS NULL AND "lastPlayed" IS NOT NULL;
