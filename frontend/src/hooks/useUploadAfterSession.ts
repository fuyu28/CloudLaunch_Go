/**
 * @fileoverview セッション終了後のアップロード処理フック
 *
 * セーブデータ差分の検知とアップロード実行を管理します。
 */

import { useCallback, useState } from "react";

import { logger } from "@renderer/utils/logger";
import { createRemotePath } from "@renderer/utils";

import type { ToastHandler } from "@renderer/hooks/useToastHandler";

type PendingUpload = {
  gameId: string;
  gameTitle: string;
  saveFolderPath: string;
  localHash: string;
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
      const localHashResult = await window.api.saveData.hash.computeLocalHash(game.saveFolderPath);
      if (!localHashResult.success || !localHashResult.data) {
        return;
      }
      const cloudHashResult = await window.api.saveData.hash.getCloudHash(gameId);
      const cloudHash = cloudHashResult.success ? cloudHashResult.data?.hash : null;
      if (!cloudHash || cloudHash !== localHashResult.data) {
        setPendingUpload({
          gameId,
          gameTitle: game.title,
          saveFolderPath: game.saveFolderPath,
          localHash: localHashResult.data,
        });
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
      const remotePath = createRemotePath(payload.gameId);
      const result = await window.api.saveData.upload.uploadSaveDataFolder(
        payload.saveFolderPath,
        remotePath,
      );
      if (result.success) {
        await window.api.saveData.hash.saveCloudHash(payload.gameId, payload.localHash);
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
