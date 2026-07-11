/**
 * @fileoverview ローディング状態管理フック
 *
 * このフックは、非同期処理のローディング状態を管理します。
 */

import { useCallback, useMemo, useState } from "react";

import { useToastHandler, executeWithToast, type ToastOptions } from "./useToastHandler";

type UseLoadingStateReturn = {
  /** ローディング中かどうか（走っている非同期処理が1つ以上あるとき true） */
  isLoading: boolean;
  executeWithLoading: <T>(
    asyncFn: () => Promise<T>,
    options?: ToastOptions,
  ) => Promise<T | undefined>;
};

/**
 * ローディング状態管理フック
 *
 * 非同期処理のローディング状態を管理し、トースト通知と統合された実行ヘルパーを提供します。
 * かつては `setLoading` / `setError` / `reset` などの互換 API を返していたが、
 * どれも呼び出し元がなくなったため撤去し、公開 API を `isLoading` / `executeWithLoading` の2つに絞る。
 *
 * @param initialLoading - 初期ローディング状態
 * @returns ローディング状態管理機能
 */
export function useLoadingState(initialLoading = false): UseLoadingStateReturn {
  // 単一 boolean だと executeWithLoading が並行に呼ばれたときに finally が先に走った側で
  // isLoading=false になってしまい、まだ処理中の他方の表示が消える不具合が出る。
  // 参照カウントに変更し、走っている非同期処理が1つでもあれば isLoading=true を維持する。
  const [runningCount, setRunningCount] = useState<number>(initialLoading ? 1 : 0);
  const isLoading = runningCount > 0;
  const toastHandler = useToastHandler();

  /**
   * トースト付きで非同期処理を実行する。
   * 並行実行を安全に扱うため、+1/-1 の参照カウントで isLoading を管理する。
   *
   * @param asyncFn - 実行する非同期関数
   * @param options - トーストオプション
   * @returns 実行結果またはundefined
   */
  const executeWithLoading = useCallback(
    async <T>(asyncFn: () => Promise<T>, options?: ToastOptions): Promise<T | undefined> => {
      setRunningCount((prev) => prev + 1);
      try {
        return await executeWithToast(asyncFn, options || {}, toastHandler);
      } catch {
        // トーストは executeWithToast 側済み。ここでは undefined を返して二重表示しない。
        return undefined;
      } finally {
        setRunningCount((prev) => Math.max(0, prev - 1));
      }
    },
    [toastHandler],
  );

  return useMemo(
    () => ({
      isLoading,
      executeWithLoading,
    }),
    [isLoading, executeWithLoading],
  );
}
