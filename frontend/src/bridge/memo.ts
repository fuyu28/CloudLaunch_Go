/**
 * @fileoverview メモブリッジ。
 */

import {
  ListAllMemos,
  GetMemoByID,
  ListMemosByGame,
  CreateMemo,
  UpdateMemo,
  DeleteMemo,
  GetMemoRootDir,
  GetMemoFilePath,
  GetGameMemoDir,
  UploadMemoToCloud,
  DownloadMemoFromCloud,
  GetCloudMemos,
  SyncMemosFromCloud,
} from "../../wailsjs/go/app/App";
import { toMemoType, toCloudMemoInfo, toApiResultVoid } from "./helpers";
import type { MemoType } from "src/types/memo";
import type { MemoSyncResult } from "src/types/memo";
import type { WindowApi } from "./types";

export function createMemoBridge(): WindowApi["memo"] {
  return {
    getAllMemos: async () => {
      const result = await ListAllMemos();
      return result.success
        ? { success: true, data: (result.data ?? []).map(toMemoType) }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
    getMemoById: async (memoId) => {
      const result = await GetMemoByID(memoId);
      return result.success
        ? {
            success: true,
            data: result.data ? toMemoType(result.data) : (undefined as unknown as MemoType),
          }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
    getMemosByGameId: async (gameId) => {
      const result = await ListMemosByGame(gameId);
      return result.success
        ? { success: true, data: (result.data ?? []).map(toMemoType) }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
    createMemo: async (data) => {
      const result = await CreateMemo({
        Title: data.title,
        Content: data.content,
        GameID: data.gameId,
      });
      return toApiResultVoid(result, "エラー");
    },
    updateMemo: async (memoId, data) =>
      toApiResultVoid(
        await UpdateMemo(memoId, { Title: data.title, Content: data.content }),
        "エラー",
      ),
    deleteMemo: async (memoId) => toApiResultVoid(await DeleteMemo(memoId), "エラー"),
    getMemoRootDir: async () => {
      const result = await GetMemoRootDir();
      return result.success
        ? { success: true, data: result.data as string }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
    getMemoFilePath: async (memoId) => {
      const result = await GetMemoFilePath(memoId);
      return result.success
        ? { success: true, data: result.data as string }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
    getGameMemoDir: async (gameId) => {
      const result = await GetGameMemoDir(gameId);
      return result.success
        ? { success: true, data: result.data as string }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
    uploadMemoToCloud: async (memoId) => toApiResultVoid(await UploadMemoToCloud(memoId), "エラー"),
    downloadMemoFromCloud: async (gameId, memoFileName) => {
      const result = await DownloadMemoFromCloud(gameId, memoFileName);
      return result.success
        ? { success: true, data: result.data as string }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
    getCloudMemos: async () => {
      const result = await GetCloudMemos();
      return result.success
        ? { success: true, data: (result.data ?? []).map(toCloudMemoInfo) }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
    syncMemosFromCloud: async (gameId) => {
      const result = await SyncMemosFromCloud(gameId ?? "");
      return result.success
        ? { success: true, data: result.data as MemoSyncResult }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
  };
}
