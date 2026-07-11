/**
 * @fileoverview wailsBridge 共有ヘルパ。
 *
 * Go モデル → フロント型の変換と ApiResult 定型化。複雑な分岐は各ブリッジ側に残す。
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

export type { modelsApp, modelsDomain, modelsServices, modelsTime };

export function normalizeApiDate(
  value: Date | string | number | null | undefined | modelsTime.Time,
): Date {
  if (value instanceof Date) {
    return value;
  }
  // Go のゼロ値 time が来ると Epoch 近傍の Date になり「未設定」と区別できない。
  if (typeof value === "string" && value.startsWith("0001-01-01T00:00:00")) {
    return new Date(Number.NaN);
  }
  // 実行時の time.Time は ISO 文字列。Date コンストラクタへ渡せる形に寄せる。
  const coerced = value as Date | string | number | null | undefined;
  return new Date(coerced ?? Number.NaN);
}

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

// 旧 commit には fileCount/totalSize が無く 0 になる。表示側は 0 を「未取得」扱いにする。
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

export const DEFAULT_ERROR_MESSAGE = "エラー";

/**
 * Go の ApiResult をフロントの ApiResult に寄せる。
 * ネスト構築や特殊分岐がある呼び出しでは使わず、ブリッジ側で組み立てる。
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
 * Go が nil data を返しうる API 用。
 * toApiResult の `as T` だと「無い」を有値に見せるので、undefined を明示する。
 */
export function toApiResultOptional<TIn, TOut>(
  result: { success: boolean; data?: TIn | null; error?: { message?: string } },
  mapper: (data: TIn) => TOut,
  fallbackMessage: string = DEFAULT_ERROR_MESSAGE,
): ApiResult<TOut | undefined> {
  if (result.success) {
    return { success: true, data: result.data ? mapper(result.data) : undefined };
  }
  return { success: false, message: result.error?.message ?? fallbackMessage };
}

/** data 欠落を空配列に潰し、呼び出し側の null 分岐を不要にする。 */
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

export function toApiResultVoid(
  result: { success: boolean; error?: { message?: string } },
  fallbackMessage: string = DEFAULT_ERROR_MESSAGE,
): ApiResult<void> {
  return result.success
    ? { success: true }
    : { success: false, message: result.error?.message ?? fallbackMessage };
}

/** 未知の throw 値でも UI に出せる文字列へ落とす（オブジェクトはそのまま出さない）。 */
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
