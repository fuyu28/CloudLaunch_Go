# CloudLaunch Go移植: 未完了タスク

最終更新: 2026-01-27

## 1. ビルド/生成系（必須）

- Goビルドキャッシュの権限エラーを解消して `go test ./...` を通す  
  - エラー例: `/home/fuyu/.cache/go-build/...: permission denied`
- Wails のバインディング再生成  
  - 例: `wails generate`（API追加後の `frontend/wailsjs` を反映）
- `golangci-lint run ./...` を実行し、lintエラーがないことを確認

## 2. 仕様/実装の差分チェック（要確認）

- 設定/認証情報の永続化方針を確定  
  - 現状: accessKey/secretKeyのみ保存。bucket/endpoint/regionは環境変数依存  
  - UIから設定する場合は保存先（設定DB/ファイル）と読み込み処理の追加が必要
- エラーレポートAPIの扱い  
  - フロントの `errorReport` はコンソール出力のみ  
  - 既存のElectron版と同等のログ/送信を行うならGo側APIを追加
- 監視/自動トラッキングの実装方針確認  
  - 現状: `UpdateAutoTracking` と `GetMonitoringStatus` はフラグのみ  
  - 実際のプロセス監視/自動計測を移植する場合はサービス追加が必要

## 3. E2E/動作確認（推奨）

- メモ同期の双方向動作  
  - ローカル新規/更新/削除 → クラウド反映  
  - クラウド新規/更新 → ローカル反映  
  - タイトル変更時のファイル名一致確認
- セーブデータのアップロード/ダウンロード  
  - リモートパス `games/{title}/save_data` 前提での互換確認
