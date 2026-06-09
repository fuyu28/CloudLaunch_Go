/**
 * @fileoverview ゲームセーブデータ操作フック
 *
 * このフックは、ゲームのセーブデータのアップロード・ダウンロード機能を提供します。
 *
 * 主な機能：
 * - セーブデータのクラウドアップロード
 * - セーブデータのクラウドダウンロード
 * - ローディング状態の管理
 * - エラーハンドリング
 * - バリデーション
 *
 * 使用例：
 * ```tsx
 * const {
 *   uploadSaveData,
 *   downloadSaveData,
 *   isUploading,
 *   isDownloading
 * } = useGameSaveData()
 * ```
 */

import { useState, useCallback } from "react";

import toast from "react-hot-toast";

import { handleApiError, handleUnexpectedError } from "@renderer/utils/errorHandler";

import type { SyncProgressEvent } from "src/wailsBridge";
import type { GameType } from "src/types/game";
import {
  downloadSaveDataAndSyncMetadata,
  uploadSaveDataAndSyncHash,
} from "@renderer/utils/saveDataUpload";

/**
 * ゲームセーブデータ操作フックの戻り値
 */
export type GameSaveDataResult = {
  /** セーブデータアップロード関数 */
  uploadSaveData: (game: GameType) => Promise<void>;
  /** セーブデータダウンロード関数 */
  downloadSaveData: (game: GameType) => Promise<void>;
  /** アップロード中かどうか */
  isUploading: boolean;
  /** ダウンロード中かどうか */
  isDownloading: boolean;
};

/**
 * ゲームセーブデータ操作フック
 *
 * ゲームのセーブデータのアップロード・ダウンロード機能を提供します。
 *
 * @returns セーブデータ操作機能とローディング状態
 */
export function useGameSaveData(): GameSaveDataResult {
  const [isUploading, setIsUploading] = useState(false);
  const [isDownloading, setIsDownloading] = useState(false);

  const uploadSaveData = useCallback(async (game: GameType): Promise<void> => {
    if (!game.saveFolderPath) {
      handleApiError({
        success: false,
        message: "セーブデータフォルダが設定されていません。",
      });
      return;
    }

    setIsUploading(true);
    const toastId = toast.loading("セーブデータをアップロード中…");
    const unsubscribe = window.api.cloudSync.onProgress((event: SyncProgressEvent) => {
      if (event.operation === "push" && event.total > 0) {
        toast.loading(`セーブデータをアップロード中… ${event.current}/${event.total}`, {
          id: toastId,
        });
      }
    });

    try {
      const result = await uploadSaveDataAndSyncHash({ gameId: game.id });
      if (result.success) {
        toast.success("セーブデータのアップロードに成功しました。", { id: toastId });
      } else {
        toast.error(result.message || "エラーが発生しました", { id: toastId });
      }
    } catch (error) {
      handleUnexpectedError(error, "セーブデータのアップロード", toastId);
    } finally {
      unsubscribe();
      setIsUploading(false);
    }
  }, []);

  const downloadSaveData = useCallback(async (game: GameType): Promise<void> => {
    if (!game.saveFolderPath) {
      handleApiError({
        success: false,
        message: "セーブデータフォルダが設定されていません。",
      });
      return;
    }

    setIsDownloading(true);
    const toastId = toast.loading("セーブデータをダウンロード中…");
    const unsubscribe = window.api.cloudSync.onProgress((event: SyncProgressEvent) => {
      if (event.operation === "pull" && event.total > 0) {
        toast.loading(`セーブデータをダウンロード中… ${event.current}/${event.total}`, {
          id: toastId,
        });
      }
    });

    try {
      const result = await downloadSaveDataAndSyncMetadata({ gameId: game.id });
      if (result.success) {
        toast.success("セーブデータのダウンロードに成功しました。", { id: toastId });
      } else {
        toast.error(result.message || "エラーが発生しました", { id: toastId });
      }
    } catch (error) {
      handleUnexpectedError(error, "セーブデータのダウンロード", toastId);
    } finally {
      unsubscribe();
      setIsDownloading(false);
    }
  }, []);

  return {
    uploadSaveData,
    downloadSaveData,
    isUploading,
    isDownloading,
  };
}

export default useGameSaveData;
