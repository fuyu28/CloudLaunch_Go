# CloudLaunch_Go Clean Architecture 進捗状況

最終更新: 2026-06-06（Phase 5 完了・残課題の記述を実装に合わせて更新）

## 概要

Clean Architecture への移行は、`internal/app` の薄型化、`internal/services` の repository 境界切り出し、t_wada TDD に倣った回帰テスト追加を中心に進行している。

現時点では、教科書どおりの全面再配置はまだ行っていないが、依存方向の整理とテスト容易性の改善はかなり前進している。今後のリファクタリングは、既存実装で成立している振る舞いを先にテストで固定し、その安全網の上で小さく進める。

## 現在地

- `internal/app` から `db.Repository` 直接依存を解消し、`App.Database` フィールドを削除した
- 主要な service は具象の `*db.Repository` ではなく interface に依存する形へ移行済み
- 移行済み service には fake repository ベースの単体テストを追加済み
- `CloudService` / `MemoCloudService` / `CredentialService` も fake port を差し替えて単体テストできる状態になった
- `CloudSyncService` は repository 境界化と純粋ロジックの分割を進めている段階
- `MemoService`, `SessionService`, `GameService`, `ProcessMonitorService` には、リファクタリング前の回帰安全網として副作用・集計・状態遷移テストを追加済み
- `api_memo_cloud.go` と `api_maintenance.go` は service 呼び出し中心の薄い adapter へ整理した
- `services` が `result.ApiResult` を返す構造は完全に除去し、Use Case 層は `value + error` / `error` を返す形に統一された
- DB 整理として `Upload` テーブルの削除（0005 → migration）と `Chapter` → `Route` 再設計（`currentChapter` TEXT → `currentRouteId` FK）が完了した

## フェーズ別進捗

### Phase 0. 方針固定

状態: 完了

- `docs/CLEAN_ARCHITECTURE_MIGRATION_PLAN.md` を作成し、移行方針を固定
- 「Hexagonal を実装スタイルとして使う実務寄り Clean Architecture」を採用
- 大規模一括リライトではなく、境界面から順に整理する方針を明文化

### Phase 1. 入口層の薄型化

状態: 完了

進んだこと:

- `ListGames` / `GetGameByID` / `CreateGame` / `UpdateGame` などの基本 API は service 呼び出し中心の薄い形になっている
- `session` / `memo` / `route` など主要 CRUD API も概ね service 経由で統一されている
- `DeleteSession` や `UpdateSessionName` など、adapter 側で最小限の戻り値整形に寄せる実装が進んだ
- `App.Database` フィールドを削除し、`internal/app` から `db.Repository` 直接参照を除去した
- `MemoCloudService` / `MaintenanceService` を導入し、厚かった `api_memo_cloud.go` / `api_maintenance.go` の処理を移管した
- サービス再構築処理を `configureServices` に集約し、起動時と復元後の結線を統一した
- `api_adapter_test.go` に game / session / route / credential のサービスエラー変換テストを追加し、adapter 層の検証範囲を広げた（cloud-sync / memo-cloud と合わせて計13テスト）

残課題:

- Wails adapter と Use Case の責務分離をさらに明確にする余地がある

### Phase 2. Port interface の導入

状態: 完了

interface 化済み service:

- `GameService`
- `SessionService`
- `MemoService`
- `RouteService`
- `CredentialService`
- `CloudService`
- `MemoCloudService`
- `CloudSyncService`
- `ScreenshotService`
- `ProcessMonitorService`

進んだこと:

- `internal/services/repositories.go` に各ユースケース向け interface を導入
- service コンストラクタが具象の `*db.Repository` ではなく interface を受け取る形へ移行
- `CloudService` / `MemoCloudService` が高レベルな cloud object store port を介して外部 I/O を扱う形へ移行
- `CredentialService` は credential store interface に対して単体テスト可能な状態を維持
- fake repository による単体テストが書ける状態を確立
- service 単体テストで DB なしに主要ユースケースと cloud / credentials の振る舞いを検証できるようになった

残課題:

