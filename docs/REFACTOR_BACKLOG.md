# Refactor Backlog

`/simplify` と `/code-review` の結果から、**正しさには影響しないが構造的に望ましくない**
項目を集めたバックログ。各項目は独立した PR として段階的に対応する想定。

- 着手判断: 「影響範囲」「価値」「リスク」を見て優先度を決める
- 完了したら本ファイルから削除し、コミットメッセージに該当 ID（A1 等）を引く

---

## A. アーキテクチャ（altitude）

> Clean Architecture の境界違反、責務漏れ、または特殊ケース層が積み上がっている箇所。
> 価値は大きいが波及も広いため、1 項目 = 1 PR を厳守。

### A1. `SessionMutationResult` のサービス→app バブルアップ削除

- **場所**: `internal/services/session_service.go`（Delete/UpdateSessionRoute/UpdateSessionName）
- **問題**: `SessionMutationResult{GameID}` を返すのは app 層の async sync（`syncGameAsync`）用。
  サービス層に app の関心が漏れている。
- **解決方針**: サービスは `error` だけ返し、app 側で事前に
  `repository.GetPlaySessionByID(sessionID)` で gameID を取得する。
- **影響**: 3 メソッド・テスト fake・app の 3 呼び出し。中規模。
- **注意**: delete-before-read の順序が変わるので、テストで先後関係を明示する。

### A2. `MemoCloudService` のサービス→リポジトリ依存への置換

- **場所**: `internal/services/memo_cloud_service.go`
- **問題**: `*GameService` / `*MemoService` を直接受け取っている（サービス間依存）。
  リポジトリ境界を経由していない。
- **解決方針**: `app.go::configureServices` で `GameRepository` / `MemoRepository` を
  `MemoCloudService` に直接注入。`gameService.GetGameByID` 等の呼び出しを
  `repository.GetGameByID` に置換。
- **影響**: コンストラクタ・全呼び出し元・テスト。中〜大規模。

### A3. `storage.BlobKind*` 定数をサービス層から外す

- **場所**: `internal/infrastructure/storage/`（定数）／`internal/services/content_sync_service.go`（消費）
- **問題**: `BlobKindCommit` 等の同期プロトコル概念が `services/` で多用されている。
  ストレージ層は「不透明な (gameID, key)」だけ扱うべき。
- **解決方針**: `PutBlob` / `GetBlob` / `ListBlobHashes` のシグネチャを
  `(gameID, key)` に変える。サービス層が `key = "commits/" + hash` を組み立てる。
- **影響**: シグネチャと全呼び出し元。大規模。
- **代替案**: BlobKind を `domain` へ移すだけで「サービス層が storage に依存」を
  「サービス層が domain に依存」に格上げできる（軽量版）。

### A4. `ApplyPullResult` の同期プロトコルロジックをサービス層へ

- **場所**: `internal/infrastructure/db/repository.go::ApplyPullResult`
- **問題**: 「存在しない Route 参照は NULL に正規化」というビジネスルールが
  リポジトリ層に混入している。
- **解決方針**: サービス層に「sync transaction executor」を作り、リポジトリには
  個別 CRUD（`UpsertGameSync` / `DeletePlaySessionsByGame` / `UpsertPlaySessionSync`）
  だけ残す。トランザクションは executor が orchestrate。
- **影響**: サービス層・トランザクションスコープ・リポジトリインターフェース。大規模。

### A5. `api_cloud.go::buildGameDirectoryNode`（~70行）をサービス層へ

- **場所**: `internal/app/api_cloud.go`
- **問題**: app 層がツリー構築という業務ロジックを持っている。Wails adapter は薄くあるべき。
- **解決方針**: `ContentSyncService.BuildGameDirectoryTree(gameID)` を新設して app から呼ぶ。
- **影響**: 中規模。返り値の domain 型化も併発する可能性あり。
- **関連**: G9 の cloud カード重複統合と同時に検討すると一回で片付く。

### A6. sync coalescer のポリシーをサービス層へ統合

- **場所**: `internal/app/sync_coalescer.go` ＋ `internal/services/content_sync_service.go::gameLocks`
- **問題**: 同一 gameID の直列化ポリシーが app（coalescer）と service（gameLocks）に
  二重で存在している。`syncGameAsync` 経由しか coalesce されず、手動 Pull は
  別経路で gameLocks にだけかかる。
- **解決方針**: サービス層に `Schedule(gameID, op)` のような単一エントリポイントを置き、
  coalescing と locking を一箇所に集約する。
- **影響**: app/services 両方・テスト。中〜大規模。
- **メモ**: code-review Fix #4 で `stop()` を追加したが根本問題は未解決。

### A7. `process_monitor_service.go` の Windows 専用コードを `_windows.go` へ封じる

- **場所**: `internal/services/process_monitor_service.go::getProcessesNative` 周辺
- **問題**: PowerShell / WMIC コマンド文字列が build tag なしのファイルにある。
  CLAUDE.md の「Windows 専用機能は `_windows.go` サフィックスに」に違反。
  macOS/Linux ビルドで永続的に「fallback も失敗」ログが出続ける。
- **解決方針**: `processProvider` ポートを切って `process_provider_windows.go` /
  `process_provider_unsupported.go` に分割。
