# CloudLaunch_Go Clean Architecture 進捗状況

最終更新: 2026-05-04

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
- `services` が `result.ApiResult` を返す構造はまだ残っており、完全な Use Case 層分離は未完了

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
- `session` / `memo` / `chapter` など主要 CRUD API も概ね service 経由で統一されている
- `DeleteSession` や `UpdateSessionName` など、adapter 側で最小限の戻り値整形に寄せる実装が進んだ
- `App.Database` フィールドを削除し、`internal/app` から `db.Repository` 直接参照を除去した
- `MemoCloudService` / `MaintenanceService` を導入し、厚かった `api_memo_cloud.go` / `api_maintenance.go` の処理を移管した
- サービス再構築処理を `configureServices` に集約し、起動時と復元後の結線を統一した

残課題:

- `ApiResult` の生成責務はまだ `services` 側にも残っている
- Wails adapter と Use Case の責務分離をさらに明確にする余地がある
- `internal/app` の adapter テストは依然として薄く、現状は `api_maintenance_test.go` が中心

### Phase 2. Port interface の導入

状態: 完了

interface 化済み service:

- `GameService`
- `SessionService`
- `MemoService`
- `ChapterService`
- `UploadService`
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
- `game` / `session` / `memo` / `chapter` / `upload` / `credential` / `cloud` / `maintenance` / `memo cloud` の返り値境界を統一
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
  - `composeLocalPlaySession`
- upload / download の表現変換ロジックを I/O なしでテストできるようになった
- `loadLocalGames` の全件取得・指定ゲームなしの振る舞いをテストで固定
- S3 ストレージ、画像ファイル書き込み、画像ロードの差し替えポイントを導入済み

まだ重い部分:

- `applyCloudGame` の副作用の塊
- metadata 保存判断とエラーハンドリングの詳細分岐
- S3 I/O を伴う分岐の失敗系テスト
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
- `chapter_service_test.go`
- `upload_service_test.go`
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

2026-04-29 に追加した主な回帰テスト:

- Cloud sync:
  - ローカル全ゲームとセッションをまとめて読み込む
  - 指定ゲームが存在しない場合に空結果を返す
- Memo:
  - メモ作成時に DB とローカルファイルの両方へ反映する
  - メモ更新時にローカルファイルをリネームして本文を更新する
  - メモ削除時に DB レコードとローカルファイルを削除する
- Session:
  - セッション名更新時に trim し、ゲーム更新時刻 touch と合計時間再計算を行う
  - セッション章更新時に章を保存し、ゲーム更新時刻 touch と合計時間再計算を行う
- Game:
  - ゲーム更新時に入力を trim しつつプレイ集計を保持する
  - ゲーム一覧検索時に検索文字列を trim する
- Process monitor:
  - 自動検出 OFF のときに監視対象を追加しない
  - hotkey 対象は中断中より現在プレイ中のゲームを優先する
  - process snapshot は注入した process provider を使う

現在の coverage:

- `internal/services`: 44.5%
- `internal/result`: 100.0%

2026-05-04 の確認結果:

- `go test ./...` は通過
- `./scripts/run-all-lint-format.sh` は通過

### 現在の評価

良い点:

- 移行済み service については、具象 DB 実装なしでユースケースを検証できる
- `CloudSyncService` と `ProcessMonitorService` の pure な判定部分は以前より明確にテストしやすい
- Windows 実行前提の path 判定を Linux 上のテストで扱えるようになった
- memo / session / game の副作用と集計更新について、リファクタリング前の安全網が増えた
- backend 全体の Go テスト、lint、format を通せる状態は維持されている

不足している点:

- `CloudSyncService` の副作用を伴う失敗系テストがまだ薄い
- DB 実装を使う統合テストが不足している
- `internal/app` の adapter 層のテストはまだかなり薄い
- frontend 側は今回の移行に対応する新規テスト追加は限定的

## 全体評価

現状は、以下の段階に入っている。

- Phase 1 は完了
- Phase 2 は完了
- Phase 3 は完了
- Phase 4 は `CloudSyncService` を中心に進行中
- Phase 5 は未着手

要するに、現在の移行は「`app` 層からの direct DB 依存除去」「厚い adapter の service 移管」「`services` からの `ApiResult` 除去」までは完了した。次の主戦場は、重い service をさらに分割して Use Case 境界を明確にすることである。

## 次の優先事項

1. `CloudSyncService` の upload / download / metadata 保存失敗系テストを厚くする
2. `internal/app` の adapter テストと、必要最小限の DB 統合テストを追加する
3. Screenshot / maintenance 周辺の OS / filesystem 依存を必要に応じてさらに port 化する
4. `usecase` / `domain` / `infrastructure` への再配置を検討する