- DB 実装と service の結線を確認する統合テストはまだ薄い
- ScreenshotService の capture 処理は `captureFunc` 注入でテスト可能になったが、保存パス組み立てやログなど周辺の OS / filesystem 依存 port 化は引き続き整理余地がある

### Phase 3. Use Case 層の明確化

状態: 完了

進んだこと:

- `services` から `result.ApiResult` 依存を除去し、service は `value + error` / `error` を返す形へ移行
- `app` 側へ API レスポンス整形を戻し、adapter と usecase の責務を分離
- `game` / `session` / `memo` / `route` / `credential` / `cloud` / `maintenance` / `memo cloud` の返り値境界を統一
- `internal/services/repositories.go` を中心に、ユースケースごとの依存境界が読み取りやすくなった
- t_wada TDD に倣い、既存の振る舞いを固定するテストを追加してから構造変更する方針に更新

残課題:

- package はまだ `internal/services` に集約されたまま
- `credential` / `cloud` などはユースケース境界がまだ粗い
- `infrastructure` への物理再配置は Phase 5 で完了。`usecase` / `domain` 相当の再配置はまだ未着手

### Phase 4. 複雑機能の分割

状態: 進行中

#### CloudSyncService

進んだこと:

- repository 境界を `CloudSyncRepository` に分離
- 同期判定と周辺ロジックを小さな関数へ分割
- 追加済みの主な helper:
  - `determineGameSyncAction`
  - `cloudMetadataToMap`
  - `copyCloudGameMap`
  - `collectUnionGameIDs`
  - `syncSingleGame`
  - `composeSyncedLocalGame`
  - `composeCloudGameMetadata`
  - `composeCloudSessions`
  - `prepareGameSyncState`（セッション統合とマージ済みゲーム情報の準備を集約）
  - `syncUploadPath` / `syncDownloadPath` / `syncSkipPath`（同期方向ごとの処理を分離）
- upload / download の表現変換ロジックを I/O なしでテストできるようになった
- `loadLocalGames` の全件取得・指定ゲームなしの振る舞いをテストで固定
- S3 ストレージ、画像ファイル書き込み、画像ロードの差し替えポイントを導入済み
- `syncExistingGamePair` の全パス（upload / download / skip）の失敗系テストを追加
- `sync` レベルの `LoadMetadata` 失敗・`SaveMetadata` 失敗・ループ内失敗をテストで固定
- `syncExistingGamePair` を `prepareGameSyncState` + `syncUploadPath` / `syncDownloadPath` / `syncSkipPath` に分割し、upload/download/skip 分岐とセッション統合の混在を解消した
- `cloud_sync_paths_test.go` を新設し、`prepareGameSyncState` / `syncUploadPath` / `syncDownloadPath` / `syncSkipPath` を `syncExistingGamePair` を経由せず直接呼び出す単体テストを追加（`shouldSaveMetadata` の切り替わりや `SkippedGames` 集計、セッション未変更時の `SaveSessions` スキップなど、各関数固有の分岐を計14テストで固定）

まだ重い部分:

- ファイル全体はまだ 1200 行超（1272 行）で、責務分割は継続が必要

#### ProcessMonitorService

進んだこと:

- repository 境界を interface 化
- Windows パス前提の helper を導入し、Linux 上でも Windows 形式のパスでテスト可能にした
- `processProvider` を差し替え可能にして、主要な状態遷移とプロセス判定をテスト可能にした
- 自動検出 OFF、hotkey 対象選択、process snapshot のテストを追加

残課題:

- 監視ループ全体の統合的な振る舞いはまだ十分に押さえられていない
- `checkProcesses` / `saveAllActiveSessions` を中心に、単一 service へ責務がまだ集中している

#### ScreenshotService

進んだこと:

- repository 境界を interface 化
- 基本的な異常系とパス生成のテストを追加
- capture 処理を `captureFunc` として注入可能にし、`CaptureGameScreenshot` の正常系・異常系（game not found / `ErrNoNewScreenshot` / capture error / 成功時のパス）をテストで固定した

残課題:

- 保存パスの組み立てやログ出力など、capture 以外の周辺責務のさらなる分離