- **影響**: process_monitor 内で完結。小〜中規模。
- **優先度**: 高（CLAUDE.md 規約違反 + クロスプラットフォームでの誤動作）

### A8. `MaintenanceRuntimeHooks` を単一の RuntimeOrchestrator ポートに

- **場所**: `internal/services/maintenance_service.go::MaintenanceRuntimeHooks`
- **問題**: 5 つの nullable callback が散らばっている。個別に nil チェックがあり、
  app が半配線でも黙って no-op で進む。
- **解決方針**: `RuntimeOrchestrator interface { StopRuntime(); CloseDB(); ReopenDB(); ... }`
  を 1 つ作り、app が実装。
- **影響**: 中規模。MaintenanceService の API は変えなくて済む。

### A10. `frontend/src/state/settings.ts` の localStorage キーレジストリ化

- **場所**: `frontend/src/state/settings.ts`
- **問題**: 13 個の `atomWithStorage` が namespace なしのキー（`'theme'`, `'screenshotHotkey'` 等）で
  バラバラ。`utils/logLevel.ts` だけ `'cloudlaunch_log_level'` プレフィックスありで不統一。
  さらに hotkey 文字列（`'Ctrl+Alt+S'`）は Go 側 `parseHotkeyCombo` と無契約。
- **解決方針**: `frontend/src/state/storageKeys.ts` でキーを集中管理し、
  全 atom がそこから参照。hotkey は zod schema で fe/be 共有形にする。
- **影響**: 中規模。マイグレーション（旧キー → 新キー）を一度書く必要あり。

---

## B. 挙動変更を伴うクリーンアップ（プロダクト判断後に着手）

> 構造的には妥当だが、ユーザー体感やデータフローに影響する可能性がある。
> 「同じ挙動を保つ前提か、改善も含めて良いか」をまず決める。

### B1. `database.ts::updatePlayStatus` の read-update-read 解消

- **場所**: `frontend/src/bridge/database.ts`
- **問題**: 読み出し→変換→更新→再読み出しの 3 往復で playStatus を変える。
  途中で他端末から sync が来ると race。
- **方針候補**:
  - バックエンドに `UpdatePlayStatusOnly(gameID, status)` を追加（アトミック）
  - フロントの read-then-update は維持し、UI 側で再 fetch を諦める

### B2. `GeneralSettings.tsx` ハンドラのフック化

- **場所**: `frontend/src/components/settings/GeneralSettings.tsx`
- **問題**: `handleSyncAllGames` / `handleExportGameData` / `handleCreateBackup` /
  `handleRestoreBackup` 等が 120 行以上の inline 実装。
- **方針候補**:
  - `useBehaviorSettings` / `useSyncAndBackup` 等の hook に分割
  - tab コンポーネント側に handler を移管（props を簡素化）

### B3. `CloudGameImportModal.tsx` の conflict 検出ロジック抽出

- **場所**: `frontend/src/components/cloud/CloudGameImportModal.tsx`
- **問題**: 390 行のロジック内で `findTitleConflicts(cloud, local)` を inline 実装。
- **方針候補**: バックエンド `MaintenanceService` か新規 `CloudImportService` に移管。
  ただし conflict 提示 UI の細かい挙動（編集中の入力との突き合わせ等）が変わりうる。

### B4. `pages/Cloud.tsx` の `path: "*"` センチネル削除モード

- **場所**: `frontend/src/pages/Cloud.tsx`
- **問題**: 削除対象のパスに `"*"` を渡すと「全削除」モード扱い、という暗黙の API。
- **方針候補**: `deleteAll(gameID)` のような明示 API に分解。
  バックエンド API 形状の変更を伴う。

### B5. `dual fetch + cache 重複`（`CloudGameImportModal.tsx`）

- **場所**: 同上
- **問題**: `localGames` を props で受け取りながら内部でも fetch している。
- **方針候補**: props だけに統一するか、内部 fetch だけに統一するか。
  どちらの情報源を「信頼できる側」とするか決める必要がある。

---

## C. 小さく安全なクリーンアップ（次の `/simplify` ラウンドで拾える）

> 1 ファイル以内 / 挙動変更なし / ヘルパー抽出レベル。

### C1. `memo_cloud_service.go` の `Details append` を `recordSyncError` に集約

- 10+ 箇所で `resultData.Details = append(..., fmt.Sprintf(...))`。
- 効果: 行数削減＋意図の明示。リスク: 文字列フォーマットの統一が起きる（実害なし）。

### C2. `memo_cloud_service.go` L111 のキー直構築

- `fmt.Sprintf("games/%s/memo/%s", gameID, fileName)` がここだけ手書き。
- `memo` パッケージに **ファイル名版** の `BuildMemoPath` ヘルパーを追加して統一。

### C3. `wrapServiceError` を `service_error.go` へ移動

- 現状 `memo_cloud_service.go` 内のみ使用だが、サービス共通の関心。
- 場所だけ移して同じ署名で公開。テスト不要レベルの単純 move。

---

## メタ

- 追加した場合は **どこから来た指摘か**（`SIMPLIFY_PASS_NOTES.md` のグループ、
  `/code-review` の Angle）を分かるよう書いておく。
- 解消したらコミットメッセージに ID を引いて、本ファイルから該当項目を削除。
- C は溜まったら一括で `/simplify` に通す（個別 PR を切るほどではない）。
