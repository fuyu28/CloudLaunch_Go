# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# フルアプリ起動（開発モード）
wails dev

# Go テスト
go test ./...
go test ./internal/services/... -v -run TestGameService

# フロントエンドテスト
cd frontend && bun run test

# lint + format（タスク完了時に必ず実行）
./scripts/run-all-lint-format.sh

# ビルド
wails build
```

## Architecture

Clean Architecture で4層構成：

```
domain/      ← モデル・型定義のみ（外部依存なし）
infrastructure/ ← DB / S3 / 認証情報の実装
services/    ← ユースケース（repositoryインターフェースに依存）
app/         ← Wailsアダプター層（薄いラッパー）
```

### Wails API フロー

フロントエンドからバックエンドへの呼び出しは以下の経路を通る：

1. `frontend/src/wailsBridge.ts` — Wails自動生成バインディングをラップし `window.api` として公開
2. `internal/app/api*.go` — `result.ApiResult[T]` を返すWailsメソッド群（型は自動的に `frontend/wailsjs/go/` に生成）
3. `internal/services/` — ビジネスロジック本体

フロントエンドからは必ず `window.api` 経由でアクセスする。Wails生成コードを直接importしない。

### `result.ApiResult[T]`

Go→TypeScript の共通レスポンス型。全APIメソッドはこれを返す：

```go
type ApiResult[T any] struct {
    Success bool      `json:"success"`
    Data    T         `json:"data,omitempty"`
    Error   *ApiError `json:"error,omitempty"`
}
```

### Services のリポジトリ境界

`internal/services/repositories.go` にサービスごとのリポジトリインターフェースが定義されている。`db.Repository` はこれらをすべて実装しているが、サービスはインターフェースにのみ依存する。テストでは `fake*Repository` 構造体でインターフェースをモックする。

### DB・マイグレーション

- `internal/infrastructure/db/repository.go` に生SQLを直書き（sqlcは使わない）
- マイグレーションは `internal/infrastructure/db/migrations/` の連番SQLファイル
- `schema_migrations` テーブルで適用済みを管理（同じファイル名は1回しか実行されない）
- マイグレーションを追加したらファイル名を `0007_xxx.sql` のように連番で命名する
- dev DB は `tmp/app.db`（マイグレーション変更後はこのファイルを削除して再作成）

### プラットフォーム対応

Windows専用機能は `_windows.go` サフィックスのファイルに実装し、`_unsupported.go` でスタブを提供する（ホットキー、認証情報ストア、プロセス起動など）。

## Frontend

- `frontend/src/types/` — TypeScript型定義（`GameType`, `PlayStatus` 等）
- `frontend/src/wailsBridge.ts` — Wailsバインディングの唯一の入口点
- `frontend/src/state/` — Jotai atoms によるグローバル状態
- `frontend/src/hooks/` — ビジネスロジックを含むカスタムフック

## Key Domain Rules

- `playStatus` は DB に保存（`'unplayed'`/`'playing'`/`'played'`）。ユーザーが手動変更可能
- `clearedAt` がセットされると、サービス層が強制的に `playStatus = 'played'` に上書きする
- `playStatus` の検証は `domain.IsValidPlayStatus()` で行う
