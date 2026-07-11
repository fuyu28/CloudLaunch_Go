/**
 * @fileoverview セーブデータブリッジ。
 *
 */

import { GetCloudFileDetailsByGame } from "../../wailsjs/go/app/App";
import { normalizeCloudFileDetail } from "./helpers";
import type { WindowApi } from "./types";

export function createSaveDataBridge(): WindowApi["saveData"] {
  return {
    download: {
      getCloudFileDetails: async (gameId) => {
        const result = await GetCloudFileDetailsByGame(gameId);
        return result.success
          ? {
              success: true,
              data: {
                exists: Boolean(result.data?.exists),
                totalSize: Number(result.data?.totalSize ?? 0),
                files: (result.data?.files ?? []).map(normalizeCloudFileDetail),
              },
            }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
    },
  };
}
