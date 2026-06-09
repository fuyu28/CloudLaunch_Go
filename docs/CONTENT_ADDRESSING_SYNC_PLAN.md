# コンテンツアドレッシング同期システム 実装計画

ブランチ: `feature/content-addressing-sync`

---

## 概要

既存の `CloudSyncService`（JSON メタデータを S3 に直置き）を廃止し、
Git 風のコンテンツアドレッシングによるセーブデータ同期システムに完全置き換えする。

**対象外:** メモ・スクショ（`MemoCloudService` / `ScreenshotService` は変更しない）

---

## ストレージ設計

### S3 構造

```
games/{gameId}/HEAD                 ← リモートHEAD（最後にpushしたMetaSnapshotのhash）
games/{gameId}/commits/{sha256}     ← MetaSnapshot JSON（git の commit 相当）
games/{gameId}/trees/{sha256}       ← SaveSnapshot JSON（git の tree 相当）
games/{gameId}/meta/{sha256}        ← game.json / sessions.json
games/{gameId}/objects/{sha256}     ← セーブファイル実データ・画像（バイナリ）
screenshots/{gameId}/{filename}     ← スクショ（コンテンツアドレッシング管理外、現行のまま）
```

種別ごとに Content-Type を設定する:
- `commits/` `trees/` `meta/` → `application/json`
- `objects/` → `application/octet-stream`

ゲーム別ディレクトリにより、ゲーム削除時に `games/{gameId}/` を一括削除できる。
スクショは mutable・直接命名のため分離する。

### ローカル状態（SQLite）

| カラム | 型 | 意味 |
|---|---|---|
| `Game.localSyncHead` | TEXT | 最後に sync したときの MetaSnapshot hash（共通祖先） |
| `Settings.device_name` | TEXT | このPCの表示名（`os.Hostname()` で初期化、ユーザー変更可） |

---

## データフォーマット

### MetaSnapshot（コミット相当）

```json
{
  "game.json":     "sha256_of_game_json",
  "sessions.json": "sha256_of_sessions_json",
  "saves":         "sha256_of_saves_snapshot",
  "deviceName":    "MacBook Pro",
  "createdAt":     "2026-06-08T12:00:00Z"
}
```

### game.json

`domain.Game` からマシン固有フィールド（`exePath`・`saveFolderPath`・`localSaveHash` 系）を除いたもの。
`imagePath`（ローカルパス）は `imageHash`（blob hash）に置き換える。
画像が変わると `game.json` の hash が変わり、MetaSnapshot も自動的に更新される。

```json
{
  "id", "title", "publisher",
  "imageHash",
  "playStatus", "totalPlayTime", "lastPlayed", "clearedAt",
  "currentRouteId", "createdAt", "updatedAt"
}
```

### sessions.json

`domain.PlaySession` から `gameId`（パスから自明）を除いたもの。

```json
[{ "id", "playedAt", "duration", "sessionName", "routeId", "updatedAt" }]
```

### SaveSnapshot（ツリー相当）

```json
{
  "slot1.sav": "sha256_of_slot1",
  "slot2.sav": "sha256_of_slot2"
}
```

MetaSnapshot → SaveSnapshot のハッシュ参照（git の commit→tree モデル）。
全オブジェクトは immutable・重複排除。

---

## 同期状態の判定

### 使用する3値

| 値 | 取得元 | 意味 |
|---|---|---|
| `currentLocalHash` | 現在のファイルをハッシュして計算 | 今のローカルの状態 |
| `localSyncHead` | SQLite | 最後に sync したときの状態（共通祖先） |
| `remoteHead` | S3 から取得 | 今のリモートの状態 |

`Status()` は毎回ローカルファイルをハッシュし直す。
アプリを経由せずゲームを直接起動してセーブが変わった場合も正しく検出できる。

### 判定ロジック

**Step 1:** `remoteHead` を取得し、空なら **`NeverSynced`** で確定。
UI では「はじめてアップロード」として扱う。

**Step 2:** `remoteHead` が存在する場合は三値比較。

| currentLocalHash | remoteHead | 状態 |
|---|---|---|
| == localSyncHead == remoteHead | — | `InSync` |
| != localSyncHead, == localSyncHead（remote 側） | ローカルのみ変更 | `PushNeeded` |
| == localSyncHead, != localSyncHead（remote 側） | リモートのみ変更 | `PullNeeded` |
| != localSyncHead, != localSyncHead（remote 側） | 両方が独立して変更 | `Conflict` |

※ `localSyncHead = NULL`（他デバイスが push 済みだがこのデバイスは未 sync）は `PullNeeded` に該当する。

### コンフリクト解決

UI で「ローカルを使う / リモートを使う」を手動選択する。

```
競合が検出されました
  ローカル: MacBook Pro から 2026-06-08 12:00
  リモート: Surface から   2026-06-07 20:30
  [ローカルを使用] [リモートを使用]
```

