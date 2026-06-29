# Simplify Pass Notes

> **⏸ 再開ポイント（2026-06-30 時点で中断）**
> - 作業ブランチ: `refactor/simplify-pass`（`refactor/ui-readability` から分岐）。作業ツリーはクリーン。
> - 完了済み: **G1**(`cf7bba4`) / 検討記録(`00248ce`) / **G2**(`920c8b3`)。各コミットは `go test ./... ` と
>   `./scripts/run-all-lint-format.sh` を通過済み。
> - **次にやること: G3（infrastructure: `internal/infrastructure/db/repository.go` / `storage/*` / `credentials/*`）から再開。**
> - 再開手順:
>   1. `git switch refactor/simplify-pass` でこのブランチに戻る。
>   2. このファイル末尾の「G3〜G10」と各グループの「見送った」メモを確認。
>   3. 対象グループを4観点（reuse/simplification/efficiency/altitude）で並列レビュー → 適用 →
>      `go test`(or `bun run test`) + lint → グループ単位でコミット → このファイルに追記。
> - **G3 で必ず再検討する持ち越し**（G2から）:
>   - `ExportGameData` の N+1（per-game `ListPlaySessionsByGame`）→ `WHERE game_id IN (...)` バッチ化。
>   - `route_service.UpdateRouteOrders` の逐次 UPDATE → バッチ UPDATE。
>   - いずれも `repository.go` にバッチメソッドを足せるか確認する。
> - **G4 で必ず再検討する持ち越し**（G2から）:
>   - `SessionMutationResult` が app層の関心（`gameId` での async sync）をサービス層に持ち込んでいる件。
>   - `MemoCloudService` がサービス実体（`*GameService`/`*MemoService`）に依存している件。
> - 進捗トラッキング: タスク #1,#2 完了 / #3〜#10 pending。


`refactor/ui-readability` → `main` の全差分（約1.6万行）を対象に、コード品質改善
（reuse / simplification / efficiency / altitude の4観点）を順番に適用していく作業の記録。

- 作業ブランチ: `refactor/simplify-pass`（`refactor/ui-readability` から分岐）
- 進め方: 論理グループ単位で「4観点レビュー → 適用 → `go test` / lint → コミット」
- バグ修正は対象外（それは `/code-review` の役割）。**品質改善のみ**。

## 適用 / 見送りの判断基準

**適用するもの**
- 挙動を変えない重複削減・簡潔化（ヘルパー抽出、ガード反転、デッドコード削除）
- 既存パターンに揃える形の効率改善（結果が同一になる並列化など）

**見送るもの（このパスでは触らない）**
- 意図した挙動を変えてしまう変更
- レビュー対象グループの外（別レイヤー・別ファイル）に大きく波及する変更
- 誤検知と判断したもの

見送ったものは「将来やるなら」のメモとして各グループに残す。

---

## G1: services コア同期

対象: `internal/services/content_sync_service.go` / `content_hash.go` / `cloud_common.go`
コミット: `cf7bba4`

### 適用した

| 観点 | 内容 |
|------|------|
| Simplification | `content_hash.go`: `validateSaveDir` / `walkSaveFiles` を抽出し、`buildSaveTree` / `buildSaveSnapshot` / `planDeletions` の3箇所で重複していた `filepath.Walk`（symlink/dir/rel/ToSlash）とディレクトリ検証を共通化 |
| Simplification | `pushCheckRemoteHead`: `if !force {…}` をガード反転（`if force { return }`）してネストを1段削減 |
| Simplification | `pullDownloadSaves`: `onProgress != nil` の二重判定を1つに統合し、`alreadyDone` をブロック内へ |
| Simplification / Reuse | `buildCloudGameView` と `buildCloudGameSummary` で重複していた HEAD→commit→game.json(title) 読み取りを `loadCloudCommit` に抽出（約30行削減） |
| Reuse / Efficiency / Altitude | `LoadCloudMetadata` / `ListCloudGameViews` / `ListCloudGameSummaries` の手書き並列 fan-out（semaphore+WaitGroup）が3箇所重複していたのを `fanOutGames[T]` ジェネリックに統合。ついでに**逐次だった `ListCloudGameViews` も並列化**（効率改善） |

