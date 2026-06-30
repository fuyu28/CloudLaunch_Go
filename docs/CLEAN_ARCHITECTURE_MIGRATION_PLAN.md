# CloudLaunch_Go Clean Architecture 移行計画書

最終更新: 2026-04-29

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
- t_wada TDD に倣い、リファクタリング前に壊したくない振る舞いをテストで固定する
- テストは実装詳細ではなく、ユーザー価値・ユースケース・外部境界で観測できる振る舞いを優先する
- 1つの回帰テストを追加して green を確認してから、小さくリファクタリングするサイクルを基本とする

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

## 9. 現在の進捗と優先順位

2026-04-29 時点の進捗:

- Phase 0 は完了
- Phase 1 はほぼ完了
  - `internal/app` から `db.Repository` への直接依存は解消済み
  - Wails API 層は service 呼び出し中心へ薄型化済み
- Phase 2 は大部分完了
  - `GameService`, `SessionService`, `MemoService`, `ChapterService`, `UploadService`, `CloudSyncService`, `ScreenshotService`, `ProcessMonitorService` は repository interface 経由へ移行済み
  - fake repository によるサービス単体テストを追加済み
- Phase 3 は着手済み
  - `services` はまだ物理的には集約されたままだが、ユースケース単位の責務は見え始めている
  - `ApiResult` が service 層に残っており、adapter/usecase 境界の整理は未完了
- Phase 4 は進行中
  - `CloudSyncService` は同期判定、メタデータ変換、セッション変換、ローカルゲーム収集を小さな関数へ切り出し済み
  - `ProcessMonitorService` は process provider 差し替えと Windows path helper により、Linux 上でも主要判定をテスト可能
- Phase 5 と Phase 6 は未着手

現在のテスト状況:

- `internal/services` の単体テストは、主要 service の happy path / invalid input / repository error / 副作用整合性を中心に追加済み
- 2026-04-29 時点で `internal/services` coverage は 30.9%
- Cloud sync / memo / session / game / process monitor について、リファクタリング前に守るべき回帰テストを追加済み

次に着手すべき順序は以下とする。

1. `CloudSyncService` の副作用を伴う処理をさらに分割する
2. Cloud sync の upload / download / metadata 保存失敗などの回帰テストを追加する
3. `services` の戻り値から `ApiResult` を外し、`app` 層へ戻り値整形を寄せる
4. DB 実装を使う最小限の統合テストを追加する
5. infrastructure 側の再配置を検討する

理由:

- 入口層の薄型化と repository interface 導入はすでに進んでいる
- これ以上の構造変更では、重いサービスの振る舞い固定が最も重要になる
- `ApiResult` 剥離は影響範囲が広いため、回帰テストを厚くしてから進める
- 物理再配置は差分が大きいため、依存方向とテストの安全網を先に固める

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

t_wada TDD に倣い、リファクタリングでは以下の進め方を守る。

1. 既存実装で成立している振る舞いを、外部から観測できるテストとして書く
2. テストが green であることを確認する
3. そのテストを安全網として、1つずつ小さくリファクタリングする
4. リファクタリング後に `go test ./...` と必要な frontend test / lint を通す
5. 実装詳細を固定しすぎるテストは避け、ユースケースの期待結果・副作用・エラーを優先する

優先して固定するテスト観点:

- Cloud sync: upload / download / skip の分岐、metadata 保存判断、セッション同期、画像同期失敗時の扱い
- Memo: DB とローカルファイルの作成・更新・削除整合性、rollback
- Session / Game: プレイ時間、LastPlayed、CurrentChapter、更新後の集計
- Process monitor: プロセス照合、開始/中断/再開/終了、hotkey 対象選択
- App adapter: service 呼び出し、DTO 変換、`ApiResult` 生成

## 13. 完了条件

移行完了の判断基準は以下とする。

- `internal/app` が Wails adapter の責務に限定されている
- Use Case 層が `ApiResult` を返していない
- Use Case 層が DB / S3 / OS API の具象実装に依存していない
- 複雑機能が責務ごとに分割されている
- 主要ユースケースが adapter 差し替えでテストできる
- 新規機能追加時に配置ルールで迷いにくい

## 14. 直近の実行タスク

直近では以下を実施する。

1. `CloudSyncService` の S3 I/O 周辺を port 化し、upload / download を fake で検証できるようにする
2. Cloud sync の失敗系回帰テストを追加する
3. `MemoService`, `SessionService`, `GameService` のテストを必要に応じて補強しながら `ApiResult` 剥離に着手する
4. `internal/app` に adapter としての薄いテストを追加する
5. `usecase` / `domain` / `infrastructure` への物理再配置は、責務分離とテストが十分に固まってから実施する

## 15. 付記

この移行は「package 名をきれいにすること」が目的ではない。目的は、CloudLaunch_Go の主要価値であるゲーム管理、セッション記録、メモ管理、クラウド同期、OS連携を今後も安全に変更できる構造を作ることにある。
