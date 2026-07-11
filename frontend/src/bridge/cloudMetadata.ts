/**
 * @fileoverview クラウド側ゲームメタデータ読み込みブリッジ。
 *
 * LoadCloudMetadata の updatedAt を Date に正規化し、games 配列は型だけ寄せる。
 */

import { LoadCloudMetadata } from "../../wailsjs/go/app/App";
import { normalizeApiDate } from "./helpers";
import type { CloudGameMetadata } from "src/types/cloud";
import type { WindowApi } from "./types";

export function createCloudMetadataBridge(): WindowApi["cloudMetadata"] {
  return {
    loadCloudMetadata: async () => {
      const result = await LoadCloudMetadata();
      if (!result.success || !result.data) {
        return { success: false, message: result.error?.message ?? "エラー" };
      }
      return {
        success: true,
        data: {
          version: result.data.version,
          updatedAt: normalizeApiDate(result.data.updatedAt),
          games: result.data.games as unknown as CloudGameMetadata[],
        },
      };
    },
  };
}