### Phase 5. Infrastructure の再配置

状態: 完了

進んだこと:

- `internal/db` → `internal/infrastructure/db` へ移動
- `internal/storage` → `internal/infrastructure/storage` へ移動
- `internal/credentials` → `internal/infrastructure/credentials` へ移動
- 全 import パスを一括更新（44ファイル）
- パッケージ名は維持（`db`, `storage`, `credentials`）、import パスのみ変更

残課題:

- `internal/memo`（ファイルシステム操作）は現時点では移動せず維持
- `internal/models` の `domain/` 相当への再配置は未着手（27ファイルへの影響があり、現状は優先度低）

## テスト進捗

### 追加済みの主な単体テスト

- `game_service_test.go`
- `session_service_test.go`
- `memo_service_test.go`
- `route_service_test.go`
- `credential_service_test.go`
- `cloud_service_test.go`
- `memo_cloud_service_test.go`
- `maintenance_service_test.go`
- `cloud_sync_service_test.go`
- `cloud_sync_logic_test.go`
- `screenshot_service_test.go`
- `process_monitor_service_test.go`
- `windows_path_test.go`

`internal/app` 側の現状:

- `api_maintenance_test.go` に加えて `api_adapter_test.go` を整備し、game / session / route / credential / cloud-sync / memo-cloud のサービスエラー変換テストを計13本追加した
- adapter 層のエラー変換は概ねカバーできたが、成功系の戻り値整形や通常 API 全体の網羅はまだ限定的

2026-06-05 に追加した主なテスト:

- Cloud sync (`syncExistingGamePair` 失敗系):
  - `loadCloudSessions` 失敗
  - セッション差分あり時の `upsertMergedLocalSessions` 失敗
  - upload パスの `buildCloudGame` 失敗（session save error）
  - download パスのクラウド `SaveSessions` 失敗
  - download パスの `applyCloudGame` (UpsertGameSync) 失敗
  - skip パスのクラウド `SaveSessions` 失敗
  - skip パスの `applyCloudGame` 失敗
  - skip パスでセッション未変更でもゲームを適用する振る舞いの確認
- Cloud sync (`sync` レベル):
  - `LoadMetadata` 非404エラー時の中断
  - `SaveMetadata` 失敗時のエラー伝播（同期後でもメタデータ保存失敗をエラーとして返す）
  - `syncSingleGame` ループ内失敗時にメタデータが保存されないことの確認

### 現在の評価

良い点:

- 移行済み service については、具象 DB 実装なしでユースケースを検証できる
- `CloudSyncService` の純粋ロジック・失敗系は以前より明確にテストできる
- Windows 実行前提の path 判定を Linux 上のテストで扱えるようになった
- memo / session / game の副作用と集計更新について、リファクタリング前の安全網が増えた
- backend 全体の Go テスト、lint、format を通せる状態は維持されている

不足している点:

- DB 実装を使う統合テストが不足している（`MaintenanceService` 以外は概ね fake repository 中心）
- frontend 側は今回の移行に対応する新規テスト追加は限定的

## 全体評価

- Phase 1 は完了
- Phase 2 は完了
- Phase 3 は完了
- Phase 4 は `CloudSyncService` を中心に進行中
- Phase 5 は完了

要するに、現在の移行は「`app` 層からの direct DB 依存除去」「厚い adapter の service 移管」「`services` からの `ApiResult` 除去」「DB 整理（Upload 削除・Chapter→Route 再設計）」「CloudSyncService 主要失敗系テスト追加と責務分割」「`internal/app` adapter テスト強化」「Screenshot capture の injectable 化」「infrastructure パッケージの物理再配置」まで完了した。

## 次の優先事項

1. `CloudSyncService`: 引き続き 1200 行超のファイルの分割（`syncSingleGame` / `sync` / `buildCloudGame` 周辺など、まだ大きな関数が残る）
2. `internal/models` の `domain/` 相当への再配置（現時点では優先度低）
3. DB 実装を使う統合テストの整備
4. `Game.playStatus` / `lastPlayed` / `clearedAt` の意味整合（状態モデルの定義）
