# コンテンツアドレッシング同期システム 実装計画

ブランチ: `feature/content-addressing-sync`（`refactor/architecture` から分岐）

## 概要

既存の `CloudSyncService`（JSON メタデータを S3 に直置き）を廃止し、
Git 風のコンテンツアドレッシングによるセーブデータ同期システムに完全置き換えする。

メモ・スクショは対象外（現状の直接 S3 管理を維持）。

---

## S3 構造

```
objects/{sha256}              ← 全ブロブ（JSON・画像・セーブファイル）
games/{gameId}/HEAD           ← リモートHEAD（最後にpushしたスナップショットhash）
```

## ローカル状態

```
SQLite: Game.localSyncHead TEXT  ← ローカルHEAD（最後にsync済みのスナップショットhash）
SQLite: Settings.device_id       ← このPCの識別子（初回起動時にUUID自動生成）
```

---

## ドメイン型

### MetaSnapshot（コミット相当）

```json
{
  "game.json":    "sha256_of_game_json",
  "sessions.json": "sha256_of_sessions_json",
  "image.jpg":    "sha256_of_image",
  "saves":        "sha256_of_saves_snapshot",
  "deviceId":     "550e8400-e29b-41d4-a716-446655440000",
  "createdAt":    "2026-06-08T12:00:00Z"
}
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

| localSyncHead | remote HEAD | 状態 |
|---|---|---|
| 同じ | 同じ | `InSync` |
| ローカルが新しい | 古い | `PushNeeded` |
| ローカルが古い | 新しい | `PullNeeded` |
| 両方が diverge | 別々 | `Conflict` |

コンフリクト時は手動解決：UI で「ローカルを使う / リモートを使う」を選択。
表示例:
```
競合が検出されました
  ローカル: MacBook Pro から 2026-06-08 12:00
  リモート: Surface から   2026-06-07 20:30
  [ローカルを使用] [リモートを使用]
```

---

## 実装フェーズ

### Phase 1 — ストレージ基盤（infrastructure 層）

**`internal/infrastructure/storage/blob_store.go`**
- `PutBlob(ctx, client, bucket, hash, data []byte) error` — 既存なら skip
- `GetBlob(ctx, client, bucket, hash) ([]byte, error)`
- `BlobExists(ctx, client, bucket, hash) (bool, error)`

**`internal/infrastructure/storage/head_store.go`**
- `WriteHEAD(ctx, client, bucket, gameId, hash) error`
- `ReadHEAD(ctx, client, bucket, gameId) (string, error)` — 未存在なら `""` 返す

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
    ImageBlob    BlobHash  `json:"image,omitempty"`
    Saves        BlobHash  `json:"saves"`
    DeviceID     string    `json:"deviceId"`
    CreatedAt    time.Time `json:"createdAt"`
}

type SyncStatus string

const (
    SyncStatusInSync      SyncStatus = "in_sync"
    SyncStatusPushNeeded  SyncStatus = "push_needed"
    SyncStatusPullNeeded  SyncStatus = "pull_needed"
    SyncStatusConflict    SyncStatus = "conflict"
)

type SyncStatusDetail struct {
    Status      SyncStatus
    LocalMeta   *MetaSnapshot // コンフリクト時の表示用
    RemoteMeta  *MetaSnapshot
}
```

### Phase 3 — ハッシュ計算・スナップショット構築

**`internal/services/content_hash.go`**
- `hashBytes(data []byte) BlobHash` — SHA-256
- `hashFile(path string) (BlobHash, []byte, error)`
- `buildSaveSnapshot(saveDir string) (SaveSnapshot, map[BlobHash][]byte, error)` — フォルダ走査
- `buildMetaSnapshot(game, sessions, imageHash, savesHash, deviceId) (MetaSnapshot, []byte, error)`

### Phase 4 — 新同期サービス

**`internal/services/content_sync_service.go`**（既存 `CloudSyncService` を置き換え）

```go
type ContentSyncService struct { ... }

func (s *ContentSyncService) Status(ctx, gameId) (domain.SyncStatusDetail, error)
func (s *ContentSyncService) Push(ctx, gameId) error
func (s *ContentSyncService) Pull(ctx, gameId) error
func (s *ContentSyncService) ResolveConflict(ctx, gameId string, useLocal bool) error
```

Push の流れ:
1. ゲーム情報・セッション・セーブフォルダをハッシュ化
2. 変更があったオブジェクトのみ S3 にアップロード（BlobExists でスキップ）
3. MetaSnapshot を生成してオブジェクトとして保存
4. `games/{gameId}/HEAD` を新 hash で更新
5. SQLite の `localSyncHead` を更新

Pull の流れ:
1. remote HEAD を取得
2. MetaSnapshot を取得・デコード
3. 差分オブジェクトをダウンロード
4. ローカルのゲーム情報・セッション・セーブフォルダを上書き
5. `localSyncHead` を更新

### Phase 5 — DB マイグレーション

**`internal/infrastructure/db/migrations/0007_content_addressing_sync.sql`**

```sql
-- localSyncHead: 最後にsync済みのMetaSnapshot hash
ALTER TABLE "Game" ADD COLUMN "localSyncHead" TEXT;

-- デバイスID管理テーブル
CREATE TABLE IF NOT EXISTS "Settings" (
  "key"   TEXT NOT NULL PRIMARY KEY,
  "value" TEXT NOT NULL
);
```

デバイスID は Go 側で初回起動時に UUID を生成して `INSERT OR IGNORE` する。

### Phase 6 — App 層 / フロントエンド

**`internal/app/api_sync.go`**（書き換え）
- `SyncStatus(gameId string) ApiResult[SyncStatusDetail]`
- `PushSync(gameId string) ApiResult[any]`
- `PullSync(gameId string) ApiResult[any]`
- `ResolveConflict(gameId string, useLocal bool) ApiResult[any]`

**`frontend/src/wailsBridge.ts`**
- 上記 4 メソッドを `window.api` に追加

---

## 実装順序

```
Phase 1（ストレージ基盤）
  ↓
Phase 2（ドメイン型）
  ↓
Phase 3（ハッシュ計算）
  ↓
Phase 5（DB マイグレーション）
  ↓
Phase 4（同期サービス）
  ↓
Phase 6（App / Frontend）
```

Phase 1〜3 は外部依存なしでテストが書きやすい。
Phase 4 が最もボリュームが大きい。

---

## 旧データ・旧フォーマットの扱い

既存の `CloudSyncService` が使っていた S3 上のデータは無視する。
移行ロジックは実装しない（アプリ未配布のため許容）。

## メモ・スクショの扱い

コンテンツアドレッシング管理の対象外。
`MemoCloudService` / `ScreenshotService` は変更しない。
