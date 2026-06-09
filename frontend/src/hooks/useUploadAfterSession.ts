/**
 * @fileoverview セッション終了後のアップロード処理フック
 *
 * セーブデータ差分の検知とアップロード実行を管理します。
 */

import { useCallback, useState } from "react";

import { logger } from "@renderer/utils/logger";

import type { ToastHandler } from "@renderer/hooks/useToastHandler";

type PendingUpload = {
  gameId: string;
  gameTitle: string;
  saveFolderPath: string;
};

type UseUploadAfterSessionResult = {
  pendingUpload: PendingUpload | null;
  checkUploadPrompt: (gameId: string) => Promise<void>;
  handleUploadAfterEnd: () => Promise<void>;
  handleSkipUploadAfterEnd: () => void;
};

export function useUploadAfterSession(
  isOfflineMode: boolean,
  isValidCreds: boolean,
  toastHandler: Pick<ToastHandler, "showToast" | "showLoading" | "showSuccess" | "showError">,
): UseUploadAfterSessionResult {
  const [pendingUpload, setPendingUpload] = useState<PendingUpload | null>(null);
  const [uploadingAfterEndGameId, setUploadingAfterEndGameId] = useState<string | null>(null);

  const checkUploadPrompt = useCallback(
    async (gameId: string): Promise<void> => {
      if (isOfflineMode || !isValidCreds) {
        return;
      }
      if (uploadingAfterEndGameId === gameId) {
        return;
      }
      const game = await window.api.database.getGameById(gameId);
      if (!game || !game.saveFolderPath) {
        return;
      }
      try {
        const statusResult = await window.api.cloudSync.status(gameId);
        if (!statusResult.success || !statusResult.data) {
          return;
        }
        const { status } = statusResult.data;
        if (status === "push_needed" || status === "conflict") {
          setPendingUpload({
            gameId,
            gameTitle: game.title,
            saveFolderPath: game.saveFolderPath,
          });
        }
      } catch {
        // バックグラウンドチェックのエラーは無視する
      }
    },
    [isOfflineMode, isValidCreds, uploadingAfterEndGameId],
  );

  const handleUploadAfterEnd = useCallback(async (): Promise<void> => {
    if (!pendingUpload) return;
    const payload = pendingUpload;
    setPendingUpload(null);
    setUploadingAfterEndGameId(payload.gameId);
    const toastId = toastHandler.showLoading("セーブデータをアップロード中…");
    try {
      const result = await window.api.cloudSync.push(payload.gameId);
      if (result.success) {
        if (toastId) {
          toastHandler.showSuccess("セーブデータをクラウドにアップロードしました", toastId);
        } else {
          toastHandler.showToast("セーブデータをクラウドにアップロードしました", "success");
        }
      } else {
        const message = result.message || "セーブデータのアップロードに失敗しました";
        if (toastId) {
          toastHandler.showError(message, toastId);
        } else {
          toastHandler.showToast(message, "error");
        }
      }
    } catch (error) {
      logger.error("セーブデータのアップロードに失敗しました:", {
        component: "useUploadAfterSession",
        function: "handleUploadAfterEnd",
        data: error,
      });
      if (toastId) {
        toastHandler.showError("セーブデータのアップロードに失敗しました", toastId);
      } else {
        toastHandler.showToast("セーブデータのアップロードに失敗しました", "error");
      }
    } finally {
      setUploadingAfterEndGameId(null);
    }
  }, [pendingUpload, toastHandler]);

  const handleSkipUploadAfterEnd = useCallback((): void => {
    setPendingUpload(null);
  }, []);

  return {
    pendingUpload,
    checkUploadPrompt,
    handleUploadAfterEnd,
    handleSkipUploadAfterEnd,
  };
}
