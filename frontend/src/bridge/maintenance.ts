/**
 * @fileoverview メンテナンス(エクスポート・バックアップ)ブリッジ。
 */

import { ExportGameData, CreateFullBackup, RestoreFullBackup } from "../../wailsjs/go/app/App";
import { toApiResult, toApiResultVoid } from "./helpers";
import type { WindowApi } from "./types";

export function createMaintenanceBridge(): WindowApi["maintenance"] {
  return {
    exportGameData: async (outputDir) =>
      toApiResult(
        await ExportGameData(outputDir),
        "エラー",
        (d) => d as { jsonPath: string; csvPath: string },
      ),
    createFullBackup: async (outputDir) =>
      toApiResult(await CreateFullBackup(outputDir), "エラー", (d) => d as string),
    restoreFullBackup: async (backupPath) => toApiResultVoid(await RestoreFullBackup(backupPath)),
  };
}
