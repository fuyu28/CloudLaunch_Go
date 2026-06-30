/**
 * @fileoverview 画像読み込みブリッジ。
 */

import { LoadImageFromLocal } from "../../wailsjs/go/app/App";
import type { WindowApi } from "./types";

export function createLoadImageBridge(): WindowApi["loadImage"] {
  return {
    loadImageFromLocal: async (path) => {
      const result = await LoadImageFromLocal(path);
      return result.success
        ? { success: true, data: result.data as string }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
    loadImageFromWeb: async (src) => ({ success: true, data: src }),
  };
}
