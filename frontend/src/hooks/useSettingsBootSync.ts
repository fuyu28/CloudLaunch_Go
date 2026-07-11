/**
 * @fileoverview 起動時に localStorage 永続設定をバックエンドへ再同期する。
 *
 * バックエンドの Config / サービス状態はプロセス起動ごとに既定値へ戻るため、
 * FE の atomWithStorage 値を MainLayout マウント時に一度だけ push する。
 */

import { useAtomValue } from "jotai";
import { useEffect } from "react";

import {
  offlineModeAtom,
  autoTrackingAtom,
  transferConcurrencyAtom,
  screenshotSyncEnabledAtom,
  screenshotUploadJpegAtom,
  screenshotJpegQualityAtom,
  screenshotClientOnlyAtom,
  screenshotLocalJpegAtom,
  screenshotHotkeyAtom,
  screenshotHotkeyNotifyAtom,
  s3ForcePathStyleAtom,
  s3UseTLSAtom,
} from "@renderer/state/settings";
import { logLevelManager } from "@renderer/utils/logLevel";

// StrictMode の effect 二重実行やレイアウト再マウントで、ホットキー再登録が
// 多重に走らないようにプロセス寿命で一度だけ同期する。
let bootSyncCompleted = false;

/**
 * アプリ常駐レイアウトから呼び、Settings タブを開かなくてもスクショ設定等が効くようにする。
 * 個々の設定変更は各タブのハンドラが担うため、ここでは初回マウント時のみ同期する。
 */
export function useSettingsBootSync(): void {
  const offlineMode = useAtomValue(offlineModeAtom);
  const autoTracking = useAtomValue(autoTrackingAtom);
  const transferConcurrency = useAtomValue(transferConcurrencyAtom);
  const screenshotSyncEnabled = useAtomValue(screenshotSyncEnabledAtom);
  const screenshotUploadJpeg = useAtomValue(screenshotUploadJpegAtom);
  const screenshotJpegQuality = useAtomValue(screenshotJpegQualityAtom);
  const screenshotClientOnly = useAtomValue(screenshotClientOnlyAtom);
  const screenshotLocalJpeg = useAtomValue(screenshotLocalJpegAtom);
  const screenshotHotkey = useAtomValue(screenshotHotkeyAtom);
  const screenshotHotkeyNotify = useAtomValue(screenshotHotkeyNotifyAtom);
  const s3ForcePathStyle = useAtomValue(s3ForcePathStyleAtom);
  const s3UseTLS = useAtomValue(s3UseTLSAtom);

  // boot sync は初回のみ。以降の atom→backend は各設定ハンドラの責務。
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (bootSyncCompleted) {
      return;
    }
    bootSyncCompleted = true;

    const settings = window.api.settings;
    void (async () => {
      // ホットキー系は並列だと stop/start が競合して already registered になるため直列化する。
      // コンボ未変更時は backend 側で no-op になる。
      await settings.updateScreenshotHotkeyNotify(screenshotHotkeyNotify);
      const trimmedHotkey = screenshotHotkey.trim();
      if (trimmedHotkey) {
        await settings.updateScreenshotHotkey(trimmedHotkey);
      }

      void settings.updateOfflineMode(offlineMode);
      void settings.updateAutoTracking(autoTracking);
      void settings.updateUploadConcurrency(transferConcurrency);
      void settings.updateScreenshotSyncEnabled(screenshotSyncEnabled);
      void settings.updateScreenshotUploadJpeg(screenshotUploadJpeg);
      void settings.updateScreenshotJpegQuality(screenshotJpegQuality);
      void settings.updateScreenshotClientOnly(screenshotClientOnly);
      void settings.updateScreenshotLocalJpeg(screenshotLocalJpeg);
      void settings.updateS3ForcePathStyle(s3ForcePathStyle);
      void settings.updateS3UseTLS(s3UseTLS);
      const feLevel = logLevelManager.getCurrentLevel();
      if (feLevel !== "off") {
        void settings.updateLogLevel(feLevel);
      }
    })();
  }, []);
}

/** テスト用: boot sync の一度きりガードをリセットする。 */
export function resetSettingsBootSyncForTests(): void {
  bootSyncCompleted = false;
}
