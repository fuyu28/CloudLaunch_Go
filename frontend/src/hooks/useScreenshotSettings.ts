/**
 * @fileoverview スクリーンショット設定フック
 *
 * atom とバックエンド同期、ホットキー登録の状態管理を担う。
 */

import { useAtom } from "jotai";
import { useEffect, useState } from "react";
import toast from "react-hot-toast";

import {
  screenshotSyncEnabledAtom,
  screenshotUploadJpegAtom,
  screenshotJpegQualityAtom,
  screenshotClientOnlyAtom,
  screenshotLocalJpegAtom,
  screenshotHotkeyAtom,
  screenshotHotkeyNotifyAtom,
} from "../state/settings";
import {
  normalizeHotkeyFailureMessage,
  normalizeHotkeyFromEvent as normalizeHotkeyEvent,
} from "../utils/hotkeyNormalize";

type SettingResult = { success: boolean; message?: string };

/**
 * 「バックエンドに反映 → 成功で atom（＝localStorage）を更新 → 失敗時 toast.error」の定型を集約する。
 * atom を先に更新すると、失敗時に localStorage の値がバックエンドと乖離し、
 * 次回起動時のマウント effect が誤った値を再 push してしまうため、
 * 「成功したら初めて反映する」順序に統一する（失敗時は atom を触らない）。
 * successMessage は文字列、または値から文字列を生成する関数を受け取る（未指定で成功トーストなし）。
 */
async function applySetting<T>(
  setter: (value: T) => void,
  apply: (value: T) => Promise<SettingResult>,
  value: T,
  errorMessage: string,
  successMessage?: string | ((value: T) => string),
): Promise<void> {
  const result = await apply(value);
  if (!result.success) {
    toast.error(errorMessage);
    return;
  }
  setter(value);
  if (successMessage !== undefined) {
    toast.success(typeof successMessage === "function" ? successMessage(value) : successMessage);
  }
}

