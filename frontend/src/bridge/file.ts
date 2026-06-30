/**
 * @fileoverview ファイル操作ブリッジ。
 */

import {
  SelectFile,
  SelectFolder,
  CheckFileExists,
  CheckDirectoryExists,
  OpenLogsDirectory,
} from "../../wailsjs/go/app/App";
import type { WindowApi } from "./types";

export function createFileBridge(): WindowApi["file"] {
  return {
    selectFile: async (filters) => {
      const result = await SelectFile(filters ?? []);
      if (!result.success) {
        return {
          success: false,
          message: result.error?.message ?? "ファイルが選択されませんでした",
        };
      }
      return { success: true, data: result.data as string };
    },
    selectFolder: async () => {
      const result = await SelectFolder();
      if (!result.success) {
        return {
          success: false,
          message: result.error?.message ?? "フォルダが選択されませんでした",
        };
      }
      return { success: true, data: result.data as string };
    },
    checkFileExists: async (filePath) => {
      const result = await CheckFileExists(filePath);
      return result.success ? Boolean(result.data) : false;
    },
    checkDirectoryExists: async (dirPath) => {
      const result = await CheckDirectoryExists(dirPath);
      return result.success ? Boolean(result.data) : false;
    },
    openLogsDirectory: async () => {
      const result = await OpenLogsDirectory();
      return result.success
        ? { success: true, data: result.data as string }
        : {
            success: false,
            message: result.error?.message ?? "ログフォルダの表示に失敗しました",
          };
    },
  };
}
