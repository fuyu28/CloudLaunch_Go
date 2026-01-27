-- name: ListChaptersByGame :many
SELECT id, name, "order", gameId, createdAt
FROM "Chapter" WHERE gameId = ? ORDER BY "order" ASC;

-- name: CreateChapter :one
INSERT INTO "Chapter" (name, "order", gameId)
VALUES (?, ?, ?)
RETURNING id, name, "order", gameId, createdAt;

-- name: UpdateChapter :one
UPDATE "Chapter" SET name = ?, "order" = ? WHERE id = ?
RETURNING id, name, "order", gameId, createdAt;

-- name: DeleteChapter :exec
DELETE FROM "Chapter" WHERE id = ?;