export function useScreenshotSettings() {
  const [screenshotSyncEnabled, setScreenshotSyncEnabled] = useAtom(screenshotSyncEnabledAtom);
  const [screenshotUploadJpeg, setScreenshotUploadJpeg] = useAtom(screenshotUploadJpegAtom);
  const [screenshotJpegQuality, setScreenshotJpegQuality] = useAtom(screenshotJpegQualityAtom);
  const [screenshotClientOnly, setScreenshotClientOnly] = useAtom(screenshotClientOnlyAtom);
  const [screenshotLocalJpeg, setScreenshotLocalJpeg] = useAtom(screenshotLocalJpegAtom);
  const [screenshotHotkey, setScreenshotHotkey] = useAtom(screenshotHotkeyAtom);
  const [screenshotHotkeyNotify, setScreenshotHotkeyNotify] = useAtom(screenshotHotkeyNotifyAtom);
  const [isCapturingHotkey, setIsCapturingHotkey] = useState(false);

  const settings = window.api.settings;

  const handleScreenshotSyncEnabledChange = (enabled: boolean): Promise<void> =>
    applySetting(
      setScreenshotSyncEnabled,
      settings.updateScreenshotSyncEnabled,
      enabled,
      "スクリーンショット同期の更新に失敗しました",
      (v) => `スクリーンショット同期を${v ? "有効" : "無効"}にしました`,
    );

  const handleScreenshotUploadJpegChange = (enabled: boolean): Promise<void> =>
    applySetting(
      setScreenshotUploadJpeg,
      settings.updateScreenshotUploadJpeg,
      enabled,
      "スクリーンショット形式の更新に失敗しました",
      (v) => `スクリーンショットを${v ? "JPEG" : "PNG"}でアップロードします`,
    );

  const handleScreenshotJpegQualityChange = (value: number): Promise<void> =>
    applySetting(
      setScreenshotJpegQuality,
      settings.updateScreenshotJpegQuality,
      Math.min(100, Math.max(1, value)),
      "スクリーンショット品質の更新に失敗しました",
    );

  const handleScreenshotClientOnlyChange = (enabled: boolean): Promise<void> =>
    applySetting(
      setScreenshotClientOnly,
      settings.updateScreenshotClientOnly,
      enabled,
      "スクリーンショット設定の更新に失敗しました",
      (v) => (v ? "タイトルバーを除外して撮影します" : "タイトルバーを含めて撮影します"),
    );

  const handleScreenshotLocalJpegChange = (enabled: boolean): Promise<void> =>
    applySetting(
      setScreenshotLocalJpeg,
      settings.updateScreenshotLocalJpeg,
      enabled,
      "スクリーンショット設定の更新に失敗しました",
      (v) => (v ? "ローカル保存をJPEGにします" : "ローカル保存をPNGにします"),
    );

  const handleScreenshotHotkeyNotifyChange = (enabled: boolean): Promise<void> =>
    applySetting(
      setScreenshotHotkeyNotify,
      settings.updateScreenshotHotkeyNotify,
      enabled,
      "ホットキー通知の更新に失敗しました",
      (v) => (v ? "ホットキー通知を有効にしました" : "ホットキー通知を無効にしました"),
    );

  // 成否を boolean で返し、呼び出し側が失敗時にローカル draft をロールバックできるようにする。
  const applyScreenshotHotkey = async (value: string, showToast: boolean): Promise<boolean> => {
    const trimmed = value.trim();
    if (!trimmed) {
      if (showToast) {
        toast.error("ホットキーを入力してください");
      }
      return false;
    }
    const result = await settings.updateScreenshotHotkey(trimmed);
    if (!result.success) {
      if (showToast) {
        toast.error(result.message || "ホットキーの更新に失敗しました");
      }
      return false;
    }
    // 成功したときのみ atom（localStorage）を更新して、バックエンドとの乖離を防ぐ
    setScreenshotHotkey(trimmed);
    if (showToast) {
      toast.success(`ホットキーを「${trimmed}」に更新しました`);
    }
    return true;
  };

  const handleScreenshotHotkeyChange = async (value: string): Promise<boolean> => {
    return applyScreenshotHotkey(value, true);
  };

  // 起動時のバックエンド同期は MainLayout の useSettingsBootSync が担う。
  // ここではユーザー操作時の反映のみ扱う。

  useEffect(() => {
    if (!isCapturingHotkey) {
      return;
    }
    const shownHints = new Set<string>();
    const handler = (event: KeyboardEvent): void => {
      event.preventDefault();
      event.stopPropagation();
      const result = normalizeHotkeyEvent(event);
      if (!result.ok) {
        if (result.reason === "cancel") {
          setIsCapturingHotkey(false);
          return;
        }
        const message = normalizeHotkeyFailureMessage(result.reason);
        if (message && !shownHints.has(result.reason)) {
          shownHints.add(result.reason);
          toast.error(message);
        }
        return;
      }
      setIsCapturingHotkey(false);
      // applyScreenshotHotkey 成功時にのみ atom を更新する（乖離防止）
      void applyScreenshotHotkey(result.combo, true);
    };
    window.addEventListener("keydown", handler, true);
    return () => {
      window.removeEventListener("keydown", handler, true);
    };
  }, [isCapturingHotkey]);

  return {
    screenshotSyncEnabled,
    screenshotUploadJpeg,
    screenshotJpegQuality,
    screenshotClientOnly,
    screenshotLocalJpeg,
    screenshotHotkey,
    setScreenshotHotkey,
    screenshotHotkeyNotify,
    isCapturingHotkey,
    setIsCapturingHotkey,
    handleScreenshotSyncEnabledChange,
    handleScreenshotUploadJpegChange,
    handleScreenshotJpegQualityChange,
    handleScreenshotClientOnlyChange,
    handleScreenshotLocalJpegChange,
    applyScreenshotHotkey,
    handleScreenshotHotkeyChange,
    handleScreenshotHotkeyNotifyChange,
  };
}
