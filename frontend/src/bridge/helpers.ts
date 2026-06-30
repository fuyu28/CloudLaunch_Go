/**
 * @fileoverview wailsBridge 共有ヘルパ関数。
 *
 * - 日時正規化: `normalizeApiDate`
 * - クラウドデータ正規化: `normalizeCloudDirectoryNode`, `normalizeCloudDataItem`, `normalizeCloudFileDetail`
 * - ドメインモデルマッパ: `toGameType`, `toPlaySessionType`, `toMemoType`, `toCloudMemoInfo`
 * - 定型変換ヘルパ: `toApiResult`, `toApiResultVoid`
 */

import type { ApiResult } from "src/types/result";
import type { GameType, PlaySessionType } from "src/types/game";
import type { MemoType, CloudMemoInfo } from "src/types/memo";
import type { CloudDataItem, CloudDirectoryNode, CloudFileDetail } from "src/types/cloud";
import type {
  app as modelsApp,
  domain as modelsDomain,
  services as modelsServices,
  time as modelsTime,
} from "../../wailsjs/go/models";

// Re-export model types used across bridge modules
export type { modelsApp, modelsDomain, modelsServices, modelsTime };

// ---------------------------------------------------------------------------
// 日時正規化
// ---------------------------------------------------------------------------

export function normalizeApiDate(
  value: Date | string | number | null | undefined | modelsTime.Time,
): Date {
  if (value instanceof Date) {
    return value;
  }
  if (typeof value === "string" && value.startsWith("0001-01-01T00:00:00")) {
    return new Date(Number.NaN);
  }
  // time.Time arrives as ISO string at runtime; coerce to string for Date constructor
  const coerced = value as Date | string | number | null | undefined;
  return new Date(coerced ?? Number.NaN);
}

// ---------------------------------------------------------------------------
// クラウドデータ正規化
// ---------------------------------------------------------------------------

export function normalizeCloudDirectoryNode(
  node: modelsApp.CloudDirectoryNode,
): CloudDirectoryNode {
  return {
    name: node.name,
    path: node.path,
    isDirectory: node.isDirectory,
    size: node.size,
    lastModified: normalizeApiDate(node.lastModified),
    children: node.children?.map(normalizeCloudDirectoryNode),
    objectKey: node.objectKey,
  };
}

/**
 * 軽量サマリ（CloudGameSummaryItem）を CloudDataItem に正規化する。
 * ファイル数・サイズは commit メタにキャッシュされていれば反映される。
 * 旧 commit にはフィールドが無く 0 が入る（表示側で「未取得」扱い）。
 */
export function normalizeCloudGameSummaryItem(item: modelsApp.CloudGameSummaryItem): CloudDataItem {
  return {
    name: item.name,
    totalSize: item.totalSize ?? 0,
    fileCount: item.fileCount ?? 0,
    lastModified: normalizeApiDate(item.lastModified),
    remotePath: item.remotePath,
  };
}

export function normalizeCloudDataItem(item: modelsApp.CloudDataItem): CloudDataItem {
  return {
    name: item.name,
    totalSize: item.totalSize,
    fileCount: item.fileCount,
    lastModified: normalizeApiDate(item.lastModified),
    remotePath: item.remotePath,
  };
}

export function normalizeCloudFileDetail(file: modelsApp.CloudFileDetail): CloudFileDetail {
  return {
    name: file.name,
    size: file.size,
    lastModified: normalizeApiDate(file.lastModified),
    key: file.key,
    relativePath: file.relativePath,
  };
}

// ---------------------------------------------------------------------------
// ドメインモデルマッパ
// ---------------------------------------------------------------------------

export function toGameType(g: modelsDomain.Game): GameType {
  return {
    id: g.id,
    title: g.title,
    publisher: g.publisher,
    imagePath: g.imagePath,
    exePath: g.exePath,
    saveFolderPath: g.saveFolderPath,
    createdAt: normalizeApiDate(g.createdAt),
    localSaveHash: g.localSaveHash,
    localSaveHashUpdatedAt: g.localSaveHashUpdatedAt
      ? normalizeApiDate(g.localSaveHashUpdatedAt)
      : undefined,
    playStatus: g.playStatus as GameType["playStatus"],
    totalPlayTime: g.totalPlayTime,
    lastPlayed: g.lastPlayed ? normalizeApiDate(g.lastPlayed) : null,
    clearedAt: g.clearedAt ? normalizeApiDate(g.clearedAt) : null,
    currentRouteId: g.currentRouteId ?? null,
  };
}

