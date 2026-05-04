-- name: CreatePlaySession :one
INSERT INTO "PlaySession" (
  "gameId",
  "playRouteId",
  "playedAt",
  "duration"
) VALUES (
  sqlc.arg(game_id),
  sqlc.arg(play_route_id),
  sqlc.arg(played_at),
  sqlc.arg(duration)
)
RETURNING
  "id",
  "gameId",
  "playRouteId",
  "playedAt",
  "duration",
  "updatedAt";

-- name: GetPlaySessionByID :one
SELECT
  "id",
  "gameId",
  "playRouteId",
  "playedAt",
  "duration",
  "updatedAt"
FROM "PlaySession"
WHERE "id" = sqlc.arg(session_id);

-- name: ListPlaySessionsByGame :many
SELECT
  "id",
  "gameId",
  "playRouteId",
  "playedAt",
  "duration",
  "updatedAt"
FROM "PlaySession"
WHERE "gameId" = sqlc.arg(game_id)
ORDER BY "playedAt" DESC;

-- name: DeletePlaySession :exec
DELETE FROM "PlaySession"
WHERE "id" = sqlc.arg(session_id);

-- name: DeletePlaySessionsByGame :exec
DELETE FROM "PlaySession"
WHERE "gameId" = sqlc.arg(game_id);

-- name: UpsertPlaySessionSync :exec
INSERT INTO "PlaySession" (
  "id",
  "gameId",
  "playRouteId",
  "playedAt",
  "duration",
  "updatedAt"
) VALUES (
  sqlc.arg(id),
  sqlc.arg(game_id),
  sqlc.arg(play_route_id),
  sqlc.arg(played_at),
  sqlc.arg(duration),
  sqlc.arg(updated_at)
)
ON CONFLICT("id") DO UPDATE SET
  "gameId" = excluded."gameId",
  "playRouteId" = excluded."playRouteId",
  "playedAt" = excluded."playedAt",
  "duration" = excluded."duration",
  "updatedAt" = excluded."updatedAt";
