# フロントエンド リファクタ調査報告・計画

## 0. 前提・スコープ

- 本計画は `feature/content-addressing-sync`（バックエンドの content-addressing 同期リファクタ完了済み）から**新しいブランチを切って**フロントエンドを整理することを前提とする。
- バックエンド（Go）↔ Wails 自動生成バインディング（`frontend/wailsjs/`）↔ `wailsBridge.ts` の**API連携自体は健全**であることを確認済み（呼び出しシグネチャ・入力ペイロード・リネーム追従すべて一致、`window.api` への不正呼び出しエラーは 0 件）。
- したがって本リファクタは**振る舞いを変えず**、型安全性・テスト・コード構成の健全化に集中する。
- 推奨ブランチ名: `refactor/frontend-cleanup`

---

## 1. 現状調査サマリ

| 指標 | 現状 | 備考 |
|---|---|---|
| `bun run lint` (oxlint) | PASS (0 件) | 問題なし |
| `bun run build` (Vite) | PASS | 問題なし |
| `oxfmt --check` 相当 | PASS | 問題なし |
| `bunx tsc --noEmit` | **約 487 件のエラー** | うち約 459 件がテストファイル、約 28 件が本体コード |
| `bun run test` (vitest) | **5 suites 失敗** | jest 形式 4 + 新規テストの assertion 不一致 1 |

### tsc エラーの内訳（エラーコード別）

| コード | 件数 | 意味 | 主因 |
|---|---|---|---|
| TS2304 | 270 | Cannot find name | テストの `describe/it/expect/vi` 未定義 |
| TS2593 | 100 | テストランナー型未導入 | 同上 |
| TS2708 | 75 | namespace を値として使用 | テストの `jest.*` 参照 |
| TS7006 | 16 | 暗黙の any | 本体フック/コンポーネント |
| TS2352/2345/2353/2322 | 14 | 型キャスト/代入不整合 | Wails 境界（Time vs Date 等） |
| TS2688/2307/2694 | 8 | 型定義ファイル/モジュール解決失敗 | Electron `preload` 残骸 |

### tsc エラーのファイル別分布

```
211  src/constants/__tests__/constants.test.ts      ← jest形式
 92  src/hooks/__tests__/useToastHandler.test.tsx    ← jest形式
 84  src/components/__tests__/GameModal.test.tsx     ← jest形式
 72  src/hooks/__tests__/useGameActions.test.tsx     ← jest形式
 16  src/wailsBridge.ts                              ← Time vs Date キャスト
  3  src/hooks/useMonitoringStatus.ts                ← 暗黙any
  3  src/components/MemoForm.tsx                      ← 暗黙any
  2  src/utils/__tests__/saveDataUpload.test.ts      ← assertion不一致
  2  src/components/CloudDataCard.tsx                 ← 暗黙any
  1  src/pages/Cloud.tsx / useFileSelection.ts / useCloudData.ts / PlaySessionManagementModal.tsx
```

---

## 2. 課題カテゴリ別の詳細

### A. テスト基盤の jest→vitest 移行が未完了【最優先】

- 新規テスト `src/utils/__tests__/saveDataUpload.test.ts` は **vitest 形式**（`import { describe, it, expect, vi } from "vitest"`）で書かれている。
- 一方、既存の 4 ファイルは **jest 形式のまま**残っている:
  - `src/constants/__tests__/constants.test.ts`（`/// <reference types="jest" />`）
  - `src/hooks/__tests__/useToastHandler.test.tsx`
  - `src/components/__tests__/GameModal.test.tsx`
  - `src/hooks/__tests__/useGameActions.test.tsx`
- `vite.config.ts` に **vitest の `test` 設定が存在しない**（`globals`/`environment`/`setupFiles` 未設定）。`@vitejs/plugin-react` のみ。
- 結果として、グローバルの `describe/it/expect/jest` が型・実行とも解決できず、tsc エラーの約 94% / テスト失敗の大半がここに集中している。
- テスト用の `tsconfig` 分離もなく、`tsconfig.json` の `types` は `["vite/client"]` のみ。

### B. Wails 境界の型安全性（Time vs Date / `as` キャスト / 暗黙 any）

- 生成モデル（`wailsjs/go/models.ts`）の日時は `time.Time`（実行時は ISO 文字列）。一方フロントの型（`GameType.createdAt: Date` 等）は `Date`。
- `wailsBridge.ts` は一部（cloud / sync メタ）を `normalizeApiDate` で `Date` に正規化しているが、`Game`/`PlaySession`/`Memo` は **`as GameType` 等のキャストのみ**で正規化していない。→ 実行時は文字列、型上は `Date` という乖離（TS2352 等 16 件の温床）。
- `window.api` の戻り値を扱うフック/コンポーネントで暗黙 any（TS7006）が発生:
  - `src/hooks/useMonitoringStatus.ts`（`game` パラメータ）
  - `src/components/MemoForm.tsx`（`gameResult`/`memoResult`）
  - `src/components/CloudDataCard.tsx`（ソート比較の `a`,`b`）
  - `src/hooks/useCloudData.ts`（`result`）
