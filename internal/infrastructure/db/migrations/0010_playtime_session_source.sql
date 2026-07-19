-- H4: Game.totalPlayTime / lastPlayed を PlaySession 集計の派生キャッシュへ揃える。
-- 既存で totalPlayTime > SUM(duration) の差分は識別可能な調整セッションとして保持する。
-- id は gameId 由来で決定的にし、再実行しても二重挿入しない（ON CONFLICT DO NOTHING）。

INSERT INTO "PlaySession" (id, gameId, playedAt, duration, sessionName, routeId)
SELECT
  'h4adj-' || g.id,
  g.id,
  COALESCE(g.lastPlayed, g.createdAt, CURRENT_TIMESTAMP),
  g.totalPlayTime - COALESCE((
    SELECT SUM(ps.duration) FROM "PlaySession" ps WHERE ps.gameId = g.id
  ), 0),
  'プレイ時間の移行調整',
  NULL
FROM "Game" g
WHERE g.totalPlayTime > COALESCE((
  SELECT SUM(ps.duration) FROM "PlaySession" ps WHERE ps.gameId = g.id
), 0)
ON CONFLICT(id) DO NOTHING;

-- セッション合計が正本。差分が負だったゲームも含め派生キャッシュを再計算する。
UPDATE "Game"
SET
  totalPlayTime = (
    SELECT COALESCE(SUM(duration), 0) FROM "PlaySession" WHERE gameId = "Game".id
  ),
  lastPlayed = (
    SELECT MAX(playedAt) FROM "PlaySession" WHERE gameId = "Game".id
  );
