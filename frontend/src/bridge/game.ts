/**
 * @fileoverview ゲーム起動・スクリーンショットブリッジ。
 */

import { LaunchGame, CaptureGameScreenshot } from "../../wailsjs/go/app/App";
import { toApiResultVoid } from "./helpers";
import type { WindowApi } from "./types";

export function createGameBridge(): WindowApi["game"] {
  return {
    launchGame: async (exePath) => toApiResultVoid(await LaunchGame(exePath), "エラー"),
    captureWindow: async (gameId) => {
      try {
        const result = await CaptureGameScreenshot(gameId);
        return result.success
          ? { success: true, data: result.data as string }
          : { success: false, message: result.error?.message ?? "エラー" };
      } catch (error) {
        const message = error instanceof Error ? error.message : "スクリーンショットに失敗しました";
        return { success: false, message };
      }
    },
  };
}
