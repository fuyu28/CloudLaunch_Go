/**
 * @fileoverview ローディング状態管理フック
 *
 * このフックは、非同期処理のローディング状態とエラー状態を管理します。
 * 主な機能：
 * - ローディング状態の管理
 * - エラー状態の管理
 * - トースト通知の統合
 * - 非同期処理の実行ヘルパー
 *
 * 使用例：
 * ```tsx
 * const { isLoading, error, executeWithLoading } = useLoadingState()
 * ```
 */

import { useState, useCallback } from "react";

import { useToastHandler, executeWithToast, type ToastOptions } from "./useToastHandler";

/**
 * ローディング状態の型定義
 */
export type LoadingState = {
  /** ローディング中かどうか */
  isLoading: boolean;
  /** エラーメッセージ */
  error: string | undefined;
};

/**
 * ローディング状態管理フック
 *
 * 非同期処理のローディング状態とエラー状態を管理し、
 * トースト通知と統合された実行ヘルパーを提供します。
 *
 * @param initialLoading - 初期ローディング状態
 * @returns ローディング状態管理機能
 */
export function useLoadingState(initialLoading = false): {
  /** ローディング中かどうか */
  isLoading: boolean;
  /** エラーメッセージ */
  error: string | undefined;
  /** ローディング状態を設定 */
  setLoading: (loading: boolean) => void;
  /** エラー状態を設定 */
  setError: (error: string | undefined) => void;
  /** 状態をリセット */
  reset: () => void;
  /** トースト付きで非同期処理を実行 */
  executeWithLoading: <T>(
    asyncFn: () => Promise<T>,
    options?: ToastOptions,
  ) => Promise<T | undefined>;
} {
  // 単一 boolean だと executeWithLoading が並行に呼ばれたときに finally が先に走った側で
  // isLoading=false になってしまい、まだ処理中の他方の表示が消える不具合が出る。
  // 参照カウントに変更し、走っている非同期処理が1つでもあれば isLoading=true を維持する。
  const [runningCount, setRunningCount] = useState<number>(initialLoading ? 1 : 0);
  const [error, setErrorState] = useState<string | undefined>(undefined);
  const isLoading = runningCount > 0;
  const toastHandler = useToastHandler();

  /**
   * ローディング状態を設定する。
   * 直接 boolean を指定するレガシーAPI互換: true で +1、false で 0 にリセット。
   *
   * @param loading - ローディング中かどうか
   */
  const setLoading = useCallback((loading: boolean) => {
    if (loading) {
      setRunningCount((prev) => prev + 1);
    } else {
      setRunningCount(0);
    }
  }, []);

  /**
   * エラー状態を設定する
   *
   * @param error - エラーメッセージまたはundefined
   */
  const setError = useCallback((nextError: string | undefined) => {
    setErrorState(nextError);
  }, []);

  /**
   * 状態をリセットする
   */
  const reset = useCallback(() => {
    setRunningCount(0);
    setErrorState(undefined);
  }, []);

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
      setErrorState(undefined);
      try {
        const result = await executeWithToast(asyncFn, options || {}, toastHandler);
        return result;
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : String(err);
        setErrorState(errorMsg);
        return undefined;
      } finally {
        setRunningCount((prev) => Math.max(0, prev - 1));
      }
    },
    [toastHandler],
  );

  return {
    isLoading,
    error,
    setLoading,
    setError,
    reset,
    executeWithLoading,
  };
}
