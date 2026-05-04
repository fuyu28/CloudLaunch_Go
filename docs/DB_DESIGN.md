# CloudLaunch_Go DB設計メモ

最終更新: 2026-05-04

## 1. 目的

CloudLaunch_Go の DB 改修前に、各テーブル・カラムの意味と更新ルールを固定する。

この文書の目的は、migration の具体実装より前に「何を永続化し、何を導出するか」を明確にすることにある。

## 2. 前提

- 本文書は現行の `internal/db/migrations/*.sql` と `internal/db/repository.go` を踏まえた設計メモである
- 直近の大きな変更対象は以下とする
  - `Upload` 系の削除
  - `Chapter` 系の廃止
  - `PlayRoute` の新設
  - `Game.playStatus` の意味整理
  - `Game.lastPlayed` / `Game.clearedAt` の定義固定
- DB access 層の `sqlc` / `sqlx` 移行は、この文書で定義する schema 方針の後段で扱う

## 3. 設計原則

### 3.1 永続化するもの

- ユーザーが入力した値
- 外部同期や差分判定に必要な値
- 集計コストを下げるために保持する値

### 3.2 導出するもの

- プレイ履歴と入力済み属性から安定して再計算できる値
- UI 表示都合で一時的に使う分類値

### 3.3 今回の方針

- `clearedAt` はユーザー入力なので永続化する
- `lastPlayed` はプレイ履歴から導出可能だが、一覧性と既存実装都合を考慮して当面は保持してよい
- `playStatus` は最終的に導出値として扱う
- 実利用されていない `Upload` / `Chapter` 系は延命せず切り捨てる
- `duration` / `totalPlayTime` は秒単位の整数で統一する
- `clearedAt` は日付のみとして扱う

## 4. テーブル設計

## 4.1 `Game`

ゲームの基本情報と、ゲーム単位で保持したい状態を表す。

### カラム

- `id`
  - ゲーム識別子
- `title`
  - ゲームタイトル
- `publisher`
  - ブランド / メーカー名
- `imagePath`
  - ローカル画像パス
- `exePath`
  - 実行ファイルパス
- `saveFolderPath`
  - セーブフォルダパス
- `createdAt`
  - ゲーム登録日時
- `updatedAt`
  - ゲームレコード更新日時
- `totalPlayTime`
  - 累計プレイ時間
- `lastPlayed`
  - 最後のプレイセッション終了時刻
- `clearedAt`
  - ユーザーが入力したクリア日
- `localSaveHash`
  - ローカルセーブデータのハッシュ
- `localSaveHashUpdatedAt`
  - `localSaveHash` を算出した日時

### 削除予定カラム

- `playStatus`
  - 理由: 履歴と `clearedAt` から導出できるため
- `currentChapter`
  - 理由: `Chapter` 廃止に伴い意味を失うため

### 意味と更新ルール

- `lastPlayed`
  - 「最後のプレイセッション終了時刻」と定義する
  - セッション終了時刻は `playedAt + duration` で求める
  - `PlaySession` の作成・更新・削除後は、対象 `gameId` の `PlaySession` を再集計して整合させる
- `clearedAt`
  - ユーザーが「このゲームをこの日にクリアした」と入力した日付
  - プレイ履歴から自動設定しない
  - 日付のみを保持し、時刻は持たない
- `totalPlayTime`
  - 秒単位の整数で保持する
  - プレイセッションの総和と整合する
- `localSaveHash` / `localSaveHashUpdatedAt`
  - 当面は `Game` 直持ちで維持する

## 4.2 `PlaySession`

1 回のプレイ記録を表す。

### カラム

- `id`
  - セッション識別子
- `gameId`
  - 紐づくゲーム ID
- `playedAt`
  - セッション開始時刻
- `duration`
  - セッション長
- `updatedAt`
  - レコード更新日時

### 新設予定カラム

- `playRouteId`
  - `PlayRoute.id` への nullable 外部キー

### 削除予定カラム

- `sessionName`
  - 理由: セッション単位の命名を維持する運用価値が低いため
- `chapterId`
  - `Chapter` 廃止に伴い削除
- `uploadId`
  - `Upload` 廃止に伴い削除

### 意味と更新ルール

- `duration` は秒単位の整数として保存する
- セッション終了時刻は `playedAt + duration` とみなす
- `PlaySession` の作成・更新・削除後は、対象 `gameId` の `PlaySession` を再集計し、`Game.totalPlayTime` と `Game.lastPlayed` を更新する

## 4.3 `PlayRoute`

ゲームごとの攻略ルート、キャラ別ルート、区分別プレイ時間集計の単位を表す。

`Chapter` の置き換えとして導入する。

### 目的

- 「章」ではなく「攻略ルート / プレイ区分」を明示する
- セッションをルート単位で集計できるようにする

### 想定カラム

- `id`
  - ルート識別子
- `gameId`
  - 紐づくゲーム ID
- `name`
  - ルート名
- `sortOrder`
  - 表示順
- `createdAt`
  - 作成日時

### 方針

- `PlaySession.playRouteId` は nullable とする
- `PlayRoute` 削除時は、紐づく `PlaySession.playRouteId` を `null` に戻す
- ルート管理を使わないゲームも許容する
- 現時点では `Game.currentRouteId` は持たない