### 見送った（将来の課題）

いずれも Altitude 観点で「サービス層に S3/インフラの詳細が漏れている」という指摘。
正しいが、`contentBlobStore` インターフェース拡張・テスト fake 更新・`internal/infrastructure/storage` や
`internal/domain` への波及を伴い、**G1の範囲を超える**ため見送り。

1. **`buildCloudGameView` が具象型 `*s3BlobStore` を受け取り `storage.ListObjects` を直接呼ぶ**
   （`bstore.client` / `bstore.bucket` に直アクセス）。
   - 深い修正案: `contentBlobStore` に `objectSizes(ctx, gameID)` を足し、サイズ解決を
     `s3BlobStore` 側へ封じ込めて、引数をインターフェースに戻す。
2. **S3 キーのプレフィックス構築がサービス層にある**
   （`DeleteFromCloud` / `buildCloudGameView` の `fmt.Sprintf("games/%s/…")`）。
   - 深い修正案: `deleteByPrefix(prefix)` を `deleteGameData(gameID)` のような意味のある
     メソッドに置き換え、キー形式をインフラ層に封じる。
3. **`storage.BlobKind*` 定数がサービス層に漏れている**（`BlobKindCommit` 等を多用）。
   - 深い修正案: `BlobKind` を `domain` へ移すか、サービス層の enum をインフラ層が変換する。
   - 影響範囲が `storage` と `domain` 両方なので、レイヤー再設計として別途扱う。

4. **`hashBytes`（services）と `blobHashBytes`（infrastructure/storage/blob_store.go）の重複**
   （どちらも sha256→hex の3行）。
   - レイヤーを跨ぐ共通化になるため、**G3（infrastructure）で合わせて判断**する。

---

## G2: services その他

対象: `maintenance_service.go` / `memo_cloud_service.go` / `route_service.go` ほか既存変更
コミット: （このグループのコミット）

### 適用した

| 観点 | 内容 |
|------|------|
| Simplification | `route_service.go`: 「`requireNonEmpty` → 警告ログ → `newServiceError`」の検証パターンが7箇所重複していたのを `requireField` メソッドに集約（約4行×7→1行×7） |
| Simplification | `memo_cloud_service.go`: 4メソッドで重複していた「S3設定解決 → エラーlog → `newServiceError`」を `resolveS3OrError` ヘルパーに集約 |
| Simplification | `maintenance_service.go`: `applyRestoredAppData` と `recoverAppDataFromRollback` で完全重複していた DB再オープン+ランタイム再開フック呼び出し（10行）を `reopenAndResume` に集約 |
| Altitude | `maintenance_service.go` に局所定義されていた `MaintenanceRepository` を `repositories.go` に移し、全リポジトリ境界の single-source を維持 |

### 見送った（将来の課題）

1. **`memo_cloud_service.go` の Details append（`recordSyncError` 化）** — `resultData.Details = append(..., fmt.Sprintf(...))` が10箇所以上。
   ヘルパー化で意図は明確になるが行数削減効果が小さく、置換箇所が多い割にリスクが上回るため見送り。
2. **`memo_cloud_service.go` L111 のキー直構築**（`fmt.Sprintf("games/%s/memo/%s", …)`）。
   他は `memo.BuildMemoPath()` を使うが、ここはタイトルでなく**ファイル名**ベースのため引数が合わず据え置き。
   memo パッケージにファイル名版ヘルパーを足すなら統一できる。
3. **`maintenance_service.go` の trim+空チェック3箇所** — エラーメッセージが各々異なり、汎用ヘルパーにしても綺麗にならないため見送り。
4. **効率（Efficiency）系はすべて見送り**:
   - `ExportGameData` の per-game `ListPlaySessionsByGame`（N+1）→ `WHERE game_id IN (...)` バッチ化はリポジトリ拡張が必要（**G3 で repository を見るときに再検討**）。
   - `route_service.UpdateRouteOrders` の逐次 UPDATE → バッチ UPDATE も同様にリポジトリ拡張が必要。
   - `GetCloudMemos` の `games/` 全列挙＋メモパスフィルタ、単一ゲーム同期での全ゲーム取得 → 挙動が変わりうるため据え置き。
