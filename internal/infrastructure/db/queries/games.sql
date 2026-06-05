-- name: GetGameByID :one
SELECT id, title, publisher, imagePath, exePath, saveFolderPath, createdAt,
       localSaveHash, localSaveHashUpdatedAt,
       playStatus, totalPlayTime, lastPlayed, clearedAt, currentRouteId
FROM "Game" WHERE id = ?;

-- name: CreateGame :one
INSERT INTO "Game" (title, publisher, imagePath, exePath, saveFolderPath, localSaveHash, localSaveHashUpdatedAt,
                    playStatus, totalPlayTime, lastPlayed, clearedAt, currentRouteId)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, title, publisher, imagePath, exePath, saveFolderPath, createdAt,
          localSaveHash, localSaveHashUpdatedAt,
          playStatus, totalPlayTime, lastPlayed, clearedAt, currentRouteId;

-- name: UpdateGame :one
UPDATE "Game"
SET title = ?, publisher = ?, imagePath = ?, exePath = ?, saveFolderPath = ?, localSaveHash = ?, localSaveHashUpdatedAt = ?,
    playStatus = ?, totalPlayTime = ?, lastPlayed = ?, clearedAt = ?, currentRouteId = ?
WHERE id = ?
RETURNING id, title, publisher, imagePath, exePath, saveFolderPath, createdAt,
          localSaveHash, localSaveHashUpdatedAt,
          playStatus, totalPlayTime, lastPlayed, clearedAt, currentRouteId;

-- name: DeleteGame :exec
DELETE FROM "Game" WHERE id = ?;