export function toPlaySessionType(s: modelsDomain.PlaySession): PlaySessionType {
  return {
    id: s.id,
    gameId: s.gameId,
    playedAt: normalizeApiDate(s.playedAt),
    duration: s.duration,
    sessionName: s.sessionName,
  };
}

export function toMemoType(m: modelsDomain.Memo): MemoType {
  return {
    id: m.id,
    title: m.title,
    content: m.content,
    gameId: m.gameId,
    createdAt: normalizeApiDate(m.createdAt),
    updatedAt: normalizeApiDate(m.updatedAt),
  };
}

export function toCloudMemoInfo(c: modelsServices.CloudMemoInfo): CloudMemoInfo {
  return {
    key: c.key,
    fileName: c.fileName,
    gameId: c.gameId,
    memoTitle: c.memoTitle,
    memoId: c.memoId,
    lastModified: normalizeApiDate(c.lastModified),
    size: c.size,
  };
}

// ---------------------------------------------------------------------------
// 定型変換ヘルパ
// ---------------------------------------------------------------------------

/** 各ブリッジ呼び出しが共通で使う既定のフォールバックメッセージ。 */
export const DEFAULT_ERROR_MESSAGE = "エラー";

/**
 * Go API レスポンスを `ApiResult<T>` に変換する定型ヘルパ。
 *
 * - success=true: `{ success: true, data: mapData(result.data) }` を返す。
 *   `mapData` を省略した場合は `result.data` をそのまま使う。
 * - success=false: `{ success: false, message: result.error?.message ?? fallbackMessage }` を返す。
 *
 * **元の戻り値と完全等価**であることが自明なケース(data 変換なし or 単純キャスト)にのみ使う。
 * 複雑なネスト構築や特殊分岐は各ブリッジモジュールで元の形を維持する。
 */
export function toApiResult<T>(
  result: { success: boolean; data?: unknown; error?: { message?: string } },
  fallbackMessage: string = DEFAULT_ERROR_MESSAGE,
  mapData?: (data: unknown) => T,
): ApiResult<T> {
  if (result.success) {
    const data = mapData ? mapData(result.data) : (result.data as T);
    return { success: true, data };
  }
  return { success: false, message: result.error?.message ?? fallbackMessage };
}

/**
 * 配列を返す Go API レスポンスを `ApiResult<T[]>` に変換する。
 * `data ?? []` を `mapItem` で要素ごとに変換する。
 */
export function toApiResultArray<TItem, TOut>(
  result: { success: boolean; data?: TItem[] | null; error?: { message?: string } },
  mapItem: (item: TItem) => TOut,
  fallbackMessage: string = DEFAULT_ERROR_MESSAGE,
): ApiResult<TOut[]> {
  if (result.success) {
    return { success: true, data: (result.data ?? []).map(mapItem) };
  }
  return { success: false, message: result.error?.message ?? fallbackMessage };
}

/**
 * data なし(void)の Go API レスポンスを `ApiResult<void>` に変換する定型ヘルパ。
 */
export function toApiResultVoid(
  result: { success: boolean; error?: { message?: string } },
  fallbackMessage: string = DEFAULT_ERROR_MESSAGE,
): ApiResult<void> {
  return result.success
    ? { success: true }
    : { success: false, message: result.error?.message ?? fallbackMessage };
}

/**
 * 例外オブジェクトから人間向けメッセージを抽出する。
 * - `Error` インスタンス → `.message`
 * - 文字列 → そのまま
 * - その他で truthy → `JSON.stringify`
 * - falsy → `fallback`
 */
export function getErrorMessage(error: unknown, fallback: string = DEFAULT_ERROR_MESSAGE): string {
  if (error instanceof Error) return error.message;
  if (typeof error === "string") return error;
  if (error) {
    try {
      return JSON.stringify(error);
    } catch {
      return fallback;
    }
  }
  return fallback;
}
