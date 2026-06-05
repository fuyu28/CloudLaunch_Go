# CloudLaunch_Go Clean Architecture 進捗状況

最終更新: 2026-06-05

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

残課題:

- Wails adapter と Use Case の責務分離をさらに明確にする余地がある
- `internal/app` の adapter テストは依然として薄く、現状は `api_maintenance_test.go` が中心

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
- Screenshot / maintenance 周辺など、一部 OS / filesystem 依存の port 化は今後の整理余地がある

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
- `usecase` / `domain` / `infrastructure` への物理再配置は未着手

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
- upload / download の表現変換ロジックを I/O なしでテストできるようになった
- `loadLocalGames` の全件取得・指定ゲームなしの振る舞いをテストで固定
- S3 ストレージ、画像ファイル書き込み、画像ロードの差し替えポイントを導入済み
- `syncExistingGamePair` の全パス（upload / download / skip）の失敗系テストを追加
- `sync` レベルの `LoadMetadata` 失敗・`SaveMetadata` 失敗・ループ内失敗をテストで固定

まだ重い部分:

- `syncExistingGamePair` 内の upload/download/skip 分岐とセッション統合が同一関数に混在している
- ファイル全体はまだ 1200 行超で、責務分割は継続が必要

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

残課題:

- 保存や同期を含む周辺責務のさらなる分離

### Phase 5. Infrastructure の再配置

状態: 未着手

未着手の内容:

- `internal/db` の `infrastructure/db` 相当への再配置
- `internal/storage` の `infrastructure/storage` 相当への再配置
- `internal/credentials` や OS依存処理の `infrastructure/*` への再編

補足:

- 物理移動より依存方向の修正を優先しているため、現時点では未着手で問題ない

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

- `api_maintenance_test.go` のみ
- adapter 層の網羅は限定的で、通常 API の薄さを保証するテストはまだ不足

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

- `CloudSyncService` の `syncExistingGamePair` 内の責務分割は未完了
- DB 実装を使う統合テストが不足している
- `internal/app` の adapter 層のテストはまだかなり薄い
- frontend 側は今回の移行に対応する新規テスト追加は限定的

## 全体評価

- Phase 1 は完了
- Phase 2 は完了
- Phase 3 は完了
- Phase 4 は `CloudSyncService` を中心に進行中
- Phase 5 は未着手

要するに、現在の移行は「`app` 層からの direct DB 依存除去」「厚い adapter の service 移管」「`services` からの `ApiResult` 除去」「DB 整理（Upload 削除・Chapter→Route 再設計）」「CloudSyncService 主要失敗系テスト追加」まで完了した。次の主戦場は、CloudSyncService の責務分割と `internal/app` の adapter テスト強化である。

## 次の優先事項

1. `internal/app` の adapter テストを追加し、adapter 層の薄さを保証するテストを整備する
2. `CloudSyncService` の `syncExistingGamePair` をさらに分割し、責務を明確にする
3. Screenshot / maintenance 周辺の OS / filesystem 依存を必要に応じてさらに port 化する
4. `usecase` / `domain` / `infrastructure` への再配置を検討する
