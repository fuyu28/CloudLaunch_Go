/** @fileoverview オフラインモード時に UI 要素を無効化するための CSS クラスを組み立てる。 */

export function getOfflineDisabledClasses(isOfflineMode: boolean): string {
  return isOfflineMode ? "opacity-50 cursor-not-allowed pointer-events-none" : "";
}
