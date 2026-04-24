# CloudLaunch_Go Clean Architecture 移行計画書

最終更新: 2026-04-24

## 1. 目的

CloudLaunch_Go を、Wails / SQLite / S3互換ストレージ / OS依存機能といった外部要因からユースケースを分離しやすい構造へ段階的に移行する。

本計画書では、教科書的な厳格分離を目指すのではなく、既存の実装資産を活かしながら以下を実現する。

- 依存方向を内側へ統一する
- Wails API 層を薄くする
- ユースケース層をテストしやすくする
- DB / Cloud / OS連携を差し替え可能にする
- 大規模な一括リライトを避け、段階的に移行する

## 2. 現状認識

現状のバックエンドは概ね以下の構造を持つ。

- `internal/app`: Wails から公開される API とアプリ初期化
- `internal/services`: 業務ロジック
- `internal/db`: SQLite への CRUD
- `internal/storage`: S3互換ストレージとの連携
- `internal/credentials`: 認証情報ストア
- `internal/models`: データモデル

現状はすでに層構造の素地がある一方、次の問題が残っている。

- `services` が `db.Repository` などの具象実装へ直接依存している
- `services` が `result.ApiResult` を返し、プレゼンテーション都合を知っている
- `app` 層が一部で `Database` を直接触っている
- 同期、監視、スクリーンショットなどの複合処理が単一サービスに集中している
- 境界が曖昧なため、ユースケース単位のテストが書きにくい

## 3. 採用方針

本プロジェクトでは、以下の実務寄り Clean Architecture を採用する。

- Entities: `models` を中心とした業務データ
- Use Cases: アプリケーション固有の処理
- Interface Adapters: Wails API、Repository 実装、Storage 実装、OS連携実装
- Frameworks / Drivers: Wails、SQLite、AWS SDK、OS API

実装方針としては Hexagonal Architecture の考え方を併用し、Use Case 層は port interface のみに依存させる。

## 4. 非目標

今回の移行では、以下は優先対象にしない。

- マイクロサービス化
- CQRS / Event Sourcing の導入
- すべての package を厳密に 4 層へ再配置すること
- フロントエンド全体の状態管理方式の全面刷新
- ドメインモデルの全面再設計

## 5. 目標アーキテクチャ

移行後の責務分担は以下を想定する。

### 5.1 層構成

1. `internal/app`
   Wails への公開 API、DTO 変換、入力検証、`ApiResult` 生成を担当する
2. `internal/usecase`
   アプリケーション固有のユースケースを担当する
3. `internal/domain`
   エンティティ、値オブジェクト、ドメインルール、Use Case が依存する interface を置く
4. `internal/infrastructure`
   SQLite、S3、認証情報保存、OS依存処理などの実装を置く

### 5.2 依存方向

依存方向は以下に統一する。

`app -> usecase -> domain`

`infrastructure -> domain`

Use Case 層は `db.Repository` や `storage` package の具象型を直接参照しない。

## 6. ディレクトリ方針

最終的な理想形は次を想定する。

```text
internal/
  app/
    api_*.go
    dto/
  domain/
    game/
    memo/
    cloudsync/
    shared/
  usecase/
    game/
    memo/
    session/
    cloudsync/
    settings/
  infrastructure/
    db/
    storage/
    credentials/
    processmonitor/
    screenshot/
    wails/
  logging/
  config/
```

ただし一括移行はしない。移行期間中は既存の `services`, `db`, `storage` を併存させてよい。

## 7. 移行原則

- 破壊的リネームを避ける
- 1フェーズごとに動作可能な状態を維持する
- 新規実装は原則として新アーキテクチャ側へ置く
- 既存コードの全面置換ではなく、境界面から順に整理する
- テスト容易性が高い箇所から優先して切り出す
- 複雑な機能ほど port 分離を優先する

## 8. フェーズ別移行計画

## Phase 0. 方針固定

目的: 移行作業中に設計判断がぶれないよう、用語と境界を固定する。

作業内容:

- 本計画書をレビューし、採用方針を確定する
- `services` を今後 `usecase` 相当に縮退させる方針を合意する
- package 命名規則を決める
- `ApiResult` は `app` 層で返す方針を明文化する

完了条件:

- ドキュメントに対してチーム内合意がある
- 新規機能追加時の配置方針が決まっている

## Phase 1. 入口層の薄型化

目的: Wails API 層を adapter として明確化する。

作業内容:

- `internal/app` から直接 `Database` を触る処理を排除する
- `api.go` 系ファイルでは Use Case 呼び出しと DTO 変換のみに責務を絞る
- `result.ApiResult` の生成場所を `app` 層へ寄せる
- `services` の戻り値をドメイン値または `error` に近づける

対象候補:

- ゲーム CRUD
- セッション更新
- メモ取得/更新
- 設定更新 API

完了条件:

- `internal/app` が `db.Repository` を直接参照しない
- API メソッドに業務判断がほぼ残っていない

## Phase 2. Port interface の導入

目的: Use Case 層が具象実装に依存しない状態を作る。

作業内容:

- `GameRepository`, `SessionRepository`, `MemoRepository` などを定義する
- `CredentialStore`, `CloudObjectStore`, `ProcessMonitor`, `ScreenshotStore` などの port を定義する
- `services` のコンストラクタを具象型ではなく interface を受け取る形へ変更する
- 既存の `internal/db`, `internal/storage`, `internal/credentials` に adapter 実装を追加する

優先度:

1. `GameService`
2. `SessionService`
3. `MemoService`
4. `CloudService`
5. `CloudSyncService`

完了条件:

