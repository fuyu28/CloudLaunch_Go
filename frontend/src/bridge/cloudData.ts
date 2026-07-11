/**
 * @fileoverview クラウド上のセーブ一覧・ディレクトリ操作ブリッジ。
 *
 * Go モデルを normalize してから返す。一覧と詳細ツリーは別 API で遅延取得する。
 */

import {
  ListCloudData,
  ListCloudGameSummaries,
  GetDirectoryTree,
  GetGameDirectoryNode,
  DeleteCloudData,
  DeleteFile,
  GetCloudFileDetails,
} from "../../wailsjs/go/app/App";
import {
  normalizeCloudDataItem,
  normalizeCloudGameSummaryItem,
  normalizeCloudDirectoryNode,
  normalizeCloudFileDetail,
  toApiResultArray,
  toApiResultVoid,
} from "./helpers";
import type { WindowApi } from "./types";

export function createCloudDataBridge(): WindowApi["cloudData"] {
  return {
    listCloudData: async () => toApiResultArray(await ListCloudData(), normalizeCloudDataItem),
    getCloudGameSummaries: async () =>
      toApiResultArray(await ListCloudGameSummaries(), normalizeCloudGameSummaryItem),
    getDirectoryTree: async () =>
      toApiResultArray(await GetDirectoryTree(), normalizeCloudDirectoryNode),
    getGameDirectoryNode: async (gameId) => {
      // toApiResult だと data 欠落時に undefined を CloudDirectoryNode 扱いになるため手組みする。
      const result = await GetGameDirectoryNode(gameId);
      return result.success && result.data
        ? { success: true, data: normalizeCloudDirectoryNode(result.data) }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
    deleteCloudData: async (path) => toApiResultVoid(await DeleteCloudData(path)),
    deleteFile: async (path) => toApiResultVoid(await DeleteFile(path)),
    getCloudFileDetails: async (path) =>
      toApiResultArray(await GetCloudFileDetails(path), normalizeCloudFileDetail),
  };
}
