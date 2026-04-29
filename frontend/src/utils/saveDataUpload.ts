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

type DownloadSaveDataInput = {
  gameId: string;
  saveFolderPath: string;
};

async function syncGameMetadata(gameId: string, actionLabel: string): Promise<ApiResult<void>> {
  const syncResult = await window.api.cloudSync.syncGame(gameId);
  if (syncResult.success) {
    return { success: true };
  }

  return {
    success: false,
    message: `${actionLabel}後のセッション同期に失敗しました: ${syncResult.message ?? "エラー"}`,
  };
}

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

  return syncGameMetadata(input.gameId, "セーブデータアップロード");
}

export async function downloadSaveDataAndSyncMetadata(
  input: DownloadSaveDataInput,
): Promise<ApiResult<void>> {
  const remotePath = createRemotePath(input.gameId);
  const downloadResult = await window.api.saveData.download.downloadSaveData(
    input.saveFolderPath,
    remotePath,
  );
  if (!downloadResult.success) {
    return downloadResult;
  }

  return syncGameMetadata(input.gameId, "セーブデータダウンロード");
}
