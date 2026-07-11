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
} from "@renderer/state/settings";

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

  // 初回マウント時のみ。atom 更新のたびに呼ぶのは各設定ハンドラの責務。
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    const settings = window.api.settings;
    void settings.updateOfflineMode(offlineMode);
    void settings.updateAutoTracking(autoTracking);
    void settings.updateUploadConcurrency(transferConcurrency);
    void settings.updateScreenshotSyncEnabled(screenshotSyncEnabled);
    void settings.updateScreenshotUploadJpeg(screenshotUploadJpeg);
    void settings.updateScreenshotJpegQuality(screenshotJpegQuality);
    void settings.updateScreenshotClientOnly(screenshotClientOnly);
    void settings.updateScreenshotLocalJpeg(screenshotLocalJpeg);
    void settings.updateScreenshotHotkeyNotify(screenshotHotkeyNotify);
    const trimmedHotkey = screenshotHotkey.trim();
    if (trimmedHotkey) {
      void settings.updateScreenshotHotkey(trimmedHotkey);
    }
  }, []);
}
