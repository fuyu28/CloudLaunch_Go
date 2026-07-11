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
          // 同期状態の取得自体が失敗した場合は、pendingUpload を未設定のまま
          // 「同期済み」と誤認しないよう、警告と通知を出して手動確認を促す。
          const failureMessage = !statusResult.success
            ? (statusResult.message ?? "unknown error")
            : "no data";
          logger.warn("同期状態の取得に失敗しました:", {
            component: "useUploadAfterSession",
            function: "checkUploadPrompt",
            data: failureMessage,
          });
          toastHandler.showToast(
            "同期状態の確認に失敗しました。手動同期をご確認ください。",
            "error",
          );
          return;
        }
        const { status, savesDiffer } = statusResult.data;
        // status=push_needed / conflict でも、fingerprint 差分の実体が sessions.json
        // または game.json のメタデータだけ（=セーブファイル内容は不変）のときは
        // ユーザーに確認するアップロード対象がない。savesDiffer でさらに狭窄する。
        if ((status === "push_needed" || status === "conflict") && savesDiffer) {
          setPendingUpload({
            gameId,
            gameTitle: game.title,
            saveFolderPath: game.saveFolderPath,
          });
        }
      } catch (error) {
        // ネットワーク・権限エラーなどをサイレントに握りつぶすと
        // 差分ありでもプロンプトが出ないまま「同期済み」と誤認するので、
        // 警告ログ＋トーストで手動同期の確認を促す。
        logger.warn("同期状態の確認に失敗しました:", {
          component: "useUploadAfterSession",
          function: "checkUploadPrompt",
          data: error,
        });
        toastHandler.showToast("同期状態の確認に失敗しました。手動同期をご確認ください。", "error");
      }
    },
    [isOfflineMode, isValidCreds, uploadingAfterEndGameId, getStatus, toastHandler],
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
