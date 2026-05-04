-- name: GetGameByID :one
SELECT id, title, publisher, imagePath, exePath, saveFolderPath, createdAt,
       localSaveHash, localSaveHashUpdatedAt,
       playStatus, totalPlayTime, lastPlayed, clearedAt
FROM "Game" WHERE id = ?;

-- name: CreateGame :one
INSERT INTO "Game" (title, publisher, imagePath, exePath, saveFolderPath, localSaveHash, localSaveHashUpdatedAt,
                    playStatus, totalPlayTime, lastPlayed, clearedAt)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, title, publisher, imagePath, exePath, saveFolderPath, createdAt,
          localSaveHash, localSaveHashUpdatedAt,
          playStatus, totalPlayTime, lastPlayed, clearedAt;

-- name: UpdateGame :one
UPDATE "Game"
SET title = ?, publisher = ?, imagePath = ?, exePath = ?, saveFolderPath = ?, localSaveHash = ?, localSaveHashUpdatedAt = ?,
    playStatus = ?, totalPlayTime = ?, lastPlayed = ?, clearedAt = ?
WHERE id = ?
RETURNING id, title, publisher, imagePath, exePath, saveFolderPath, createdAt,
          localSaveHash, localSaveHashUpdatedAt,
          playStatus, totalPlayTime, lastPlayed, clearedAt;

-- name: DeleteGame :exec
DELETE FROM "Game" WHERE id = ?;
