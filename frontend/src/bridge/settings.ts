/**
 * @fileoverview アプリ設定更新ブリッジ。
 *
 * 各 Update* API を toApiResultVoid で薄いラップするだけ。
 */

import {
  UpdateAutoTracking,
  UpdateOfflineMode,
  UpdateUploadConcurrency,
  UpdateScreenshotSyncEnabled,
  UpdateScreenshotUploadJpeg,
  UpdateScreenshotJpegQuality,
  UpdateScreenshotClientOnly,
  UpdateScreenshotLocalJpeg,
  UpdateScreenshotHotkey,
  UpdateScreenshotHotkeyNotify,
  UpdateS3ForcePathStyle,
  UpdateS3UseTLS,
  UpdateLogLevel,
} from "../../wailsjs/go/app/App";
import { toApiResultVoid } from "./helpers";
import type { WindowApi } from "./types";

export function createSettingsBridge(): WindowApi["settings"] {
  return {
    updateAutoTracking: async (enabled) => toApiResultVoid(await UpdateAutoTracking(enabled)),
    updateOfflineMode: async (enabled) => toApiResultVoid(await UpdateOfflineMode(enabled)),
    updateUploadConcurrency: async (value) => toApiResultVoid(await UpdateUploadConcurrency(value)),
    updateScreenshotSyncEnabled: async (enabled) =>
      toApiResultVoid(await UpdateScreenshotSyncEnabled(enabled)),
    updateScreenshotUploadJpeg: async (enabled) =>
      toApiResultVoid(await UpdateScreenshotUploadJpeg(enabled)),
    updateScreenshotJpegQuality: async (value) =>
      toApiResultVoid(await UpdateScreenshotJpegQuality(value)),
    updateScreenshotClientOnly: async (enabled) =>
      toApiResultVoid(await UpdateScreenshotClientOnly(enabled)),
    updateScreenshotLocalJpeg: async (enabled) =>
      toApiResultVoid(await UpdateScreenshotLocalJpeg(enabled)),
    updateScreenshotHotkey: async (combo) => toApiResultVoid(await UpdateScreenshotHotkey(combo)),
    updateScreenshotHotkeyNotify: async (enabled) =>
      toApiResultVoid(await UpdateScreenshotHotkeyNotify(enabled)),
    updateS3ForcePathStyle: async (enabled) =>
      toApiResultVoid(await UpdateS3ForcePathStyle(enabled)),
    updateS3UseTLS: async (enabled) => toApiResultVoid(await UpdateS3UseTLS(enabled)),
    updateLogLevel: async (level) => toApiResultVoid(await UpdateLogLevel(level)),
  };
}