- 影響: 型による回帰検知が効かない箇所が残っている。

### C. Electron 時代の残骸

- Wails 移行後にもかかわらず Electron の `preload` への参照が残存（`preload/` ディレクトリは既に存在しない）:
  - `src/hooks/useGameActions.ts:20` `/// <reference types="../../../preload/index.d.ts" />`
  - `src/hooks/useFileSelection.ts:11` 同上 → TS2688
  - `src/hooks/__tests__/useGameActions.test.tsx:16` `import type { API } from ".../preload/preload.d"`
  - `src/components/__tests__/types.d.ts:1` `import type { API } from ".../preload/preload"`
- `wailsBridge.ts` ヘッダコメントが「Electron IPC互換のWailsブリッジ」と Electron 前提の表現のまま。
- `window.api` のグローバル型は既に `src/types/window.d.ts`（`WindowApi`）が正しく供給しているため、上記参照は**全て不要・削除可能**。

### D. 型定義の重複と置き場所の乱れ

- `CloudDataItem` / `CloudFileDetail` / `CloudDirectoryNode` が **3 か所**で重複定義:
  - `src/types/cloud.d.ts`（本来の正典）
  - `src/hooks/useCloudData.ts`（再定義 + 再エクスポート）
  - `src/utils/cloudUtils.ts`（`CloudDirectoryNode` を再定義）
- さらに `wailsBridge.ts` がこれらの型を **`./hooks/useCloudData` から import** しており、低レベルのブリッジが高レベルのフックに依存する**逆方向依存**になっている。
- 単一の出所（`src/types/`）へ集約すべき。

### E. クラウド同期ロジックの分散・重複

- `cloudSync.status/push/pull/resolveConflict` の呼び出しと「`conflict`/`push_needed`/`pull_needed` 分岐 + 未追跡削除確認フロー」が複数箇所に散在:
  - `src/pages/GameDetail.tsx`
  - `src/pages/Cloud.tsx`
  - `src/pages/Home.tsx`
  - `src/components/GeneralSettings.tsx`
  - `src/hooks/useUploadAfterSession.ts`
  - `src/components/CloudGameImportModal.tsx`
- 同期ステータス解釈・進捗購読（`onProgress`）・競合解決・未追跡削除確認は**共通フック（例: `useCloudSync`）へ抽出**することで重複を解消し、新規モーダル（`SyncConflictModal`/`UntrackedDeleteModal`）との結線も一元化できる。

### F. `wailsBridge.ts` の肥大化（972 行・単一ファイル）

- 全ドメイン（window/settings/maintenance/file/database/memo/credential/cloudData/saveData/cloudSync/game/erogameScape/errorReport）が 1 ファイルに集約され、`success ? ... : { success:false, message }` の定型変換が大量に重複。
- ドメイン別ファイル分割（`bridge/database.ts`, `bridge/cloudSync.ts` …）＋ 共通の `toApiResult()` ヘルパ抽出で可読性・保守性を改善できる。`window.api` の公開形（`WindowApi`）は不変のまま内部分割する。

### G. ディレクトリ構成（任意・低優先）

- `src/components/` がフラットに 40 ファイル。ドメイン別サブディレクトリ（`components/game/`, `components/cloud/`, `components/memo/`）へのグルーピングは可読性向上に寄与するが、import パスの一括更新を伴うため最終フェーズで検討。

---

## 3. リファクタ計画（フェーズ分割）

各フェーズは独立して PR 化でき、`bun run lint` / `bunx tsc --noEmit`（対象範囲）/ `bun run test` で都度検証する。**振る舞いは変えない**。

### Phase 1: テスト基盤の vitest 統一【最優先・他フェーズの前提】
- `vite.config.ts` に vitest の `test` 設定を追加（`globals: true`, `environment: "jsdom"`, `setupFiles`（jest-dom 読込））。`/// <reference types="vitest/config" />`。
- テスト用 tsconfig（`tsconfig.test.json` など）または `tsconfig.json` の `types` に `vitest/globals` を追加。
- 既存 4 ファイルを jest→vitest へ移行:
  - `/// <reference types="jest" />` 等を削除
  - `jest.fn()`→`vi.fn()`、`jest.Mock`→`Mock`(from vitest)、`jest.mock`→`vi.mock`
  - 必要に応じ `import { describe, it, expect, vi, beforeEach } from "vitest"`
- 新規 `saveDataUpload.test.ts` の assertion 不一致を修正（`pull` の第 2 引数 `false` を期待値に反映）。
- Electron `preload` を参照しているテスト（`useGameActions.test.tsx`, `components/__tests__/types.d.ts`）の型供給を `WindowApi`（`src/types/window.d.ts`）ベースへ置換。
- **受け入れ条件**: テストファイル起因の tsc エラー 0、`bun run test` 全 green。