5. **Altitude（設計）系の大物は見送り**（app層・コンストラクタ・新インターフェースへ波及するため）:
   - **`SessionMutationResult` が app層の関心（`gameId` による async sync）をサービス層へ持ち込んでいる**。
     深い修正は `(*domain.PlaySession, error)` を返す等。app層（`api.go`）と合わせて **G4 で再検討**。
   - **`MemoCloudService` が `*GameService` / `*MemoService` に依存**（リポジトリ境界でなくサービス実体）。
     正しくは Game/Memo の repository インターフェースに依存すべきだが、コンストラクタ・app層の組み立て変更を伴うため見送り。
   - `wrapServiceError`（memo_cloud 内のみ使用）を `service_error.go` へ寄せる案は、現状単一ファイル利用のため据え置き。

## G3: infrastructure

対象: `internal/infrastructure/db/repository.go` / `db.go` / `migrations.go` /
`storage/*` / `credentials/*` + G2持ち越し（バッチメソッド）
コミット: （このグループのコミット）

### 適用した

| 観点 | 内容 |
|------|------|
| Reuse / Altitude（持ち越し） | `internal/util/hash.go` に `Sha256Hex` を新設し、`services/content_hash.go` の `hashBytes` と `storage/blob_store.go` の `blobHashBytes` から呼ぶよう統合（G1からの持ち越し） |
| Reuse | `blob_store.go` の `PutBlob` / `PutBlobs` 内ワーカーで重複していた `client.PutObject(...)` 呼び出し2箇所を、既存の `UploadBytes()` 経由に統一 |
| Simplification | `repository.go`: 同じ SELECT カラム列を Game(4箇所) / Memo(5箇所) / Route(3箇所) / PlaySession(3箇所) で繰り返していたのを `*SelectCols` パッケージ定数化（列追加時の更新漏れ防止） |
| Simplification | `repository.go`: 5つのListメソッドで重複していた「QueryContext → defer Close → loop → scan → append」を `queryAll[T]` ジェネリックヘルパーに集約（各メソッドが10行台→1行） |
| Simplification | `repository.go`: `normalizeSortDirection` の switch を `strings.EqualFold` ベースの if に簡素化 |
| Efficiency（G2持ち越し） | `repository.go` に `ListPlaySessionsByGames(ctx, gameIDs)` を新設し、`MaintenanceService.ExportGameData` の per-game N+1 をバッチ1クエリに（`WHERE gameId IN (?, ?, ...)`） |
| Efficiency（G2持ち越し） | `repository.go` に `UpdateRouteOrders(ctx, items)` を新設し、`RouteService.UpdateRouteOrders` の逐次 UPDATE を単一トランザクションに |
| Altitude | `domain.RouteOrderItem` を新設し、サービス層の `RouteOrderUpdate`（API入力）と分離 |

### 見送った（将来の課題）

1. **`ApplyPullResult` がリポジトリにある（Altitude #1）** — 「存在しない Route 参照は NULL に正規化」という同期プロトコル固有のビジネスルールがリポジトリ層に混入している。
   深い修正案: サービス層に「sync transaction executor」を作り、リポジトリには個別の CRUD（`UpsertGameSync` / `DeletePlaySessionsByGame` / `UpsertPlaySessionSync`）だけ残す。
   - 影響範囲: サービス層・トランザクションスコープ・リポジトリインターフェース。大規模リファクタになるため見送り。
2. **`BlobKind` enum がストレージ層にある（Altitude #3）** — `commits` / `trees` / `meta` / `objects` という同期プロトコル概念。
   深い修正案: ストレージ層は `(gameID, key)` の不透明キーだけを扱い、サービス層が key を組み立てる。
   - 影響範囲: `PutBlob` / `GetBlob` / `ListBlobHashes` のシグネチャと全呼び出し元。波及が広いため見送り。
3. **`normalizeSortColumn` のホワイトリスト** — インフラ層に許可カラム名がある（軽い altitude 懸念）が、SQL インジェクション防御として infra に置く方が安全。現状維持。

### G4 で再検討する持ち越し（G2から）

