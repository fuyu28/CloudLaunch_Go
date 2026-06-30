import type { WindowApi } from "src/wailsBridge";

declare global {
  interface Window {
    api: WindowApi;
  }
}
