/**
 * @fileoverview トーストハンドリングフック
 *
 * このファイルは、非同期操作でのトースト通知を管理するフックを提供します。
 */

import { useCallback, useMemo } from "react";
import toast from "react-hot-toast";

export type ToastOptions = {
  loadingMessage?: string;
  successMessage?: string;
  errorMessage?: string;
  showToast?: boolean;
};

export type ToastHandler = {
  showLoading: (message?: string, toastId?: string) => string | undefined;
  showSuccess: (message: string, toastId?: string) => void;
  showError: (message: string, toastId?: string) => void;
  dismiss: (toastId: string) => void;
  showToast: (message: string, type: "success" | "error" | "loading") => void;
};

export function useToastHandler(): ToastHandler {
  const showLoading = useCallback((message?: string, toastId?: string): string | undefined => {
    if (message) {
      return toast.loading(message, toastId ? { id: toastId } : undefined);
    }
    return undefined;
  }, []);

  const showSuccess = useCallback((message: string, toastId?: string): void => {
    if (toastId) {
      toast.success(message, { id: toastId });
    } else {
      toast.success(message);
    }
  }, []);

  const showError = useCallback((message: string, toastId?: string): void => {
    if (toastId) {
      toast.error(message, { id: toastId });
    } else {
      toast.error(message);
    }
  }, []);

  const dismiss = useCallback((toastId: string): void => {
    toast.dismiss(toastId);
  }, []);

  const showToast = useCallback((message: string, type: "success" | "error" | "loading"): void => {
    switch (type) {
      case "success":
        toast.success(message);
        break;
      case "error":
        toast.error(message);
        break;
      case "loading":
        toast.loading(message);
        break;
    }
  }, []);

  // 参照が毎レンダー変わると executeWithLoading が再生成され無限ループの温床になる。
  return useMemo(
    () => ({
      showLoading,
      showSuccess,
      showError,
      dismiss,
      showToast,
    }),
    [showLoading, showSuccess, showError, dismiss, showToast],
  );
}

export async function executeWithToast<T>(
  asyncFn: () => Promise<T>,
  options: ToastOptions,
  toastHandler: ToastHandler,
): Promise<T | undefined> {
  const { loadingMessage, successMessage, errorMessage, showToast = true } = options;

  let toastId: string | undefined;

  try {
    if (showToast && loadingMessage) {
      toastId = toastHandler.showLoading(loadingMessage);
    }

    const result = await asyncFn();

    if (showToast) {
      if (successMessage) {
        toastHandler.showSuccess(successMessage, toastId);
      } else if (toastId) {
        toastHandler.dismiss(toastId);
      }
    }

    return result;
  } catch (error) {
    const errorMsg = error instanceof Error ? error.message : String(error);

    if (showToast) {
      const displayMessage = errorMessage || errorMsg;
      toastHandler.showError(displayMessage, toastId);
    }

    // エラーを再スローして呼び出し元でハンドリングできるようにする
    throw error;
  }
}