### Phase 2: Electron 残骸の除去
- `src/hooks/useGameActions.ts` / `src/hooks/useFileSelection.ts` の `/// <reference types=".../preload/index.d.ts" />` 行を削除。
- `wailsBridge.ts` ヘッダコメントを Wails 前提の説明に更新。
- 不要になった Electron 関連 import / 型を一掃。
- **受け入れ条件**: TS2688/2307（preload 関連）0 件。

### Phase 3: 型定義の単一化
- `CloudDataItem`/`CloudFileDetail`/`CloudDirectoryNode` を `src/types/cloud.d.ts` に一本化。
- `src/hooks/useCloudData.ts` / `src/utils/cloudUtils.ts` の重複定義を削除し、`src/types` から import する形へ。
- `wailsBridge.ts` の型 import 元を `./hooks/useCloudData` → `src/types/cloud` に変更（逆方向依存の解消）。
- **受け入れ条件**: 重複型定義 0、ブリッジがフックに型依存しない。

### Phase 4: Wails 境界の型安全化
- 日時正規化を一貫化: `Game`/`PlaySession`/`Memo` の戻り値も `normalizeApiDate` 等で `Date` に揃える、もしくは「ブリッジ境界の型は文字列 ISO」と明示しアプリ側で変換する方針に統一。
- 各 `as` キャストを正規の変換関数（`toGameType()` 等のマッパ）に置換し、TS2352/2345 を解消。
- 暗黙 any（TS7006）の解消: `useMonitoringStatus.ts` / `MemoForm.tsx` / `CloudDataCard.tsx` / `useCloudData.ts` に明示的な型注釈を付与。
- `GameDetail.tsx:338` / `Home.tsx:240` の `localSaveHashUpdatedAt`（content-addressing 移行後の意味）を確認し、必要なら新 sync メタへ寄せる。
- **受け入れ条件**: 本体コードの tsc エラー 0。

### Phase 5: クラウド同期ロジックの共通フック化
- `useCloudSync`（仮）を新設し、status 解釈・push/pull・resolveConflict・未追跡削除確認・`onProgress` 購読を集約。
- `GameDetail`/`Cloud`/`Home`/`GeneralSettings`/`useUploadAfterSession`/`CloudGameImportModal` を共通フックに寄せて重複を削減。
- **受け入れ条件**: 同期分岐ロジックの重複が 1 箇所に集約され、各画面はフック呼び出しのみ。

### Phase 6: `wailsBridge.ts` のドメイン分割（任意）
- `bridge/` 配下にドメイン別モジュールを切り出し、`createWailsBridge()` で合成。
- `toApiResult()` 等の定型変換ヘルパを共通化。
- `WindowApi` 公開形は不変。
- **受け入れ条件**: 1 ファイル 972 行→ドメイン別に分割、外部インターフェース不変。

### Phase 7: ディレクトリ構成の整理（任意・最終）
- `components/` のドメイン別サブディレクトリ化。import パス一括更新。

---

## 4. 推奨実施順と依存関係

```
Phase 1 (テスト基盤) ──→ 以降全フェーズの回帰検知の土台
        │
        ├─ Phase 2 (Electron残骸) … 独立・低リスク
        ├─ Phase 3 (型単一化) ──→ Phase 4 の前提
        │         │
        │         └─ Phase 4 (境界型安全化)
        │
        └─ Phase 5 (同期フック化) … Phase 3/4 後が望ましい
                  │
                  └─ Phase 6 (bridge分割) / Phase 7 (構成) … 任意・最後
```

- まず **Phase 1** を完了させ、リファクタ全体の安全網（green なテスト）を確保する。
- Phase 2 / 3 は低リスクで効果が大きく、早期に着手可能。
- Phase 6 / 7 は任意。時間が無ければ見送り可。

---

## 5. リスクと方針

- **振る舞い不変が大原則**。型・テスト・構成の変更にとどめ、ロジックの意味は変えない。
- 各フェーズ完了時に必ず `./scripts/run-all-lint-format.sh` を実行（リポジトリ規約）。
- 日時正規化（Phase 4）は実行時挙動に影響しうるため、表示箇所（一覧の最終更新、起動前確認メッセージ等）を実機（`wails dev`）で確認する。
- 既存テストの移行（Phase 1）では、テストの**意図を変えずに**ランナー差異のみを吸収する。

---

## 6. 完了の定義（Definition of Done）

- `bunx tsc --noEmit`: エラー 0 件
- `bun run test`: 全 suite green
- `bun run lint` / `oxfmt`: クリーン
- `wails dev` で主要フロー（ゲーム起動・セッション記録・同期 push/pull/競合解決・メモ・クラウド一覧）が従来どおり動作
- Electron 残骸・重複型定義・逆方向依存が解消
