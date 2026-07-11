/** @fileoverview セーブデータアップロードとダウンロードの共通処理。 */

import type { ApiResult } from "src/types/result";
import type { PullResult } from "src/wailsBridge";

type UploadSaveDataInput = {
  gameId: string;
};

type PullSaveDataInput = {
  gameId: string;
  /** 同期管理外のローカル固有ファイルの削除を承認するか（既定 false=確認を返す） */
  deleteUntracked?: boolean;
};

export async function uploadSaveDataAndSyncHash(
  input: UploadSaveDataInput,
): Promise<ApiResult<void>> {
  return window.api.cloudSync.push(input.gameId);
}

export async function downloadSaveDataAndSyncMetadata(
  input: PullSaveDataInput,
): Promise<ApiResult<PullResult>> {
  return window.api.cloudSync.pull(input.gameId, input.deleteUntracked ?? false);
}
