/**
 * @fileoverview Window.api 型拡張
 *
 * Wails ブリッジを window.api として公開するためのグローバル宣言。
 */

import type { WindowApi } from "../wailsBridge";

declare global {
  interface Window {
    api: WindowApi;
  }
}

export {};
