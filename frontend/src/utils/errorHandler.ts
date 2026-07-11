/**
 * @fileoverview 共通エラーハンドリングユーティリティ
 *
 * このファイルは、アプリケーション全体で使用されるエラーハンドリング機能を提供します。
 */

import toast from "react-hot-toast";

import { logger } from "./logger";
import type { ApiResult } from "src/types/result";

/** ApiResult 失敗時にトースト表示。toastId 指定時は既存トーストを差し替える。 */
export function handleApiError<T = void>(
  result: ApiResult<T>,
  fallbackMessage: string = "エラーが発生しました",
  toastId?: string,
): void {
  let message: string;

  if (result.success) {
    message = fallbackMessage;
  } else {
    message = (result as { success: false; message: string }).message || fallbackMessage;
  }

  if (toastId) {
    toast.error(message, { id: toastId });
  } else {
    toast.error(message);
  }
}

/** 予期しない例外をログ＋トースト。toastId 指定時は既存トーストを差し替える。 */
export function handleUnexpectedError(error: unknown, context: string, toastId?: string): void {
  const isDev = process.env.NODE_ENV === "development" || process.env.NODE_ENV === "test";
  if (isDev) {
    logger.error(`予期しないエラー (${context}):`, {
      component: "errorHandler",
      function: "handleUnexpectedError",
      error: error instanceof Error ? error : new Error(String(error)),
      data: { context },
    });
  }

  const message = "予期しないエラーが発生しました";
  if (toastId) {
    toast.error(message, { id: toastId });
  } else {
    toast.error(message);
  }
}

export function showSuccessToast(message: string, toastId?: string): void {
  if (toastId) {
    toast.success(message, { id: toastId });
  } else {
    toast.success(message);
  }
}

export async function withLoadingToast<T>(
  asyncOperation: () => Promise<ApiResult<T>>,
  loadingMessage: string,
  successMessage: string,
  errorContext: string,
): Promise<ApiResult<T>> {
  const loadingToastId = toast.loading(loadingMessage);

  try {
    const result = await asyncOperation();

    if (result.success) {
      showSuccessToast(successMessage, loadingToastId);
    } else {
      handleApiError(result, "エラーが発生しました", loadingToastId);
    }

    return result;
  } catch (error) {
    handleUnexpectedError(error, errorContext, loadingToastId);
    return { success: false, message: "予期しないエラーが発生しました" };
  }
}
