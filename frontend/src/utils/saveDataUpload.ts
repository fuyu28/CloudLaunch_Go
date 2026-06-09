/**
 * @fileoverview セーブデータアップロードとダウンロードの共通処理
 */

import type { ApiResult } from "src/types/result";

type UploadSaveDataInput = {
  gameId: string;
  saveFolderPath?: string;
  localHash?: string;
  localUpdatedAt?: Date | string | null;
};

type DownloadSaveDataInput = {
  gameId: string;
  saveFolderPath?: string;
};

export async function uploadSaveDataAndSyncHash(
  input: UploadSaveDataInput,
): Promise<ApiResult<void>> {
  return window.api.cloudSync.push(input.gameId);
}

export async function downloadSaveDataAndSyncMetadata(
  input: DownloadSaveDataInput,
): Promise<ApiResult<void>> {
  return window.api.cloudSync.pull(input.gameId);
}
