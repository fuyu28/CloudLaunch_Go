-- name: GetGameByID :one
SELECT id, title, publisher, imagePath, exePath, saveFolderPath, createdAt,
       playStatus, totalPlayTime, lastPlayed, clearedAt, currentChapter
FROM "Game" WHERE id = ?;

-- name: CreateGame :one
INSERT INTO "Game" (title, publisher, imagePath, exePath, saveFolderPath, playStatus, totalPlayTime, lastPlayed, clearedAt, currentChapter)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, title, publisher, imagePath, exePath, saveFolderPath, createdAt,
          playStatus, totalPlayTime, lastPlayed, clearedAt, currentChapter;

-- name: UpdateGame :one
UPDATE "Game"
SET title = ?, publisher = ?, imagePath = ?, exePath = ?, saveFolderPath = ?, playStatus = ?, totalPlayTime = ?, lastPlayed = ?, clearedAt = ?, currentChapter = ?
WHERE id = ?
RETURNING id, title, publisher, imagePath, exePath, saveFolderPath, createdAt,
          playStatus, totalPlayTime, lastPlayed, clearedAt, currentChapter;

-- name: DeleteGame :exec
DELETE FROM "Game" WHERE id = ?;
