# Code Review Findings

日付: 2026-07-11（第2パス追記同日）  
作業ブランチ: `fix/code-review-p0`（`fix/frontend-bugs` から分岐）  
関連: `docs/REFACTOR_BACKLOG.md` / `docs/PERF_BACKLOG.md`

正しさ・データ整合・実ユーザー影響に直結する問題の追跡用。  
構造改善のみの項目は REFACTOR / PERF バックログへ。

## ブランチ方針

- **`fix/frontend-bugs` を先に main へマージする**（フロント correctness のまとまった PR）
- 本ドキュメントと P0/P1 修正は **`fix/code-review-p0`** で進める（backend sync 含むため混ぜない）
- `fix/code-review-p0` の PR は frontend-bugs マージ後に rebase する想定

## 優先度

| ラベル | 意味 |
|--------|------|
| **P0** | データ破損・同期破綻・復旧困難 |
| **P1** | 特定環境で確実に壊れる / 境界の穴 |
| **P2** | 整合性・UX・保守性 |
| **P3** | 低頻度・影響小 |

## ステータス凡例

`todo` / `doing` / `done` / `deferred`

---

## P0 / P1

| ID | 優先 | 状態 | 要約 |
|----|------|------|------|
| H1 | P1 | done | `services.resolveS3Config` が `ForcePathStyle` を落とす |
| H2 | P1 | done | `UpdateUploadConcurrency` が ContentSyncService に届かない |
| H3 | P0 | deferred | Pull がディスク先行 → DB 失敗で乖離（要ステージング設計） |
| H4 | P1 | deferred | プレイ時間 `+=` と SUM の二系統・非原子 |
| H5 | P1 | deferred | Home/GameDetail 起動前同期の二重実装（H11 後に抽出） |
| H6 | P1 | done | `openExternalUrl` 化済み（`fix/frontend-bugs`） |
| H7 | P0 | done | メモ同期がクラウド memo ID を捨てて再採番 |
| H8 | P0 | deferred | Route 未同期 → Pull で FK NULL 化（要プロトコル拡張） |
| H9 | P1 | done | 復元後 hotkey 失敗で AppData ロールバック |
| H10 | P1 | done | 起動時 `autoTracking` / concurrency 未同期 |
| H11 | P0 | done | 起動前確認が `conflict` を pull 扱い（ローカル上書き） |

## P2（抜粋）

| ID | 状態 | 要約 |
|----|------|------|
| M1 | todo | DeleteGame がメモファイルを残す |
| M2 | todo | CreateGame の CreateRoute 失敗無視 |
| M3 | todo | Status が lockGame 外 |
| M4 | todo | ErogameScape ホスト未検証 |
| M5 | todo | OpenFolder が explorer.exe 固定 |
| M11 | done | CreatePlaySession が誤った行を返す |
| M12 | todo | Push HEAD 後の local baseline 非原子 |
| M14 | done | DownloadMemoFromCloud キー未サニタイズ |
| M18 | done | 設定 atom を backend 成功前に更新 |
| M19 | done | download→launch が失敗でも起動 |
| M20 | done | 全ゲーム同期が conflict 無視 |

詳細な再現手順・修正方針は会話ログおよび下記「実装メモ」を参照。

## 実装メモ（本ブランチで着手中）

### H1
`internal/services/cloud_common.go::resolveS3Config` に `ForcePathStyle: base.S3ForcePathStyle` を追加。

### H2
`ContentSyncService.SetUploadConcurrency` を追加し、`App.UpdateUploadConcurrency` から呼ぶ。

### H7
`CreateMemo` が任意 ID を INSERT できるようにし、`syncCloudToLocal` で `cloudMemo.MemoID` を渡す。

### H9
`resumeRuntimeServicesAfterRestore` の hotkey 失敗は Warn のみ（restoreErr にしない）。

### H10
`MainLayout` 起動時に `updateAutoTracking` / `updateUploadConcurrency` を同期。

### H11
`pull_needed` のみダウンロード確認。`conflict` は `SyncConflictModal`。

### H3 / H8
影響大のため本 PR では着手せず、別コミット／ADR 後に実施。
