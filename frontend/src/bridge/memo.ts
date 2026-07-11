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
import {
  toMemoType,
  toCloudMemoInfo,
  toApiResult,
  toApiResultArray,
  toApiResultVoid,
} from "./helpers";
import type { MemoType } from "src/types/memo";
import type { MemoSyncResult } from "src/types/memo";
import type { WindowApi } from "./types";

export function createMemoBridge(): WindowApi["memo"] {
  return {
    getAllMemos: async () => toApiResultArray(await ListAllMemos(), toMemoType),
    getMemoById: async (memoId) =>
      toApiResult<MemoType>(await GetMemoByID(memoId), undefined, (data) =>
        data
          ? toMemoType(data as Parameters<typeof toMemoType>[0])
          : (undefined as unknown as MemoType),
      ),
    getMemosByGameId: async (gameId) => toApiResultArray(await ListMemosByGame(gameId), toMemoType),
    createMemo: async (data) =>
      toApiResult<MemoType>(
        await CreateMemo({ Title: data.title, Content: data.content, GameID: data.gameId }),
        undefined,
        (raw) =>
          raw
            ? toMemoType(raw as Parameters<typeof toMemoType>[0])
            : (undefined as unknown as MemoType),
      ),
    updateMemo: async (memoId, data) =>
      toApiResultVoid(await UpdateMemo(memoId, { Title: data.title, Content: data.content })),
    deleteMemo: async (memoId) => toApiResultVoid(await DeleteMemo(memoId)),
    getMemoRootDir: async () => toApiResult<string>(await GetMemoRootDir()),
    getMemoFilePath: async (memoId) => toApiResult<string>(await GetMemoFilePath(memoId)),
    getGameMemoDir: async (gameId) => toApiResult<string>(await GetGameMemoDir(gameId)),
    uploadMemoToCloud: async (memoId) => toApiResultVoid(await UploadMemoToCloud(memoId)),
    downloadMemoFromCloud: async (gameId, memoFileName) =>
      toApiResult<string>(await DownloadMemoFromCloud(gameId, memoFileName)),
    getCloudMemos: async () => toApiResultArray(await GetCloudMemos(), toCloudMemoInfo),
    syncMemosFromCloud: async (gameId) =>
      toApiResult<MemoSyncResult>(await SyncMemosFromCloud(gameId ?? "")),
  };
}
