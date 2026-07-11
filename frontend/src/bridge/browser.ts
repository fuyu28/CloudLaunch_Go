/**
 * @fileoverview 外部ブラウザ起動ブリッジ。
 *
 * fragment / 相対リンクは BrowserOpenURL に流さず、webview 既定動作に任せる。
 */

import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";
import type { WindowApi } from "./types";

/** http/https 以外を弾く。mailto: や #fragment を OS ブラウザに渡さないため。 */
function isExternalUrl(url: string): boolean {
  return /^https?:\/\//i.test(url);
}

export function createBrowserBridge(): WindowApi["browser"] {
  return {
    openExternalUrl: (url) => {
      if (!isExternalUrl(url)) {
        // 呼び出し側は preventDefault せず、webview 内の既定ナビゲーションに任せる。
        return;
      }
      BrowserOpenURL(url);
    },
  };
}
