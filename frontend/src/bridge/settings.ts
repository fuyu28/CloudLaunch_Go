/**
 * @fileoverview 設定ブリッジ。
 */

import {
  UpdateAutoTracking,
  UpdateUploadConcurrency,
  UpdateScreenshotSyncEnabled,
  UpdateScreenshotUploadJpeg,
  UpdateScreenshotJpegQuality,
  UpdateScreenshotClientOnly,
  UpdateScreenshotLocalJpeg,
  UpdateScreenshotHotkey,
  UpdateScreenshotHotkeyNotify,
} from "../../wailsjs/go/app/App";
import { toApiResultVoid } from "./helpers";
import type { WindowApi } from "./types";

export function createSettingsBridge(): WindowApi["settings"] {
  return {
    updateAutoTracking: async (enabled) =>
      toApiResultVoid(await UpdateAutoTracking(enabled), "エラー"),
    updateUploadConcurrency: async (value) =>
      toApiResultVoid(await UpdateUploadConcurrency(value), "エラー"),
    updateScreenshotSyncEnabled: async (enabled) =>
      toApiResultVoid(await UpdateScreenshotSyncEnabled(enabled), "エラー"),
    updateScreenshotUploadJpeg: async (enabled) =>
      toApiResultVoid(await UpdateScreenshotUploadJpeg(enabled), "エラー"),
    updateScreenshotJpegQuality: async (value) =>
      toApiResultVoid(await UpdateScreenshotJpegQuality(value), "エラー"),
    updateScreenshotClientOnly: async (enabled) =>
      toApiResultVoid(await UpdateScreenshotClientOnly(enabled), "エラー"),
    updateScreenshotLocalJpeg: async (enabled) =>
      toApiResultVoid(await UpdateScreenshotLocalJpeg(enabled), "エラー"),
    updateScreenshotHotkey: async (combo) =>
      toApiResultVoid(await UpdateScreenshotHotkey(combo), "エラー"),
    updateScreenshotHotkeyNotify: async (enabled) =>
      toApiResultVoid(await UpdateScreenshotHotkeyNotify(enabled), "エラー"),
  };
}
