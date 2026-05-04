-- name: GetGameByID :one
SELECT
  "id",
  "title",
  "publisher",
  "imagePath",
  "exePath",
  "saveFolderPath",
  "createdAt",
  "updatedAt",
  "localSaveHash",
  "localSaveHashUpdatedAt",
  "totalPlayTime",
  "lastPlayed",
  "clearedAt"
FROM "Game"
WHERE "id" = sqlc.arg(game_id);

-- name: GetGameByExePath :one
SELECT
  "id",
  "title",
  "publisher",
  "imagePath",
  "exePath",
  "saveFolderPath",
  "createdAt",
  "updatedAt",
  "localSaveHash",
  "localSaveHashUpdatedAt",
  "totalPlayTime",
  "lastPlayed",
  "clearedAt"
FROM "Game"
WHERE lower("exePath") = lower(sqlc.arg(exe_path));

-- name: CreateGame :one
INSERT INTO "Game" (
  "title",
  "publisher",
  "imagePath",
  "exePath",
  "saveFolderPath",
  "localSaveHash",
  "localSaveHashUpdatedAt",
  "totalPlayTime",
  "lastPlayed",
  "clearedAt"
) VALUES (
  sqlc.arg(title),
  sqlc.arg(publisher),
  sqlc.arg(image_path),
  sqlc.arg(exe_path),
  sqlc.arg(save_folder_path),
  sqlc.arg(local_save_hash),
  sqlc.arg(local_save_hash_updated_at),
  sqlc.arg(total_play_time),
  sqlc.arg(last_played),
  sqlc.arg(cleared_at)
)
RETURNING
  "id",
  "title",
  "publisher",
  "imagePath",
  "exePath",
  "saveFolderPath",
  "createdAt",
  "updatedAt",
  "localSaveHash",
  "localSaveHashUpdatedAt",
  "totalPlayTime",
  "lastPlayed",
  "clearedAt";

-- name: UpdateGame :one
UPDATE "Game"
SET
  "title" = sqlc.arg(title),
  "publisher" = sqlc.arg(publisher),
  "imagePath" = sqlc.arg(image_path),
  "exePath" = sqlc.arg(exe_path),
  "saveFolderPath" = sqlc.arg(save_folder_path),
  "localSaveHash" = sqlc.arg(local_save_hash),
  "localSaveHashUpdatedAt" = sqlc.arg(local_save_hash_updated_at),
  "totalPlayTime" = sqlc.arg(total_play_time),
  "lastPlayed" = sqlc.arg(last_played),
  "clearedAt" = sqlc.arg(cleared_at)
WHERE "id" = sqlc.arg(id)
RETURNING
  "id",
  "title",
  "publisher",
  "imagePath",
  "exePath",
  "saveFolderPath",
  "createdAt",
  "updatedAt",
  "localSaveHash",
  "localSaveHashUpdatedAt",
  "totalPlayTime",
  "lastPlayed",
  "clearedAt";

-- name: DeleteGame :exec
DELETE FROM "Game"
WHERE "id" = sqlc.arg(game_id);

-- name: UpsertGameSync :exec
INSERT INTO "Game" (
  "id",
  "title",
  "publisher",
  "imagePath",
  "exePath",
  "saveFolderPath",
  "createdAt",
  "updatedAt",
  "localSaveHash",
  "localSaveHashUpdatedAt",
  "totalPlayTime",
  "lastPlayed",
  "clearedAt"
) VALUES (
  sqlc.arg(id),
  sqlc.arg(title),
  sqlc.arg(publisher),
  sqlc.arg(image_path),
  sqlc.arg(exe_path),
  sqlc.arg(save_folder_path),
  sqlc.arg(created_at),
  sqlc.arg(updated_at),
  sqlc.arg(local_save_hash),
  sqlc.arg(local_save_hash_updated_at),
  sqlc.arg(total_play_time),
  sqlc.arg(last_played),
  sqlc.arg(cleared_at)
)
ON CONFLICT("id") DO UPDATE SET
  "title" = excluded."title",
  "publisher" = excluded."publisher",
  "imagePath" = excluded."imagePath",
  "exePath" = excluded."exePath",
  "saveFolderPath" = excluded."saveFolderPath",
  "createdAt" = excluded."createdAt",
  "updatedAt" = excluded."updatedAt",
  "localSaveHash" = excluded."localSaveHash",
  "localSaveHashUpdatedAt" = excluded."localSaveHashUpdatedAt",
  "totalPlayTime" = excluded."totalPlayTime",
  "lastPlayed" = excluded."lastPlayed",
  "clearedAt" = excluded."clearedAt";

-- name: TouchGameUpdatedAt :exec
UPDATE "Game"
SET "updatedAt" = CURRENT_TIMESTAMP
WHERE "id" = sqlc.arg(game_id);