---

## 実装フェーズ

### Phase 1 — ストレージ基盤

**`internal/infrastructure/storage/blob_store.go`**

```go
// BlobKind はS3上のオブジェクト種別を表す
type BlobKind = string
const (
    BlobKindCommit BlobKind = "commits"
    BlobKindTree   BlobKind = "trees"
    BlobKindMeta   BlobKind = "meta"
    BlobKindObject BlobKind = "objects"
)

PutBlob(ctx, client, bucket, gameId, kind, hash, data) error  // 既存なら skip。kind に応じた Content-Type を設定
GetBlob(ctx, client, bucket, gameId, kind, hash) ([]byte, error)
PutBlobs(ctx, client, bucket, gameId, blobs, concurrency, onProgress) error  // objects/ 固定、ListObjectsV2 で差分のみ並列アップ
DownloadBlobs(ctx, client, bucket, gameId, saveDir, blobs, concurrency, onProgress) error  // objects/ 固定、並列ダウンロード
ListBlobHashes(ctx, client, bucket, gameId) (map[string]struct{}, error)  // objects/ のハッシュ一覧取得
```

S3 キー: `games/{gameId}/{kind}/{hash}`（`blobExists` は内部専用で非公開）

**`internal/infrastructure/storage/head_store.go`**

```
WriteHEAD(ctx, client, bucket, gameId, hash) error
ReadHEAD(ctx, client, bucket, gameId) (string, error)   // 未存在なら "" 返す
```

S3 キー: `games/{gameId}/HEAD`

---

### Phase 2 — ドメイン型

**`internal/domain/sync.go`**

```go
type BlobHash = string

type SaveSnapshot struct {
    Files map[string]BlobHash `json:"files"`
}

type MetaSnapshot struct {
    GameJSON     BlobHash  `json:"game.json"`
    SessionsJSON BlobHash  `json:"sessions.json"`
    Saves        BlobHash  `json:"saves"`
    DeviceName   string    `json:"deviceName"`
    CreatedAt    time.Time `json:"createdAt"`
}

type SyncStatus string

const (
    SyncStatusNeverSynced SyncStatus = "never_synced"
    SyncStatusInSync      SyncStatus = "in_sync"
    SyncStatusPushNeeded  SyncStatus = "push_needed"
    SyncStatusPullNeeded  SyncStatus = "pull_needed"
    SyncStatusConflict    SyncStatus = "conflict"
)

type SyncStatusDetail struct {
    Status     SyncStatus
    LocalMeta  *MetaSnapshot // コンフリクト時の表示用
    RemoteMeta *MetaSnapshot
}
```

---

### Phase 3 — ハッシュ計算・スナップショット構築

**`internal/services/content_hash.go`**

```
hashBytes(data []byte) BlobHash
hashFile(path string) (BlobHash, []byte, error)
buildSaveSnapshot(saveDir string) (SaveSnapshot, map[BlobHash][]byte, error)
buildMetaSnapshot(game, sessions, imageHash, savesHash, deviceName) (MetaSnapshot, []byte, error)
```

`buildSaveSnapshot` のエラー方針:
- `saveFolderPath` が未設定（NULL）→ エラー
- パスは設定されているがディスク上にフォルダが存在しない → エラー

---

### Phase 4 — 同期サービス

**`internal/services/content_sync_service.go`**（既存 `CloudSyncService` を置き換え）

S3 操作を `contentBlobStore` インターフェースで抽象化し、テストで `fakeBlobStore`（インメモリ）に差し替えられる設計。

```go
// contentBlobStore はテスト差し替え用インターフェース
type contentBlobStore interface {
    readHEAD(ctx, gameID) (string, error)
    writeHEAD(ctx, gameID, hash) error
    getBlob(ctx, gameID, kind, hash) ([]byte, error)
    putBlob(ctx, gameID, kind, hash string, data) error
    putBlobs(ctx, gameID, blobs, concurrency, onProgress) error
    downloadBlobs(ctx, gameID, saveDir, blobs, concurrency, onProgress) error
    deleteByPrefix(ctx, prefix) error
}

// ContentSyncService は newBlobStore フィールドにクロージャを持ち、
// テストでは fakeBlobStore を返すクロージャを注入する
type ContentSyncService struct {
    newBlobStore func(ctx) (contentBlobStore, error)
    ...
}

// ProgressFunc はセーブファイルの転送進捗を報告するコールバック。
// current: 完了件数, total: 総件数（セーブファイルのみカウント）
type ProgressFunc func(current, total int)

func (s *ContentSyncService) Status(ctx, gameId) (domain.SyncStatusDetail, error)
func (s *ContentSyncService) Push(ctx, gameId string, onProgress ProgressFunc) error
func (s *ContentSyncService) Pull(ctx, gameId string, onProgress ProgressFunc) error
func (s *ContentSyncService) ResolveConflict(ctx, gameId string, useLocal bool) error
func (s *ContentSyncService) DeleteFromCloud(ctx, gameId) error
```