これらは引き続き G4（app層）で再検討:
- **`SessionMutationResult` がサービス層に app の関心を持ち込んでいる** — `gameId` を返す理由は app の async sync 用。深い修正は app 層で `repository.GetPlaySessionByID(sessionID)` を呼んで `gameId` を取り出す等。
- **`MemoCloudService` が `*GameService` / `*MemoService` に依存** — サービス間依存。app 層のコンストラクタ組立を見直すタイミングで再検討。

## G4: app層

対象: `internal/app/api.go` / `api_sync.go` / `api_helpers.go` / `app.go` ほか
コミット: （このグループのコミット）

### 適用した

| 観点 | 内容 |
|------|------|
| Reuse / Simplification | `api_helpers.go` に `boolResult` を新設し、「サービス呼んで success なら `OkResult(true)`」だけの7メソッド（DeleteGame / UpdateRouteOrders / SetCurrentRoute / DeleteRoute / DeleteMemo / SaveCredential / DeleteCredential）を1行化 |
| Reuse / Simplification | `api_helpers.go` に `requireGameID[T]` を新設し、`api_sync.go` の5箇所で重複していた「trim → 空チェック → ErrorResult」を集約 |
| Simplification | `api.go`: `ReportLog` / `ReportError` で重複していた「TrimSpace 後に非空ならattrsに追加」を `appendIfNonEmpty` に集約（約30行削減） |
| Simplification | `api.go`: `UpdateScreenshotHotkey` / `UpdateScreenshotHotkeyNotify` で重複していた「stop → start → 失敗時 rollback → 再 start」を `applyHotkeyChange(operation, errMessage, rollback, attrs...)` に集約 |
| Simplification | `api.go`: `Pause/Resume/EndMonitoringSession` の3つで重複していた `ProcessMonitor == nil` 警告ログ＋エラー返却を `requireProcessMonitor` ヘルパーに集約 |

### 見送った（将来の課題）

1. **`SessionMutationResult` 経由のサービス→app関心バブルアップ（G2持ち越し）**
   - `SessionService.{Delete,UpdateSessionRoute,UpdateSessionName}` が `SessionMutationResult{GameID}` を返すのは app の async sync 用。
   - 深い修正案: サービスは `error` だけ返し、app 側は事前に `repository.GetPlaySessionByID(sessionID)` で gameID を取って `syncGameAsync` を呼ぶ。
   - 影響: 3つのサービスメソッド・テスト fake・app の3メソッド。中規模変更で挙動の境界（delete前にread）にも注意が必要。**今回は見送り、将来の独立した一発リファクタへ。**
2. **`MemoCloudService` がサービス実体に依存（G2持ち越し）**
   - 深い修正案: app.go で `GameRepository` / `MemoRepository` を直接 `MemoCloudService` に注入し、`gameService.GetGameByID` 等の呼び出しをリポジトリ呼び出しに置換。
   - 影響: コンストラクタ・既存呼び出し全部・テスト。**今回は見送り。**
3. **`api_maintenance.go` の DBスナップショット/サービスライフサイクルorchestration**
   - 現状は `MaintenanceRuntimeHooks` 経由でサービスから app のメソッドを呼び出している。これ自体は責務分離の妥協形だが、ホスト固有処理を app に置く現実的解。深掘りは G5（logging/domain）と一緒に再検討。
4. **`sync_coalescer.go` の debounce/coalesce ポリシーをサービスへ移管**
   - サービス層が「同期戦略」を持つべき、というのは正しい。ただし現状は app の起動順序とライフサイクルにきれいに統合されているため、移動するメリットがコストを下回ると判断し見送り。
5. **`api_cloud.go` の `buildGameDirectoryNode` ツリー構築**
   - ~70行のロジック。サービスに移すべき。**G9（frontend cloud + Cloud.tsx）で API 形状と一緒に再検討**。
6. **app.go の closure が `*App` を捕捉している（Efficiency #1）**
   - レビュアー指摘は GC 観点だが、`*App` はプロセス唯一のルートで GC されない。さらに `configureServices` が復元時にも呼ばれるため、値捕捉に変えると古いインスタンスを参照するリスク。**現状維持。**
7. **`UpdateScreenshotHotkey` の rollback で旧設定を二度 start する微小なムダ**
   - エラー経路のみ・ユーザー体感に影響なし。挙動を変えないので見送り。

