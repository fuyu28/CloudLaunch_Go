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

type SettingResult = { success: boolean; message?: string };

/**
 * 「状態を更新 → バックエンドに反映 → 失敗時 toast.error / 成功時 toast.success」の定型を集約する。
 * successMessage は文字列、または値から文字列を生成する関数を受け取る（未指定で成功トーストなし）。
 */
async function applySetting<T>(
  setter: (value: T) => void,
  apply: (value: T) => Promise<SettingResult>,
  value: T,
  errorMessage: string,
  successMessage?: string | ((value: T) => string),
): Promise<void> {
  setter(value);
  const result = await apply(value);
  if (!result.success) {
    toast.error(errorMessage);
    return;
  }
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

  const applyScreenshotHotkey = async (value: string, showToast: boolean): Promise<void> => {
    const trimmed = value.trim();
    if (!trimmed) {
      if (showToast) {
        toast.error("ホットキーを入力してください");
      }
      return;
    }
    const result = await settings.updateScreenshotHotkey(trimmed);
    if (!result.success) {
      if (showToast) {
        toast.error(result.message || "ホットキーの更新に失敗しました");
      }
      return;
    }
    if (showToast) {
      toast.success(`ホットキーを「${trimmed}」に更新しました`);
    }
  };

  const handleScreenshotHotkeyChange = async (value: string): Promise<void> => {
    setScreenshotHotkey(value);
    await applyScreenshotHotkey(value, true);
  };

  const normalizeHotkeyFromEvent = (event: KeyboardEvent): string | null => {
    if (event.key === "Escape") {
      setIsCapturingHotkey(false);
      return null;
    }
    const modifiers: string[] = [];
    if (event.ctrlKey) modifiers.push("Ctrl");
    if (event.altKey) modifiers.push("Alt");
    if (event.shiftKey) modifiers.push("Shift");
    if (event.metaKey) modifiers.push("Win");

    const key = event.key;
    if (key === "Control" || key === "Alt" || key === "Shift" || key === "Meta") {
      return null;
    }
    let mainKey = "";
    if (/^F(1[0-2]|[1-9])$/.test(key)) {
      mainKey = key.toUpperCase();
    } else if (key.length === 1) {
      mainKey = key.toUpperCase();
    } else {
      return null;
    }

    if (modifiers.length === 0) {
      return null;
    }
    return [...modifiers, mainKey].join("+");
  };

  // 初回マウント時にLocalStorage設定をバックエンドへ同期する。
  // 個々の handle* と違って各設定値が変わるたびではなく、起動時の1回だけ送る意図のため deps は [] のまま。
  useEffect(() => {
    void settings.updateScreenshotSyncEnabled(screenshotSyncEnabled);
    void settings.updateScreenshotUploadJpeg(screenshotUploadJpeg);
    void settings.updateScreenshotJpegQuality(screenshotJpegQuality);
    void settings.updateScreenshotClientOnly(screenshotClientOnly);
    void settings.updateScreenshotLocalJpeg(screenshotLocalJpeg);
    void settings.updateScreenshotHotkeyNotify(screenshotHotkeyNotify);
    void applyScreenshotHotkey(screenshotHotkey, false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!isCapturingHotkey) {
      return;
    }
    const handler = (event: KeyboardEvent): void => {
      event.preventDefault();
      const hotkey = normalizeHotkeyFromEvent(event);
      if (!hotkey) {
        return;
      }
      setIsCapturingHotkey(false);
      setScreenshotHotkey(hotkey);
      void applyScreenshotHotkey(hotkey, true);
    };
    window.addEventListener("keydown", handler);
    return () => {
      window.removeEventListener("keydown", handler);
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
    normalizeHotkeyFromEvent,
  };
}
