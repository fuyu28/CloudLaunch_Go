# CloudLaunch_Go DB見直しメモ

最終更新: 2026-06-05

## 1. 目的

`tmp/app.db` の実データと現行スキーマ、コード上の参照状況を突き合わせて、将来の DB 見直し候補を整理する。

このメモは、即時の削除作業ではなく、今後の schema 改修の判断材料を残すことを目的とする。

## 2. 調査対象

- 実 DB: `tmp/app.db`
- migration:
  - `internal/db/migrations/0001_init.sql`
  - `internal/db/migrations/0002_add_updated_at.sql`
  - `internal/db/migrations/0003_add_local_save_hash.sql`
  - `internal/db/migrations/0004_remove_upload.sql`
  - `internal/db/migrations/0005_rename_chapter_to_route.sql`
- 実装参照:
  - `internal/db/repository.go`
  - `internal/services/*`
  - `internal/app/*`
  - `frontend/src/*`

## 3. 実データの概要（2026-05-04 時点）

### テーブル件数

- `Game`: 19
- `PlaySession`: 84
- `Route`: 0（旧 `Chapter`、0件のまま再設計）
- `Upload`: 削除済み（migration 0004）
- `Memo`: 0

### `Game` の埋まり方

- `imagePath`: 19 / 19
- `saveFolderPath`: 1 / 19
- `lastPlayed`: 15 / 19
- `clearedAt`: 0 / 19
- `currentRouteId`: 0 / 19（旧 `currentChapter`、migration 0005 で FK 化）
- `updatedAt`: 19 / 19
- `localSaveHash`: 1 / 19
- `localSaveHashUpdatedAt`: 1 / 19

### `PlaySession` の埋まり方

- `sessionName`: 84 / 84
- `routeId`: 0 / 84（旧 `chapterId`、migration 0005 で FK 先変更）
- `updatedAt`: 84 / 84

## 4. 実装と実データを合わせて見た判断

### 4.1 `Upload` テーブル — 完了

**状態: 削除済み（migration 0004）**

対応内容:

- `Upload` テーブル削除
- `PlaySession.uploadId` 削除
- upload 関連 API / service / repository / frontend をすべて除去
- コミット: `chore(db): remove upload table and related code`

### 4.2 `Chapter` / `PlaySession.chapterId` / `Game.currentChapter` — 完了

**状態: Route へ再設計済み（migration 0005）**

対応内容:

- `Chapter` テーブル → `Route` テーブルへリネーム
- `PlaySession.chapterId` → `routeId`（FK 先を `Route` へ）
- `Game.currentChapter` (TEXT) → `currentRouteId` (FK → `Route.id`)
- ゲーム作成時のデフォルトルート名: 「メインルート」
- `ChapterService` → `RouteService`、関連する全レイヤー（models / services / repository / app / frontend）をリネーム
- コミット: `refactor(db): rename Chapter to Route across all layers`

### 4.3 当面維持でよい項目

#### `Game.localSaveHash`
#### `Game.localSaveHashUpdatedAt`

背景:

- 実データ上は 1 件のみ
- 将来的には、クラウド同期時にローカルとクラウドのハッシュが一致した場合、アップロード/ダウンロードを skip したい

現時点の判断:

- いまは `Game` 直持ちのままでよい

理由:

- 現在の前提はほぼ「1ゲーム = 1セーブフォルダ」
- 保存したい値も「そのゲームの最新ローカルハッシュ 1 個」
- 直近の用途は「差分判定して skip する」だけで、別テーブル化の必然性がまだ低い

別テーブル化を検討する条件:

- 1ゲームで複数セーブ領域を扱う
- デバイス別同期状態を持つ
- remote hash も永続化したい
- last sync result など同期状態メタ情報が増える
- 履歴を持ちたくなる

その段階の候補:

- `GameSaveState`
  - `gameId`
  - `localSaveHash`
  - `localSaveHashUpdatedAt`
  - `remoteSaveHash`
  - `remoteSaveHashUpdatedAt`
  - `lastSyncCheckedAt`
  - `lastSyncResult`

結論:

- 今は維持
- skip 判定実装を先に進める
- 同期メタ情報が増えた時点で切り出しを再検討する

### 4.4 意味整理が必要な項目

#### `Game.playStatus`
#### `Game.lastPlayed`
#### `Game.clearedAt`

観測:

- `playStatus = 'unplayed'` が 17 件
- そのうち `lastPlayed` が入っているレコードが 13 件ある
- `clearedAt` は 0 件

示唆:

- これは削除候補というより、列の意味や更新ルールが揺れているサイン

論点:

- `playStatus` を手動設定の状態として持つのか
- `lastPlayed` / `totalPlayTime` から導出するのか
- `clearedAt` を使う UI / 運用が今後本当に必要か

方針:

- DB 削除より前に、状態モデルを整理する

## 5. いつやるべきか

2026-06-05 現在のステータス:

- `Upload` 削除 → **完了**
- `Chapter` → `Route` 再設計 → **完了**
- `services` から `ApiResult` を外す → **完了**
- Use Case 境界の明確化 → **完了**（Phase 3）

次に検討すべき DB 改修:

1. `Game.playStatus` / `lastPlayed` / `clearedAt` の意味整理（状態モデルの定義）
2. `sqlx` 移行（schema 安定後）

## 6. `sqlx` 移行メモ

背景:

- 将来的に DB アクセス層を `sqlx` ベースへ寄せたい意向がある
- 現状は `internal/db/repository.go` に生 SQL と手動 scan が集まっている

期待する効果:

- scan 処理の重複削減
- struct mapping の見通し改善
- query 実装の保守性向上

注意点:

- schema 改修と `sqlx` 移行を同時にやると、変更範囲が広がりすぎる

方針:

- `sqlx` 移行はやりたいこととして明記する
- schema の方向性が固まった後に DB access 実装を `sqlx` へ寄せる

対象候補:

- `internal/db/repository.go`
- `internal/db/queries/*.sql`
- model への scan / mapping ロジック

## 7. 次回の見直し対象

優先度中:

- `Game.playStatus`
- `Game.clearedAt`（`lastPlayed` との意味整合）

当面維持:

- `Game.localSaveHash`
- `Game.localSaveHashUpdatedAt`

将来検討:

- `sqlx` 移行（DB schema 安定後）

## 8. 現時点の結論

- `Upload` は削除済み（migration 0004）
- `Chapter` は `Route` として再設計済み（migration 0005）、`currentRouteId` FK 化も完了
- `localSaveHash` 系は、クラウド同期 skip 判定のために当面 `Game` 直持ちで維持
- `playStatus` / `clearedAt` の意味整合は今後の課題
- `sqlx` 移行はやりたいが、schema 安定後に段階を分けて進める
