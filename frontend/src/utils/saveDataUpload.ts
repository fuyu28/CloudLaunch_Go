/**
 * @fileoverview セーブデータアップロードとダウンロードの共通処理
 */

import type { ApiResult } from "src/types/result";

type UploadSaveDataInput = {
  gameId: string;
};

type PullSaveDataInput = {
  gameId: string;
};

export async function uploadSaveDataAndSyncHash(
  input: UploadSaveDataInput,
): Promise<ApiResult<void>> {
  return window.api.cloudSync.push(input.gameId);
}

export async function downloadSaveDataAndSyncMetadata(
  input: PullSaveDataInput,
): Promise<ApiResult<void>> {
  return window.api.cloudSync.pull(input.gameId);
}
