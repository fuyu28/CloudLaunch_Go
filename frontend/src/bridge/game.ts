/**
 * @fileoverview ゲーム起動・スクリーンショット取得ブリッジ。
 *
 * captureWindow は Wails 例外を ApiResult 失敗に落とす（他 API は result 型で返る）。
 */

import { LaunchGame, CaptureGameScreenshot } from "../../wailsjs/go/app/App";
import { getErrorMessage, toApiResult, toApiResultVoid } from "./helpers";
import type { WindowApi } from "./types";

export function createGameBridge(): WindowApi["game"] {
  return {
    launchGame: async (exePath) => toApiResultVoid(await LaunchGame(exePath)),
    captureWindow: async (gameId) => {
      try {
        return toApiResult<string>(await CaptureGameScreenshot(gameId));
      } catch (error) {
        return {
          success: false,
          message: getErrorMessage(error, "スクリーンショットに失敗しました"),
        };
      }
    },
  };
}
