/**
 * @fileoverview オフラインモード関連のユーティリティ関数
 */

/**
 * オフラインモード時に無効化すべきUI要素のクラスを取得
 *
 * @param isOfflineMode オフラインモードかどうか
 * @returns 無効化用のCSSクラス
 */
export function getOfflineDisabledClasses(isOfflineMode: boolean): string {
  return isOfflineMode ? "opacity-50 cursor-not-allowed pointer-events-none" : "";
}