## G5: logging / domain / main / config

対象: `internal/logging/*` / `internal/domain/*` / `internal/memo/content_hash.go` / `main.go` / `internal/config/*`
コミット: （このグループのコミット）

### 適用した

| 観点 | 内容 |
|------|------|
| Reuse | `memo/content_hash.go` の `CalculateContentHash` を `util.Sha256Hex` 経由に変更（G3で `util.Sha256Hex` を作った際の積み残し3箇所目を解消） |
| Simplification | `logging/logger.go`: 2箇所重複していたローテーション付きログオープン+stderr通知を `tryOpenRotatingLog` に集約 |
| Simplification | `logging/logger.go`: `dirErr == nil` 後の冗長な `strings.TrimSpace(appDataDir) != ""` 再チェックを削除（`ensureLogDir` 内で既に検証済み）。条件をガード反転して読みやすく |

### 見送った（将来の課題）

1. **`config/config.go` の `getEnv` / `getEnvBool` / `getEnvInt` の形が同じ** — レビュアー自身が「3つだけならむしろ現状の方が明快」と評価。ジェネリック/factoryで集約しても可読性は上がらない。**現状維持。**
2. **`main.go` の panic 再 throw と `logging/recover.go` の panic swallow** — 意図的に挙動を分けている（main は再 throw / バックグラウンド goroutine は swallow）。共通化すべきでない。
3. **domain 層には behavior が一切なく Clean Architecture 上クリーン** — 修正不要。

## G6: frontend bridge/*

対象: `frontend/src/bridge/*` (Wailsバインディングラッパー、19ファイル約1.3k行)
コミット: （このグループのコミット）

### 適用した

| 観点 | 内容 |
|------|------|
| Reuse | `helpers.ts` に共通定数 `DEFAULT_ERROR_MESSAGE = "エラー"` を追加し、既存の `toApiResult` / `toApiResultVoid` のフォールバック引数をデフォルト化 → 全モジュールで20+箇所の `, "エラー"` を削除 |
| Reuse | `helpers.ts` に `toApiResultArray<TItem, TOut>` を追加。`memo.ts` / `cloudData.ts` の `(result.data ?? []).map(fn)` パターン7+箇所を1行化 |
| Reuse / Simplification | `helpers.ts` に `getErrorMessage(error, fallback)` を追加。`erogameScape.ts` の同一エラー抽出ロジック2箇所と `game.ts` を集約 |
| Simplification | `memo.ts`, `cloudData.ts`, `file.ts`, `game.ts`, `erogameScape.ts` で残っていた手書きの `success ? {success:true,data:...} : {success:false,message:...}` ターナリ20+箇所を `toApiResult`/`toApiResultArray` で置換 |

### 見送った（将来の課題）

レビューで挙がった altitude 系（バリデーション/エラー処理を hooks 層へ移管、`processMonitor.ts` のサイレントエラーを ApiResult 化、`database.ts updatePlayStatus` の read-update-read 解消等）は、**G7（hooks）で hooks 側と一緒に見直す**方が綺麗。今回は bridge 内の純粋な重複削減に絞った。

1. **`erogameScape.ts` / `game.ts` の `try/catch` バリデーション** — bridge が入力 trim や空チェックも担っている。**G7 で hooks に移管を検討**。
2. **`processMonitor.ts` の silent error swallowing** — `getMonitoringStatus` / `getProcessSnapshot` がエラー時に空配列を返している。`ApiResult<T>` に統一すべきだが、呼び出し側のコンポーネントも合わせて修正が必要。**G7 で再検討**。
3. **`database.ts updatePlayStatus` の read-update-read** — 読み出し→変換→更新→再読み出しの3往復。バックエンドAPI追加かhooks化で改善可能だが behavior change を伴う。**見送り**。
4. **`loadImage.ts` の `loadImageFromWeb`** — 単純pass-through。削除候補だが呼び出し元の対応を伴う。**G10 で再検討**。
5. **`credential.ts` / `database.ts` の PascalCase payload マッピング** — Go側がPascalCaseを期待するため必須。reviewerも「dedupは難しい」と評価。**現状維持**。
6. **`settings.ts` の 9 update メソッド** — factory化は可能だが明示的な現状の方が読みやすい。**現状維持**。

