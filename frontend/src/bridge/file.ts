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
import { toApiResult } from "./helpers";
import type { WindowApi } from "./types";

export function createFileBridge(): WindowApi["file"] {
  return {
    selectFile: async (filters) =>
      toApiResult<string>(await SelectFile(filters ?? []), "ファイルが選択されませんでした"),
    selectFolder: async () =>
      toApiResult<string>(await SelectFolder(), "フォルダが選択されませんでした"),
    checkFileExists: async (filePath) => {
      const result = await CheckFileExists(filePath);
      return result.success ? Boolean(result.data) : false;
    },
    checkDirectoryExists: async (dirPath) => {
      const result = await CheckDirectoryExists(dirPath);
      return result.success ? Boolean(result.data) : false;
    },
    openLogsDirectory: async () =>
      toApiResult<string>(await OpenLogsDirectory(), "ログフォルダの表示に失敗しました"),
  };
}
