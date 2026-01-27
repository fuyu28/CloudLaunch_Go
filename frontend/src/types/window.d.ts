import type { WindowApi } from "../wailsBridge"

declare global {
  interface Window {
    api: WindowApi
  }
}

export {}