- Use Case 相当の層が `internal/db` 具象型に依存していない
- in-memory fake で主要ユースケースをテストできる

## Phase 3. Use Case 層の明確化

目的: `services` を単なる便利箱ではなく、機能単位のユースケースへ整理する。

作業内容:

- `internal/services` の責務を見直し、機能単位で package を再編する
- 例:
  - `game` 系ユースケース
  - `session` 系ユースケース
  - `memo` 系ユースケース
  - `cloudsync` 系ユースケース
  - `settings` 系ユースケース
- 入出力を DTO とドメインモデルで整理する
- 共通 validation を Use Case 寄りの形へ整理する

完了条件:

- `services` 内の責務が機能単位で説明可能
- 1つの Use Case が複数の外部詳細を直接扱いすぎていない

## Phase 4. 複雑機能の分割

目的: 巨大サービスを分割し、変更容易性を上げる。

最重要対象:

- `CloudSyncService`
- `ProcessMonitorService`
- `ScreenshotService`

作業内容:

- `CloudSyncService` を以下の責務に分ける
  - 同期方針決定
  - ローカルゲーム収集
  - クラウドメタデータ入出力
  - セッション同期
  - 画像同期
  - 競合解決
- `ProcessMonitorService` を
  - プロセス列挙
  - ゲーム照合
  - セッション開始/停止
 へ分離する
- `ScreenshotService` を
  - キャプチャ
  - 変換
  - 保存
  - 同期
 へ分離する

完了条件:

- 300行超の肥大化したユースケースが段階的に縮小している
- 同期処理の主要分岐ごとにテストがある

## Phase 5. Infrastructure の再配置

目的: 外部依存の実装を adapter として明確にまとめる。

作業内容:

- `internal/db` を `infrastructure/db` 相当へ整理する
- `internal/storage` を `infrastructure/storage` 相当へ整理する
- `internal/credentials` を `infrastructure/credentials` 相当へ整理する
- OS依存実装を `infrastructure/processmonitor`, `infrastructure/screenshot`, `infrastructure/hotkey` へ整理する

注意点:

- import path の全面変更は差分が大きいため、package comment と段階リネームで対応する
- 物理移動より先に責務分離を終える

完了条件:

- adapter 実装の置き場所が一貫している
- domain / usecase から framework 依存が見えない

## Phase 6. フロントエンド境界の整流化

目的: バックエンドのユースケース分離に対応し、フロントも feature 単位で接続しやすくする。

作業内容:

- `window.api` 呼び出しを feature ごとの gateway に寄せる
- `Home.tsx` のようなページに集まりすぎたユースケース呼び出しを hook / service へ分離する
- `game`, `memo`, `cloud`, `settings` 単位の feature 構造へ整理する

完了条件:

- 画面コンポーネントがユースケースの調停役になりすぎていない
- Wails bridge が backend API の薄い写像になっている

## 9. 優先順位

最初に着手すべき順序は以下とする。

1. `app` 層から `Database` 直接参照を排除
2. `GameService`, `SessionService`, `MemoService` に interface 導入
3. `ApiResult` を `app` 層へ寄せる
4. `CloudSyncService` の分割
5. infrastructure 側の再配置

理由:

- 変更波及が比較的小さい
- テスト可能性の改善が早い
- 既存機能を止めずに進めやすい
- 最も重い `CloudSyncService` に着手する前に境界定義を固められる

## 10. リスク

- package 再編だけ先行すると差分が大きい割に価値が出ない
- interface を細かく切りすぎると実装が追いにくくなる
- `ApiResult` 剥離の途中でエラーハンドリングが二重化する
- 同期処理の分割時に挙動差分が混入しやすい
- Wails binding と frontend 側型定義の追従漏れが起きやすい

## 11. リスク対策

- 物理移動より依存方向の修正を優先する
- interface はユースケース起点で切り、過度に汎化しない
- フェーズごとに backend test と frontend 型整合を確認する
- `CloudSyncService` は分割前に回帰テスト対象を洗い出す
- 既存公開 API のシグネチャ変更はフェーズ単位で限定する

## 12. テスト戦略

各フェーズで最低限以下を確認する。

- `go test ./...`
- `bun run test`
- Wails binding 再生成が必要な場合は `wails generate`
- 手動確認:
  - ゲーム作成/更新/削除
  - セッション作成/更新/削除
  - メモ作成/同期
  - クラウド同期
  - 設定更新

Use Case 層については、段階的に mock ではなく fake adapter を使ったテストを増やす。

## 13. 完了条件

移行完了の判断基準は以下とする。

- `internal/app` が Wails adapter の責務に限定されている
- Use Case 層が `ApiResult` を返していない
- Use Case 層が DB / S3 / OS API の具象実装に依存していない
- 複雑機能が責務ごとに分割されている
- 主要ユースケースが adapter 差し替えでテストできる
- 新規機能追加時に配置ルールで迷いにくい

## 14. 直近の実行タスク

最初の 1 スプリントでは以下を実施する。

1. `internal/app` から `Database` 直接参照を洗い出す
2. `GameService` と `SessionService` の repository interface を定義する
3. `services` の戻り値から `ApiResult` を外す試験的リファクタを 1 系統で実施する
4. `CloudSyncService` の責務一覧を分解し、分割設計メモを作る
5. 回帰確認用の最小テスト観点を文書化する

## 15. 付記

この移行は「package 名をきれいにすること」が目的ではない。目的は、CloudLaunch_Go の主要価値であるゲーム管理、セッション記録、メモ管理、クラウド同期、OS連携を今後も安全に変更できる構造を作ることにある。
