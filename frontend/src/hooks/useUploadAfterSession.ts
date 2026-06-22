/**
 * @fileoverview セッション終了後のアップロード処理フック
 *
 * セーブデータ差分の検知とアップロード実行を管理します。
 */

import { useCallback, useState } from "react";

import { logger } from "@renderer/utils/logger";

import { useCloudSync } from "@renderer/hooks/useCloudSync";
import type { SyncProgressEvent } from "src/wailsBridge";
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
  const { getStatus, push, subscribeProgress } = useCloudSync(isOfflineMode);

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
        const statusResult = await getStatus(gameId);
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
    [isOfflineMode, isValidCreds, uploadingAfterEndGameId, getStatus],
  );

  const handleUploadAfterEnd = useCallback(async (): Promise<void> => {
    if (!pendingUpload) return;
    const payload = pendingUpload;
    setPendingUpload(null);
    setUploadingAfterEndGameId(payload.gameId);
    const toastId = toastHandler.showLoading("セーブデータをアップロード中…");
    const unsubscribe = subscribeProgress((event: SyncProgressEvent) => {
      if (event.operation === "push" && event.total > 0) {
        toastHandler.showLoading(
          `セーブデータをアップロード中… ${event.current}/${event.total}`,
          toastId,
        );
      }
    });
    try {
      const op = await push(payload.gameId);
      if (op.ok) {
        if (toastId) {
          toastHandler.showSuccess("セーブデータをクラウドにアップロードしました", toastId);
        } else {
          toastHandler.showToast("セーブデータをクラウドにアップロードしました", "success");
        }
      } else {
        const message = op.message || "セーブデータのアップロードに失敗しました";
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
      unsubscribe();
      setUploadingAfterEndGameId(null);
    }
  }, [pendingUpload, toastHandler, push, subscribeProgress]);

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
