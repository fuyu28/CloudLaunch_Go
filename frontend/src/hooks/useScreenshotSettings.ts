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

export function useScreenshotSettings() {
  const [screenshotSyncEnabled, setScreenshotSyncEnabled] = useAtom(screenshotSyncEnabledAtom);
  const [screenshotUploadJpeg, setScreenshotUploadJpeg] = useAtom(screenshotUploadJpegAtom);
  const [screenshotJpegQuality, setScreenshotJpegQuality] = useAtom(screenshotJpegQualityAtom);
  const [screenshotClientOnly, setScreenshotClientOnly] = useAtom(screenshotClientOnlyAtom);
  const [screenshotLocalJpeg, setScreenshotLocalJpeg] = useAtom(screenshotLocalJpegAtom);
  const [screenshotHotkey, setScreenshotHotkey] = useAtom(screenshotHotkeyAtom);
  const [screenshotHotkeyNotify, setScreenshotHotkeyNotify] = useAtom(screenshotHotkeyNotifyAtom);
  const [isCapturingHotkey, setIsCapturingHotkey] = useState(false);

  const handleScreenshotSyncEnabledChange = async (enabled: boolean): Promise<void> => {
    setScreenshotSyncEnabled(enabled);
    const result = await window.api.settings.updateScreenshotSyncEnabled(enabled);
    if (!result.success) {
      toast.error("スクリーンショット同期の更新に失敗しました");
      return;
    }
    toast.success(`スクリーンショット同期を${enabled ? "有効" : "無効"}にしました`);
  };

  const handleScreenshotUploadJpegChange = async (enabled: boolean): Promise<void> => {
    setScreenshotUploadJpeg(enabled);
    const result = await window.api.settings.updateScreenshotUploadJpeg(enabled);
    if (!result.success) {
      toast.error("スクリーンショット形式の更新に失敗しました");
      return;
    }
    toast.success(`スクリーンショットを${enabled ? "JPEG" : "PNG"}でアップロードします`);
  };

  const handleScreenshotJpegQualityChange = async (value: number): Promise<void> => {
    const nextValue = Math.min(100, Math.max(1, value));
    setScreenshotJpegQuality(nextValue);
    const result = await window.api.settings.updateScreenshotJpegQuality(nextValue);
    if (!result.success) {
      toast.error("スクリーンショット品質の更新に失敗しました");
      return;
    }
  };

  const handleScreenshotClientOnlyChange = async (enabled: boolean): Promise<void> => {
    setScreenshotClientOnly(enabled);
    const result = await window.api.settings.updateScreenshotClientOnly(enabled);
    if (!result.success) {
      toast.error("スクリーンショット設定の更新に失敗しました");
      return;
    }
    toast.success(enabled ? "タイトルバーを除外して撮影します" : "タイトルバーを含めて撮影します");
  };

  const handleScreenshotLocalJpegChange = async (enabled: boolean): Promise<void> => {
    setScreenshotLocalJpeg(enabled);
    const result = await window.api.settings.updateScreenshotLocalJpeg(enabled);
    if (!result.success) {
      toast.error("スクリーンショット設定の更新に失敗しました");
      return;
    }
    toast.success(enabled ? "ローカル保存をJPEGにします" : "ローカル保存をPNGにします");
  };

  const applyScreenshotHotkey = async (value: string, showToast: boolean): Promise<void> => {
    const trimmed = value.trim();
    if (!trimmed) {
      if (showToast) {
        toast.error("ホットキーを入力してください");
      }
      return;
    }
    const result = await window.api.settings.updateScreenshotHotkey(trimmed);
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

  const handleScreenshotHotkeyNotifyChange = async (enabled: boolean): Promise<void> => {
    setScreenshotHotkeyNotify(enabled);
    const result = await window.api.settings.updateScreenshotHotkeyNotify(enabled);
    if (!result.success) {
      toast.error("ホットキー通知の更新に失敗しました");
      return;
    }
    toast.success(enabled ? "ホットキー通知を有効にしました" : "ホットキー通知を無効にしました");
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

  useEffect(() => {
    void window.api.settings.updateScreenshotSyncEnabled(screenshotSyncEnabled);
  }, []);

  useEffect(() => {
    void window.api.settings.updateScreenshotUploadJpeg(screenshotUploadJpeg);
  }, []);

  useEffect(() => {
    void window.api.settings.updateScreenshotJpegQuality(screenshotJpegQuality);
  }, []);

  useEffect(() => {
    void window.api.settings.updateScreenshotClientOnly(screenshotClientOnly);
  }, []);

  useEffect(() => {
    void window.api.settings.updateScreenshotLocalJpeg(screenshotLocalJpeg);
  }, []);

  useEffect(() => {
    // 初回マウント時にLocalStorage設定をバックエンドへ同期する。
    void applyScreenshotHotkey(screenshotHotkey, false);
  }, []);

  useEffect(() => {
    void window.api.settings.updateScreenshotHotkeyNotify(screenshotHotkeyNotify);
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
