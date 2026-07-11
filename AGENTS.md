# Repository Guidelines

CloudLaunch は Wails v2 + Go + React (Vite + TypeScript + Tailwind + DaisyUI) の
デスクトップアプリで、PC ゲームのセーブデータを S3 互換ストレージに
コンテンツアドレッサブル方式で同期する。

## プロジェクト構成

- `main.go` — Wails エントリポイント
- `internal/` — バックエンド（Clean Architecture 4層）
  - `domain/` — モデル・型定義のみ（外部依存なし）
  - `infrastructure/` — DB / S3 / 認証情報の実装
  - `services/` — ユースケース（repository インターフェースに依存）
  - `app/` — Wails アダプター層（薄いラッパー）
- `frontend/` — Vite + React UI（TS）。ビルド成果物 `frontend/dist` は Wails が埋め込む
- `build/` — Wails ビルド用のプラットフォーム固有アセット（NSIS installer 等）
- `docs/` — 設計メモとリファクタバックログ
- `scripts/` — lint/format 一括や screencap-cli 取得スクリプト

## 主要コマンド

```bash
# フルアプリ起動（開発モード）
wails dev

# フロントエンドのみの開発
cd frontend && bun run dev

# バックエンドテスト
go test ./...
go test ./internal/services/... -v -run TestGameService

# フロントエンドテスト
cd frontend && bun run test

# lint + format（タスク終了時は必ず実行）
./scripts/run-all-lint-format.sh

# プロダクションビルド
wails build
```

## アーキテクチャ

### Wails API フロー

フロントエンドからバックエンドへの呼び出しは以下の経路を通る:

1. `frontend/src/wailsBridge.ts` — Wails 自動生成バインディングをラップし
   `window.api` として公開
2. `internal/app/api*.go` — `result.ApiResult[T]` を返す Wails メソッド群
   （型は `frontend/wailsjs/go/` に自動生成される）
3. `internal/services/` — ビジネスロジック本体

フロントエンドからは必ず `window.api` 経由でアクセスする。
Wails 生成コード（`wailsjs/`）を直接 import しない。

### `result.ApiResult[T]`

Go→TypeScript の共通レスポンス型。全 API メソッドはこれを返す:

```go
type ApiResult[T any] struct {
    Success bool      `json:"success"`
    Data    T         `json:"data,omitempty"`
    Error   *ApiError `json:"error,omitempty"`
}
```

### Services のリポジトリ境界

`internal/services/repositories.go` にサービスごとのリポジトリインターフェースが
定義されている。`db.Repository` はこれらをすべて実装しているが、サービスは
インターフェースにのみ依存する。テストでは `fake*Repository` 構造体で
インターフェースをモックする。

### DB・マイグレーション

- `internal/infrastructure/db/repository.go` に生 SQL を直書き（sqlc は使わない）
- マイグレーションは `internal/infrastructure/db/migrations/` の連番 SQL ファイル
- `schema_migrations` テーブルで適用済みを管理（同じファイル名は1回しか実行されない）
- マイグレーションを追加したらファイル名を `0007_xxx.sql` のように連番で命名する
- dev DB は `tmp/app.db`（マイグレーション変更後はこのファイルを削除して再作成）

### プラットフォーム対応

Windows 専用機能は `_windows.go` サフィックスのファイルに実装し、
`_unsupported.go` でスタブを提供する（ホットキー、認証情報ストア、
プロセス起動、スクリーンショット等）。

## Frontend

- `frontend/src/types/` — TypeScript 型定義（`GameType`, `PlayStatus` 等）
- `frontend/src/wailsBridge.ts` — Wails バインディングの唯一の入口点
- `frontend/src/bridge/` — 各ドメインごとの `window.api` サブモジュール
- `frontend/src/state/` — Jotai atoms によるグローバル状態
- `frontend/src/hooks/` — ビジネスロジックを含むカスタムフック
- `frontend/src/components/` — UI コンポーネント（`common/`, `cloud/`, `game/`, `memo/`, `settings/`）
- `frontend/src/pages/` — 画面単位のトップコンポーネント

## ドメインルール

- `playStatus` は DB に保存（`'unplayed'` / `'playing'` / `'played'`）。
  ユーザーが手動変更可能
- `clearedAt` がセットされると、サービス層が強制的に
  `playStatus = 'played'` に上書きする
- `playStatus` の検証は `domain.IsValidPlayStatus()` で行う

## コーディングスタイル

- Go は `gofmt` に従う（タブ、標準スタイル）。パッケージ名は短く lower-case、
  エクスポート識別子は `CamelCase`
- Frontend は TypeScript + React、既存ファイルパターンとフォーマットを踏襲
- `internal/` は小さく焦点を絞ったパッケージにする。UI ステート/ロジックは
  対応コンポーネントの近くに置く
- 日本語のコメントを尊重（既存コードに合わせる）
- 「エラー」を発話する UI 文言はユーザ視点で書く

### コメント方針（how / what / why / why not）

- コード本体は **how**（実装そのもの）
- テストは **what**（何を検証するか。関数名・`it(...)` 文言）
- コミット本文は **why**（なぜ変えたか。動機・トレードオフ）
- インラインコメントの正規は **why not**（別案を採らない理由・やってはいけない制約）。
  一文は why not に寄せる
- **一眼見てわからない複雑な処理**は、手順やデータの流れの説明（how）を書いてよい。
  複数行にして「何をしているか／どの順か／どこまで保証するか」を残す。
  自明な次行の復唱（例: `putBlobs` の直前に「アップロードする」だけ）は書かない
- **ファイル説明は what で OK、かつ本番ファイルでは必須:**
  - Go: `package` 直前のコメント（build tag の直後）。ファイルの役割を 1〜数文
  - Frontend: 先頭の `/** @fileoverview ... */`
  - 対象は本番 `.go` / `.ts` / `.tsx`（`*_test.go`・`*.test.ts(x)`・生成物は除外可）
  - 各 exported 記号への godoc / JSDoc 網羅は必須にしない（既存の記号コメントは残してよい）

## テスト

- Go テストは `*_test.go` 命名でパッケージと同居
  （例: `internal/result/result_test.go`）
- Frontend テストは Vitest。`*.test.ts(x)` または `*.spec.ts(x)` で
  コンポーネント近傍に配置
- データハンドリング、ストレージ、UI フローに影響する新規挙動は
  必ずテストでカバーする
- **タスク終了時は必ず `./scripts/run-all-lint-format.sh` を実行**する

## コミット / PR

- Conventional Commits を採用（`feat(scope):` / `fix(scope):` / `refactor(...):`
  / `chore(release):` / `build(...):`）
- サブジェクトは日本語で簡潔・命令形（既存履歴に合わせる）
- 本文には「なぜそう変えたか」（動機・トレードオフ）を書く。コードから読める
  「何をしたか」は最小限
- PR には概要、テスト計画、UI 変更ならスクリーンショットを添える
- `Co-Authored-By: Claude ...` はコミット末尾に付ける
- **`main` への取り込みは必ず PR 経由**（ブランチ push → `gh pr create` →
  明示指示があるときだけ merge）。`main` へ直接 merge して push しない
