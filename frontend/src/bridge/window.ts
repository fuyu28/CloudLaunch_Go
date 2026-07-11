/**
 * @fileoverview ウィンドウ操作・プラットフォーム取得ブリッジ。
 *
 * Wails runtime の minimise / maximise / quit とフォルダ表示をまとめる。
 */

import {
  WindowMinimise,
  WindowToggleMaximise,
  Quit,
  Environment,
} from "../../wailsjs/runtime/runtime";
import { OpenFolder } from "../../wailsjs/go/app/App";
import type { WindowApi } from "./types";

export function createWindowBridge(): WindowApi["window"] {
  return {
    minimize: async () => {
      await WindowMinimise();
    },
    toggleMaximize: async () => {
      await WindowToggleMaximise();
    },
    close: async () => {
      await Quit();
    },
    openFolder: async (path) => {
      await OpenFolder(path);
    },
    getPlatform: async () => {
      const env = await Environment();
      return env.platform;
    },
  };
}
