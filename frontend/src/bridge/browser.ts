/**
 * @fileoverview ブラウザ操作ブリッジ。
 *
 * Wails ランタイム API のうち、外部ブラウザで URL を開く機能を提供する。
 * フロントエンドから `wailsjs/runtime` を直接 import せず、必ずこのブリッジ経由で呼ぶ。
 */

import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";
import type { WindowApi } from "./types";

/**
 * URL が http:// または https:// で始まるかを判定する。
 * fragment（"#..."）や相対リンク・mailto: 等をブラウザで開いてしまわないためのガード。
 */
function isExternalUrl(url: string): boolean {
  return /^https?:\/\//i.test(url);
}

export function createBrowserBridge(): WindowApi["browser"] {
  return {
    openExternalUrl: (url) => {
      if (!isExternalUrl(url)) {
        // Wails webview 内で fragment/相対リンクを BrowserOpenURL に流すと不正な遷移になるため、
        // http/https 以外は無視する。呼び出し側は e.preventDefault() をしないことで既定動作に任せる。
        return;
      }
      BrowserOpenURL(url);
    },
  };
}
