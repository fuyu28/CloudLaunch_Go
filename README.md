# CloudLaunch Go

CloudLaunch Go は、Electron 版の **CloudLaunch** を Go + Wails で移植することを目的としたデスクトップアプリです。
オリジナル版（CloudLaunch）の機能やユーザー体験を引き継ぎつつ、Go ベースのアーキテクチャで再構築しています。

## ステータス

- 開発中（機能移植・仕様整理の途中）
- まずはコア機能の移植と基盤整備を優先しています

## 目標（移植対象）

- セーブデータのクラウド同期（S3 互換ストレージ）
- ゲームライブラリ管理 / プレイ状況の可視化
- プレイメモ（Markdown）管理
- 既存 CloudLaunch での UX を維持した UI/UX

※ 現時点での実装範囲は移行中のため、詳細は `docs/` を参照してください。

## 開発環境

### 必要要件

- Go 1.25+（`go.mod` 準拠）
- Wails v2
- Bun
- （必要に応じて）`sqlc`

### セットアップ

```bash
# 依存関係の取得
bun install

# Wails 開発モード起動
wails dev
```

### ビルド

```bash
wails build
```

## 技術スタック

- **バックエンド**: Go, Wails v2
- **フロントエンド**: Vite + React (TypeScript), Tailwind CSS, DaisyUI
- **データベース**: SQLite（`sqlc` でコード生成）
- **クラウド**: AWS SDK for Go v2（S3 互換ストレージ）
- **認証情報**: OS のクレデンシャル管理（Windows: wincred）

## ディレクトリ構成

```
CloudLaunch_Go/
├── cmd/                 # CLI / app エントリ
├── internal/            # アプリの中核ロジック
├── frontend/            # Vite + React UI
├── build/               # Wails ビルド用アセット
├── docs/                # 設計メモ・計画
├── main.go              # Wails エントリポイント
├── sqlc.yaml            # sqlc 設定
└── wails.json           # Wails 設定
```

## 関連

- CloudLaunch（オリジナル / Electron 版）: `../CloudLaunch`

---

CloudLaunch Go は学習・実験用途の側面も含みます。重要なデータを扱う場合は必ずバックアップを取り、自己責任でご利用ください。