## G7: frontend hooks/*

対象: `frontend/src/hooks/*` (useScreenshotSettings / useFileSelection ほか)
コミット: （このグループのコミット）

### 適用した

| 観点 | 内容 |
|------|------|
| Simplification | `useScreenshotSettings.ts`: 「setState → updateAPI → 失敗 toast.error / 成功 toast.success」を `applySetting<T>` ジェネリックヘルパーに集約。6つの handler の重複（約60行）を圧縮。**意図的な mount-only sync** の useEffect は1つに統合（コメントで意図を明記） |
| Simplification | `useFileSelection.ts`: `result.data !== undefined && result.data !== undefined` の冗長な重複条件を削除 |

### 見送った（将来の課題）

1. **`useScreenshotSettings.ts` の mount-only useEffect が「バグ」だというレビュアー指摘** — 直前のコメント「初回マウント時にLocalStorage設定をバックエンドへ同期する」で意図的だと判明。**バグではない**ので /simplify の対象外。
2. **進捗リスナー（push/pull）の重複** — `useGameSaveData.ts` と `useUploadAfterSession.ts` で似たパターン。前者は `react-hot-toast` 直接、後者は `toastHandler` 経由でインターフェースが違うため、共通化に書き換え+インターフェース統一が要る。**G8/G9 で再検討候補**。
3. **`useUploadAfterSession.ts` の `if (toastId) ... else ...` パターン3箇所** — `showLoading` が undefined を返しうるかの確認が必要。波及確認のコスト vs 削減効果が見合わない。**現状維持**。
4. **`useCloudData.ts` の `buildCloudDataFromTree` フルツリー再構築** — 1ゲーム詳細展開時にツリー全体を再構築。挙動を変えずに直接更新するには細かい注意が必要。**G9 で cloud 全体と一緒に再検討**。
5. **`useScreenshotSettings.ts` の `isCapturingHotkey` ローカル UI 状態** — コンポーネントへ移すべきという altitude 指摘だが、フック側がイベント captures も管理しているので分離が単純ではない。**現状維持**。
6. **bridge レイヤーの altitude 持ち越し**（erogameScape の validation、processMonitor の silent エラー swallow） — hooks 側で対応するなら呼び出し側 components まで波及するため**G9/G10 と合わせて見直す候補**。

## G8: frontend settings/*

対象: `frontend/src/components/settings/*` (5タブ + GeneralSettings + R2S3Settings)
コミット: （このグループのコミット）

### 適用した

| 観点 | 内容 |
|------|------|
| Reuse | `SettingsToggle` コンポーネントを新設。`ScreenshotSettingsTab` の5箇所のトグル（label + description + checkbox）を集約（約65行削減） |
| Reuse | `TabSectionHeader` コンポーネントを新設（`border-l-4 + 色付き見出し + 説明`）。Tailwind JIT 対応のため色クラスを Record で明示マップ化。5タブ（Appearance/Behavior/Defaults/SyncAndLogs/Screenshot）の冒頭を集約 |

### 見送った（将来の課題）

1. **`GeneralSettings.tsx` ハンドラのフック化（120行削減候補）** — `handleSyncAllGames` / `handleExportGameData` / `handleCreateBackup` / `handleRestoreBackup` / `handleOfflineModeChange` / `handleAutoTrackingChange` を `useBehaviorSettings` / `useSyncAndBackup` 等の hook に分解。挙動と props インターフェースに広く波及するため見送り。
2. **`BehaviorTab.tsx` の独自トグル markup** — `SettingsToggle` と markup（label.cursor-pointer 包み）が異なるため統一すると挙動が変わる可能性。**現状維持**。
3. **タブの atom 購読の絞り込み** — `GeneralSettings.tsx` の atom 購読を子に降ろせばリレンダーが減る、という efficiency 指摘。挙動を変える可能性があり、効果も実測がいる。**見送り**。
4. **`useScreenshotSettings` の mount-only sync を削除提案** — G7と同じ理由で意図的な動作のため /simplify 対象外。

## G9〜G10

（着手時に追記）
