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
    updateAutoTracking: async (enabled) => toApiResultVoid(await UpdateAutoTracking(enabled)),
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
  };
}
