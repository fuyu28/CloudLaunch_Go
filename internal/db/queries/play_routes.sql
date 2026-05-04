-- name: CreatePlayRoute :one
INSERT INTO "PlayRoute" (
  "gameId",
  "name",
  "sortOrder"
) VALUES (
  sqlc.arg(game_id),
  sqlc.arg(name),
  sqlc.arg(sort_order)
)
RETURNING
  "id",
  "gameId",
  "name",
  "sortOrder",
  "createdAt";

-- name: ListPlayRoutesByGame :many
SELECT
  "id",
  "gameId",
  "name",
  "sortOrder",
  "createdAt"
FROM "PlayRoute"
WHERE "gameId" = sqlc.arg(game_id)
ORDER BY "sortOrder" ASC, "createdAt" ASC;

-- name: DeletePlayRoute :exec
DELETE FROM "PlayRoute"
WHERE "id" = sqlc.arg(route_id);

-- name: DeletePlayRoutesByGame :exec
DELETE FROM "PlayRoute"
WHERE "gameId" = sqlc.arg(game_id);

-- name: UpsertPlayRouteSync :exec
INSERT INTO "PlayRoute" (
  "id",
  "gameId",
  "name",
  "sortOrder",
  "createdAt"
) VALUES (
  sqlc.arg(id),
  sqlc.arg(game_id),
  sqlc.arg(name),
  sqlc.arg(sort_order),
  sqlc.arg(created_at)
)
ON CONFLICT("id") DO UPDATE SET
  "gameId" = excluded."gameId",
  "name" = excluded."name",
  "sortOrder" = excluded."sortOrder",
  "createdAt" = excluded."createdAt";
