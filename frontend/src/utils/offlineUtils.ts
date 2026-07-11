/**
 * @fileoverview オフラインモード関連のユーティリティ関数
 *
 */

export function getOfflineDisabledClasses(isOfflineMode: boolean): string {
  return isOfflineMode ? "opacity-50 cursor-not-allowed pointer-events-none" : "";
}