テストは `internal/services/content_sync_service_test.go` に 16 本（Push 4 / Pull 4 / Status 5 / ResolveConflict 1 / DeleteFromCloud 1）。

**Status の流れ**
1. `remoteHead` を S3 から取得 → 空なら `NeverSynced` を返す
2. ローカルファイルをハッシュして `currentLocalHash` を算出
3. SQLite から `localSyncHead` を取得
4. 三値比較で状態を決定

**Push の流れ**
1. ゲーム情報・セッション・セーブフォルダをハッシュ化
2. 新規セーブファイル数を集計して `onProgress(0, total)` を呼ぶ
3. セーブファイルを1件ずつアップロードし `onProgress(current, total)` を呼ぶ
4. game.json / sessions.json / SaveSnapshot / MetaSnapshot をアップロード
5. `games/{gameId}/HEAD` を新 hash で更新
6. SQLite の `localSyncHead` を更新

**Pull の流れ**
1. remote HEAD を取得
2. MetaSnapshot を取得・デコード
3. SaveSnapshot を取得し必要なセーブファイル数を集計して `onProgress(0, total)` を呼ぶ
4. セーブファイルを1件ずつダウンロードし `onProgress(current, total)` を呼ぶ
5. ローカルのゲーム情報・セッション・セーブフォルダを上書き
6. SQLite の `localSyncHead` を更新

**DeleteFromCloud の流れ**
1. S3 の `games/{gameId}/` を一括削除

---

### Phase 5 — DB マイグレーション

**`internal/infrastructure/db/migrations/0007_content_addressing_sync.sql`**

```sql
ALTER TABLE "Game" ADD COLUMN "localSyncHead" TEXT;

CREATE TABLE IF NOT EXISTS "Settings" (
  "key"   TEXT NOT NULL PRIMARY KEY,
  "value" TEXT NOT NULL
);
```

デバイス名は初回起動時に `os.Hostname()` を取得して `INSERT OR IGNORE` する。

---

### Phase 6 — App 層 / フロントエンド

**`internal/app/api_sync.go`**（書き換え）

```
SyncStatus(gameId string) ApiResult[SyncStatusDetail]
PushSync(gameId string) ApiResult[any]
PullSync(gameId string) ApiResult[any]
ResolveConflict(gameId string, useLocal bool) ApiResult[any]
DeleteGameFromCloud(gameId string) ApiResult[any]
```

`PushSync` / `PullSync` は `ProgressFunc` のコールバック内で `runtime.EventsEmit` を呼び、
進捗をフロントエンドにストリームする。

```go
// App 層でのコールバック組み立てイメージ
onProgress := func(current, total int) {
    runtime.EventsEmit(ctx, "sync:progress", map[string]any{
        "operation": "push", // or "pull"
        "current":   current,
        "total":     total,
    })
}
app.ContentSyncService.Push(ctx, gameId, onProgress)
```

**`frontend/src/wailsBridge.ts`**
- 上記 5 メソッドを `window.api` に追加
- `window.api.onSyncProgress(callback)` — `"sync:progress"` イベントをサブスクライブ

**イベントペイロード**
```ts
type SyncProgressEvent = {
  operation: "push" | "pull"
  current: number   // 完了したセーブファイル数
  total: number     // 転送対象セーブファイルの総数
}
```

**コンフリクト解決 UI**

`frontend/src/components/SyncConflictModal.tsx` を実装。
- `GameDetail` ページが `handleSyncGame` でステータスを確認し、`conflict` 時にモーダルを開く
- ローカル・リモートのデバイス名と `createdAt` を並べて表示
- 「ローカルを使う」→ `window.api.cloudSync.resolveConflict(gameId, true)`
- 「クラウドを使う」→ `window.api.cloudSync.resolveConflict(gameId, false)`
- `CloudDataCard` に「同期確認」ボタン（`onSync` prop）を追加し、`GameDetail` から制御

**ゲーム削除 UI**
- **表示のみ削除** — ローカル DB からのみ削除。S3 は残す。
- **クラウドからも削除** — ローカル DB 削除 + `DeleteGameFromCloud` を呼ぶ。

---

## 実装順序

```
Phase 1（ストレージ基盤）
  └─▶ Phase 2（ドメイン型）
        └─▶ Phase 3（ハッシュ計算）
              └─▶ Phase 5（DB マイグレーション）
                    └─▶ Phase 4（同期サービス）
                          └─▶ Phase 6（App / Frontend）
```

Phase 1〜3 は外部依存なしでテストが書きやすい。Phase 4 が最もボリュームが大きい。

---

## 移行方針

- 既存の `CloudSyncService` が使っていた S3 上のデータは無視する（移行ロジックなし）
- アプリ未配布のため許容
