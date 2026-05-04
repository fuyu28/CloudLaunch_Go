# CloudLaunch_Go DB見直しメモ

最終更新: 2026-05-04

## 1. 目的

`tmp/app.db` の実データと現行スキーマ、コード上の参照状況を突き合わせて、将来の DB 見直し候補を整理する。

このメモは、即時の削除作業ではなく、今後の schema 改修の判断材料を残すことを目的とする。

## 2. 調査対象

- 実 DB: `tmp/app.db`
- migration:
  - `internal/db/migrations/0001_init.sql`
  - `internal/db/migrations/0002_add_updated_at.sql`
  - `internal/db/migrations/0003_add_local_save_hash.sql`
- 実装参照:
  - `internal/db/repository.go`
  - `internal/services/*`
  - `internal/app/*`
  - `frontend/src/*`

## 3. 実データの概要

### テーブル件数

- `Game`: 19
- `PlaySession`: 84
- `Chapter`: 0
- `Upload`: 0
- `Memo`: 0

### `Game` の埋まり方

- `imagePath`: 19 / 19
- `saveFolderPath`: 1 / 19
- `lastPlayed`: 15 / 19
- `clearedAt`: 0 / 19
- `currentChapter`: 0 / 19
- `updatedAt`: 19 / 19
- `localSaveHash`: 1 / 19
- `localSaveHashUpdatedAt`: 1 / 19

### `PlaySession` の埋まり方

- `sessionName`: 84 / 84
- `chapterId`: 0 / 84
- `uploadId`: 0 / 84
- `updatedAt`: 84 / 84

### `Upload` の埋まり方

- `Upload` 自体が 0 件

## 4. 実装と実データを合わせて見た判断

### 4.1 即削除候補

#### `Upload` テーブル

理由:

- 実データが 0 件
- `PlaySession.uploadId` も 0 件
- 機能として不要であることを確認済み

関連削除対象:

- `Upload` テーブル
- `PlaySession.uploadId`
- `Upload.clientId`
- `Upload.comment`
- upload 関連 API / service / repository / frontend

方針:

- 次回の DB 見直しで削除対象として扱う
- 削除前にコード参照を先に落とす

### 4.2 再設計候補

#### `Chapter` / `PlaySession.chapterId` / `Game.currentChapter`

背景:

- 当初は「どのキャラのルートにどのぐらい時間がかかったか」を集計するために導入した
- ただし実データ上は `Chapter` 0 件、`PlaySession.chapterId` 0 件、`Game.currentChapter` 0 件

現状の問題:

- `Chapter` という名前が、「章」なのか「ルート」なのか「進行区分」なのか曖昧
- `Game.currentChapter` は文字列で保持されており、外部キー整合性がない
- 実際に欲しいのは「章管理」より「ルート別プレイ時間集計」に近い

現時点の判断:

- 今の `Chapter` 設計は延命より再設計が望ましい
- 残す場合でも、`Chapter` をそのまま使うより「ルート/区分」概念に置き換える方が自然

有力案:

- `GameRoute` 的なテーブルへ再定義する
- `PlaySession.routeId` nullable
- 必要なら `Game.currentRouteId` を持つ

保留事項:

- 本当に「現在の進行位置」を永続化したいか
- 集計だけで足りるなら、現在値を DB に持たず UI 状態や最新セッションから導出する案もありうる

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

現時点では、DB の即時大改修より先に以下を優先する。

1. `services` から `ApiResult` を外す
2. use case 境界をもう少し明確にする
3. その後に DB 見直しへ入る
4. `sqlx` 移行は schema 見直しの後、または並行でも変更単位を分けて行う

理由:

- いま DB 構造とコード責務を同時に大きく動かすと、回帰原因の切り分けが難しくなる
- 特にカラム削除は戻しにくいため、Use Case 境界が安定してからの方が安全

実務上の推奨順:

1. 先に棚卸しを完了する
2. Upload 廃止のコード削除を進める
3. Chapter を Route 系へ再設計するか判断する
4. migration で schema を整理する
5. その後に `internal/db/repository.go` 周辺の実装を `sqlx` ベースへ段階的に移す

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
- 不具合が出た時に、schema 側と access layer 側のどちらが原因か切り分けづらくなる

方針:

- `sqlx` 移行はやりたいこととして明記する
- ただし、`Upload` 削除や `Chapter` 再設計のような schema 改修とは別の変更単位で進める
- 先に schema の方向性を固め、その後に DB access 実装を `sqlx` へ寄せる

推奨タイミング:

1. Use Case 境界の整理
2. 不要カラム / 不要テーブルの削除
3. `Chapter` 周辺の再設計判断
4. その後に `sqlx` 移行

対象候補:

- `internal/db/repository.go`
- `internal/db/queries/*.sql`
- model への scan / mapping ロジック
## 7. 次回の見直し対象

優先度高:

- `Upload`
- `PlaySession.uploadId`
- `Chapter`
- `PlaySession.chapterId`
- `Game.currentChapter`

優先度中:

- `Game.playStatus`
- `Game.clearedAt`

当面維持:

- `Game.localSaveHash`
- `Game.localSaveHashUpdatedAt`

## 8. 現時点の結論

- `Upload` は完全に不要なので、次回の schema 改修で削除対象
- `Chapter` は今のまま維持するより、ルート集計を主目的に再設計した方がよい
- `localSaveHash` 系は、クラウド同期 skip 判定のために当面 `Game` 直持ちで維持してよい
- DB の物理削除や分割は、Use Case 境界の整理がもう一段進んだ後に行う
- `sqlx` 移行はやりたいが、schema 改修と同時には行わず、段階を分けて進める
