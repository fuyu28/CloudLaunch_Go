# Simplify Pass Notes

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

## G2〜G10

（着手時に追記）
