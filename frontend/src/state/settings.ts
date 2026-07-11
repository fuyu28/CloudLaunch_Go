/**
 * @fileoverview 設定関連のJotai atoms
 *
 * アプリケーションの設定値をグローバルに管理するatoms。
 * LocalStorageとの同期も自動的に行われます。
 */

import { atom } from "jotai";
import { atomWithStorage } from "jotai/utils";
import toast from "react-hot-toast";

import { logger } from "@renderer/utils/logger";

import type { ThemeName } from "@renderer/constants/themes";
import type { SortOption, FilterOption } from "src/types/menu";

export const themeAtom = atomWithStorage<ThemeName>("theme", "cloudlaunch");

export const defaultSortOptionAtom = atomWithStorage<SortOption>("defaultSortOption", "title");

export const defaultFilterStateAtom = atomWithStorage<FilterOption>("defaultFilterState", "all");

export const offlineModeAtom = atomWithStorage<boolean>("offlineMode", false);

export const autoTrackingAtom = atomWithStorage<boolean>("autoTracking", true);

export const transferConcurrencyAtom = atomWithStorage<number>("transferConcurrency", 6);

export const screenshotSyncEnabledAtom = atomWithStorage<boolean>("screenshotSyncEnabled", false);

export const screenshotUploadJpegAtom = atomWithStorage<boolean>("screenshotUploadJpeg", true);

export const screenshotJpegQualityAtom = atomWithStorage<number>("screenshotJpegQuality", 85);

export const screenshotClientOnlyAtom = atomWithStorage<boolean>("screenshotClientOnly", true);

export const screenshotLocalJpegAtom = atomWithStorage<boolean>("screenshotLocalJpeg", false);

export const screenshotHotkeyAtom = atomWithStorage<string>("screenshotHotkey", "Ctrl+Alt+S");

export const screenshotHotkeyNotifyAtom = atomWithStorage<boolean>("screenshotHotkeyNotify", true);

export const s3ForcePathStyleAtom = atomWithStorage<boolean>("s3ForcePathStyle", false);

export const s3UseTLSAtom = atomWithStorage<boolean>("s3UseTLS", true);

// 一時的な状態なので LocalStorage には保存しない（atomWithStorage を使わない）
export const isChangingThemeAtom = atom(false);

export const changeThemeAtom = atom(null, async (_, set, newTheme: ThemeName) => {
  set(isChangingThemeAtom, true);
  try {
    // 先に atom (=LocalStorage) を更新する。atomWithStorage の setItem が例外を投げた場合、
    // DOM を巻き戻すロールバックが煩雑になるので、成功してから DOM を反映する。
    set(themeAtom, newTheme);

    document.documentElement.setAttribute("data-theme", newTheme);

    toast.success(`テーマを「${newTheme}」に変更しました`);

    return { success: true };
  } catch (error) {
    logger.error("テーマの変更に失敗:", {
      component: "settings",
      function: "unknown",
      data: error,
    });
    toast.error("テーマの変更に失敗しました");
    return { success: false, error };
  } finally {
    set(isChangingThemeAtom, false);
  }
});

export const sortOptionLabels: Record<SortOption, string> = {
  title: "タイトル順",
  lastPlayed: "最近プレイした順",
  totalPlayTime: "プレイ時間が長い順",
  publisher: "ブランド順",
  lastRegistered: "最近登録した順",
};

export const filterStateLabels: Record<FilterOption, string> = {
  all: "すべて",
  unplayed: "未プレイ",
  playing: "プレイ中",
  played: "クリア済み",
};
