/**
 * @fileoverview ウィンドウ操作ブリッジ。
 */

import { WindowMinimise, WindowToggleMaximise, Quit } from "../../wailsjs/runtime/runtime";
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
  };
}