### 理由

- 「現在どのルートをプレイ中か」は UI 状態や最新セッションから扱える余地があり、先に永続化すると意味が重くなる
- まずは集計と分類の最小責務に絞る

## 4.4 `Memo`

ゲームに紐づくメモを表す。

現時点では大きな schema 再設計対象ではない。

### カラム

- `id`
- `title`
- `content`
- `gameId`
- `createdAt`
- `updatedAt`

## 4.5 削除対象

## `Upload`

実データ・利用実態の両面から不要と判断する。

### 削除対象

- `Upload` テーブル
- `PlaySession.uploadId`
- upload 関連 API / service / repository / frontend

## `Chapter`

意味が曖昧で実データもないため、いったん廃止する。

### 削除対象

- `Chapter` テーブル
- `PlaySession.chapterId`
- `Game.currentChapter`
- chapter 関連 API / service / repository / frontend

## 5. 導出ルール

## 5.1 `playStatus`

`playStatus` は最終的に DB カラムではなく導出値として扱う。

### 導出定義

- `unplayed`
  - プレイセッション 0 件
- `playing`
  - プレイセッション 1 件以上かつ `clearedAt == null`
- `cleared`
  - `clearedAt != null`

### 備考

- 現行 UI のフィルター都合で一時的に返却 DTO に残すのは許容する
- ただし意味の正本は DB カラムではなく導出ロジックに置く
- 既存互換のため返却値に `played` を残す期間がある場合、`played` は「プレイ経験あり」ではなく「クリア済み」を意味する

## 5.2 `lastPlayed`

### 定義

- 最後のプレイセッション終了時刻

### 算出式

- `playedAt + duration`

### 運用方針

- 当面は `Game` に保持してよい
- ただし意味の正本は `PlaySession` とし、整合性が崩れた場合は再集計で復元できる構造にする
- `PlaySession` が 0 件の場合、`lastPlayed = null` とする

## 5.3 `totalPlayTime`

### 定義

- そのゲームに属する `PlaySession.duration` の合計秒数

### 運用方針

- 一覧表示や集計コストのため保持してよい
- ただし `PlaySession` から再計算可能であることを前提にする
- `PlaySession` が 0 件の場合、`totalPlayTime = 0` とする

## 5.4 集計値の更新方針

`PlaySession` の作成・更新・削除後は、対象 `gameId` に属する `PlaySession` を毎回再集計し、`Game.totalPlayTime` と `Game.lastPlayed` を更新する。

### 再集計ルール

- `totalPlayTime = SUM(duration)`
- `lastPlayed = MAX(playedAt + duration)`
- セッションが 0 件の場合
  - `totalPlayTime = 0`
  - `lastPlayed = null`

差分更新ではなく、まずは毎回再集計を基本方針とする。

## 5.5 時刻・日付の扱い

- `playedAt` / `lastPlayed` / `createdAt` / `updatedAt` は日時として保存する
- `clearedAt` は日付のみとして保存する
- `duration` / `totalPlayTime` は秒単位の整数として保存する

## 6. migration 方針

一度に全部を入れ替えず、変更単位を分ける。

### Phase 1

- `Upload` 関連コードの削除
- `PlaySession.uploadId` と `Upload` テーブルの削除
- `PlaySession.sessionName` の削除
- `Chapter` 関連コードの削除
- `Game.currentChapter` / `PlaySession.chapterId` / `Chapter` テーブルの削除

### Phase 2

- `PlaySession` 作成・更新・削除時の再集計処理を整理する
- `Game.totalPlayTime` / `Game.lastPlayed` の更新責務を統一する
- `playStatus` の返却値を導出ベースに切り替える

### Phase 3

- `PlayRoute` テーブル新設
- `PlaySession.playRouteId` 追加
- 必要最小限の CRUD と集計 UI を追加

### Phase 4

- フィルター・表示の動作確認後に `Game.playStatus` カラムを削除する

## 7. DB access 層の方針

schema の方向性確定後、DB access 層を整理する。

### 推奨

- 主体は `sqlc`
- ただし動的クエリは手書き併用でよい

### 理由

- schema 変更時の型ズレをコンパイル時に検知しやすい
- 現状の `repository.go` の手動 scan 重複を減らしやすい
- 既に `sqlc.yaml` と `internal/db/queries` が存在する

### 注意

- 現在の `internal/db/queries/*.sql` は現行 schema とズレがあるため、そのままでは使わない
- schema 改修と DB access 置き換えは同一コミットにまとめない

## 8. 未確定事項

- `PlayRoute` に説明文や色などの UI 属性を持たせるか
- `lastPlayed` / `totalPlayTime` を将来的に完全導出へ寄せるか
- `playStatus` の返却値を `cleared` に寄せるか、互換期間中は `played` を残すか

## 9. 当面の結論

- `Upload` は削除する
- `Chapter` は一度切り捨て、`PlayRoute` として再設計する
- `PlaySession.sessionName` は削除する
- `lastPlayed` は「最後のセッション終了時刻」と定義する
- `clearedAt` は「ユーザー入力のクリア日」と定義し、日付のみで扱う
- `playStatus` は最終的に導出値へ移行する
