/**
 * @fileoverview 画像 URL 解決ブリッジ。
 *
 * ローカルは Go 経由で data URL 化。Web はパスをそのまま返す（追加 fetch しない）。
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
    // http(s) / data URL はそのまま <img src> に渡せるので Go を経由しない。
    loadImageFromWeb: async (src) => ({ success: true, data: src }),
  };
}
