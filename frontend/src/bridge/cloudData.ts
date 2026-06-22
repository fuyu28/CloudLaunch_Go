/**
 * @fileoverview クラウドデータ操作ブリッジ。
 */

import {
  ListCloudData,
  GetDirectoryTree,
  DeleteCloudData,
  DeleteFile,
  GetCloudFileDetails,
} from "../../wailsjs/go/app/App";
import {
  normalizeCloudDataItem,
  normalizeCloudDirectoryNode,
  normalizeCloudFileDetail,
  toApiResultVoid,
} from "./helpers";
import type { WindowApi } from "./types";

export function createCloudDataBridge(): WindowApi["cloudData"] {
  return {
    listCloudData: async () => {
      const result = await ListCloudData();
      return result.success
        ? {
            success: true,
            data: (result.data ?? []).map(normalizeCloudDataItem),
          }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
    getDirectoryTree: async () => {
      const result = await GetDirectoryTree();
      return result.success
        ? {
            success: true,
            data: (result.data ?? []).map(normalizeCloudDirectoryNode),
          }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
    deleteCloudData: async (path) => toApiResultVoid(await DeleteCloudData(path), "エラー"),
    deleteFile: async (path) => toApiResultVoid(await DeleteFile(path), "エラー"),
    getCloudFileDetails: async (path) => {
      const result = await GetCloudFileDetails(path);
      return result.success
        ? {
            success: true,
            data: (result.data ?? []).map(normalizeCloudFileDetail),
          }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
  };
}
