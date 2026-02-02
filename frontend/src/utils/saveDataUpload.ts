/**
 * @fileoverview セーブデータアップロードとハッシュ同期の共通処理
 */

import type { ApiResult } from "src/types/result";

import { createRemotePath } from "@renderer/utils";

type UploadSaveDataInput = {
  gameId: string;
  saveFolderPath: string;
  localHash?: string;
};

export async function uploadSaveDataAndSyncHash(
  input: UploadSaveDataInput,
): Promise<ApiResult<void>> {
  const remotePath = createRemotePath(input.gameId);
  const uploadResult = await window.api.saveData.upload.uploadSaveDataFolder(
    input.saveFolderPath,
    remotePath,
  );
  if (!uploadResult.success) {
    return uploadResult;
  }

  const hash =
    input.localHash ??
    (await window.api.saveData.hash.computeLocalHash(input.saveFolderPath)).data ??
    null;
  if (hash) {
    await window.api.saveData.hash.saveCloudHash(input.gameId, hash);
  }

  return uploadResult;
}
