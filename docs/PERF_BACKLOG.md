# Performance Backlog

`/simplify` と `/code-review`（Angle F: efficiency）が指摘した効率課題。
**まず計測してボトルネックを確定**してから着手する（早すぎる最適化を避ける）。

優先順は「ユーザー体感への影響」×「修正コスト」で並べてある。

---

## P1. ProcessMonitor の 2 秒 tick での全件正規化

- **場所**: `internal/services/process_monitor_service.go`
  - `autoAddGamesFromDatabase`（L588 周辺）が tick ごとに `ListGames` 全件取得
  - `matchGameProcess` / `updateMonitoredGameState` が監視中ゲーム × 全プロセスで
    `normalizeProcessToken` を毎回再計算
- **規模感**: 200 ゲーム × 200 プロセス × 2 秒 tick =
  毎分 ~120k SQLite read、~24M トークン正規化
- **方針**: addMonitoredGame / autoAddGamesFromDatabase 時点で
  `(game.ID → 正規化済みトークン)` をキャッシュし、`UpdateGame` で invalidate。
  ListGames は 30 秒間隔等の遅い tick で更新。
- **計測**: macOS では使われないコードパス（A7 参照）なので、まず Windows で
  実機計測。pprof で CPU/メモリ取って判断。

## P2. saveSession でセーブツリーを二度ハッシュ

- **場所**: `internal/services/process_monitor_service.go::saveSession`
  → `buildSaveTree`（SHA-256 全ファイル）→ `cloudSync.Push`（内部で `buildSaveSnapshot`
  がもう一度全ファイル SHA-256）
- **規模感**: 5,000 ファイルの RPG-Maker セーブで、セッション終了が 2 倍待ち時間
- **方針**: スナップショットを 1 度作って Push に渡す形に。
  または `buildSaveTree` 呼び出し自体を削除（Push が `localSyncHead` を設定するので
  `LocalSaveHash` は不要かもしれない）
- **注意**: `LocalSaveHash` の意味（同期判定で実際に参照されているか）を
  まず Audit。**現状未参照の dead field の可能性**（`/code-review` Angle B-B6 が指摘）。

## P3. `pullDownloadSaves` の単一スレッド hash

- **場所**: `internal/services/content_sync_service.go::pullDownloadSaves`
- **問題**: ダウンロードは `S3UploadConcurrency` で並列化されているが、
  ローカル差分判定のための `hashFileStream` が単一 goroutine 直列。
- **規模感**: 2,000 ファイル中 5 個だけ更新のケースで、ダウンロード開始前の
  2,000 回直列 SHA-256 がボトルネック
- **方針**: `errgroup` で並列化（concurrency は同じ knob を流用）

## P4. `MemoCloudService` の N+1 と AWS config 再生成

- **場所**: `internal/services/memo_cloud_service.go::syncCloudToLocal`（N+1）／
  `internal/services/cloud_common.go::storageCloudObjectStore`（config 再生成）
- **問題**:
  - cloudMemos ループ内で `GetMemoByID` を 1 件ずつ叩いている → 300 メモで 300 クエリ
  - `ListObjects` / `UploadBytes` / `DownloadObject` の各呼び出しで
    `storage.NewClient` → `awsconfig.LoadDefaultConfig`（ファイル I/O＋環境変数）が再走
- **規模感**: 200 メモ同期で `~/.aws/{config,credentials}` を 200 回 open、
  `s3.Client` を 200 個生成
- **方針**:
  - サービスに `ListMemosByGame(gameID)` / `ListMemosByIDs(ids)` を追加して N+1 解消
  - `storageCloudObjectStore` が `*s3.Client` を 1 つキャッシュ（`ContentSyncService`
    の `newBlobStore` と同じパターン）
- **関連**: `REFACTOR_BACKLOG.md::A2`（MemoCloudService のリポジトリ注入）と
  同時にやると interface 設計が一度で済む。

## P5. `Home.tsx::resolveWarnings` のキーストロークごと IPC stat

- **場所**: `frontend/src/pages/Home.tsx`
- **問題**: `visibleGames` 参照変更のたびに `checkFileExists(exePath)` ＋
  `checkDirectoryExists(savePath)` を **直列 Promise** で全件叩く。
  検索のキーストローク・ソート変更・初回マウントの度に発火。
- **規模感**: 200 ゲーム = 400 IPC stat × デバウンスごと
- **方針**:
  1. 各ゲームの 2 stat を `Promise.all` で並列化（一次対処）
  2. `{exePath, saveFolderPath}` でメモ化して、未変化行はスキップ
  3. バックエンドに `checkPathsExist(paths[])` を追加して 1 IPC にまとめる（恒久対処）

## P6. `useCloudData.ts::buildCloudDataFromTree` の全件再構築

- **場所**: `frontend/src/hooks/useCloudData.ts`
- **問題**: 1 ゲーム詳細を展開するたびにクラウドツリー全体を再構築。
- **方針**: 直接更新（必要な subtree だけ書き換え）に書き換え。
  ただし差分検知の挙動がわずかに変わるため、テストでカバーしてから。

## P7. `GeneralSettings.tsx` のタブ atom 購読の絞り込み

- **場所**: `frontend/src/components/settings/GeneralSettings.tsx`
- **問題**: 親が複数の atom を購読しているため、無関係な atom 更新でも
  全タブが re-render する。
- **方針**: atom 購読をタブ側 component に降ろす（実測ベースで判断）。
- **規模感**: 体感差は要計測。設定画面はホットパスではないので優先度低。

---

## 計測の指針

- バックエンド: pprof（CPU / heap / goroutine）
- フロント: React Profiler / Lighthouse の Performance パネル
- I/O: `dtruss`（macOS）や Process Monitor（Windows）で stat / SQLite open 頻度を確認
- 「修正後にどの数値がどれだけ変わるか」を着手前にメモする
  （後から「効いたかどうか」が判断できる）
