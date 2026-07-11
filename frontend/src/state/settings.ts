/**
 * @fileoverview 設定関連のJotai atoms
 *
 * アプリケーションの設定値をグローバルに管理するatoms。
 * LocalStorageとの同期も自動的に行われます。
 *
 * 主な機能：
 * - テーマ設定の管理
 * - デフォルトソート順の管理
 * - デフォルトフィルターの管理
 * - オフラインモードの管理
 * - 起動時の自動計測の管理
 * - LocalStorageとの自動同期
 */

import { atom } from "jotai";
import { atomWithStorage } from "jotai/utils";
import toast from "react-hot-toast";

import { logger } from "@renderer/utils/logger";

import type { ThemeName } from "@renderer/constants/themes";
import type { SortOption, FilterOption } from "src/types/menu";

/**
 * 設定関連のatoms
 */

/**
 * テーマ設定atom
 * LocalStorageに自動保存される
 */
export const themeAtom = atomWithStorage<ThemeName>("theme", "cloudlaunch");

/**
 * デフォルトソート順atom
 * LocalStorageに自動保存される
 */
export const defaultSortOptionAtom = atomWithStorage<SortOption>("defaultSortOption", "title");

/**
 * デフォルトフィルター状態atom
 * LocalStorageに自動保存される
 */
export const defaultFilterStateAtom = atomWithStorage<FilterOption>("defaultFilterState", "all");

/**
 * オフラインモード設定atom
 * LocalStorageに自動保存される
 */
export const offlineModeAtom = atomWithStorage<boolean>("offlineMode", false);

/**
 * 起動時の自動計測設定atom
 * LocalStorageに自動保存される
 */
export const autoTrackingAtom = atomWithStorage<boolean>("autoTracking", true);

/**
 * 同時転送数設定atom（アップロード/ダウンロード共通）
 * LocalStorageに自動保存される
 */
export const transferConcurrencyAtom = atomWithStorage<number>("transferConcurrency", 6);

/**
 * スクリーンショット同期設定atom
 * LocalStorageに自動保存される
 */
export const screenshotSyncEnabledAtom = atomWithStorage<boolean>("screenshotSyncEnabled", false);

/**
 * スクリーンショットのJPEG変換設定atom
 * LocalStorageに自動保存される
 */
export const screenshotUploadJpegAtom = atomWithStorage<boolean>("screenshotUploadJpeg", true);

/**
 * スクリーンショットJPEG品質設定atom
 * LocalStorageに自動保存される
 */
export const screenshotJpegQualityAtom = atomWithStorage<number>("screenshotJpegQuality", 85);

/**
 * スクリーンショットをクライアント領域のみ取得する設定atom
 * LocalStorageに自動保存される
 */
export const screenshotClientOnlyAtom = atomWithStorage<boolean>("screenshotClientOnly", true);

/**
 * スクリーンショットのローカル保存をJPEGにする設定atom
 * LocalStorageに自動保存される
 */
export const screenshotLocalJpegAtom = atomWithStorage<boolean>("screenshotLocalJpeg", false);

/**
 * スクリーンショットのホットキー設定atom
 * LocalStorageに自動保存される
 */
export const screenshotHotkeyAtom = atomWithStorage<string>("screenshotHotkey", "Ctrl+Alt+S");

/**
 * ホットキー通知の有効設定atom
 * LocalStorageに自動保存される
 */
export const screenshotHotkeyNotifyAtom = atomWithStorage<boolean>("screenshotHotkeyNotify", true);

/**
 * S3 path-style アドレス指定（MinIO 等）
 * LocalStorageに自動保存される
 */
export const s3ForcePathStyleAtom = atomWithStorage<boolean>("s3ForcePathStyle", false);

/**
 * S3 TLS 利用設定
 * LocalStorageに自動保存される
 */
export const s3UseTLSAtom = atomWithStorage<boolean>("s3UseTLS", true);

/**
 * テーマ変更中の状態atom
 * 一時的な状態なのでLocalStorageには保存しない
 */
export const isChangingThemeAtom = atom(false);

/**
 * テーマ変更アクションatom
 * テーマを変更し、HTMLのdata-theme属性も更新する
 */
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

/**
 * ソート順の表示名マップ
 */
export const sortOptionLabels: Record<SortOption, string> = {
  title: "タイトル順",
  lastPlayed: "最近プレイした順",
  totalPlayTime: "プレイ時間が長い順",
  publisher: "ブランド順",
  lastRegistered: "最近登録した順",
};

/**
 * フィルター状態の表示名マップ
 */
export const filterStateLabels: Record<FilterOption, string> = {
  all: "すべて",
  unplayed: "未プレイ",
  playing: "プレイ中",
  played: "クリア済み",
};
