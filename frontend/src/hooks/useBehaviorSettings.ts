/**
 * @fileoverview 動作設定（オフライン / 自動検出 / 同時転送）の更新ハンドラ。
 *
 * atom はバックエンド更新成功後に書き換える（同時転送の入力中ドラフトは除く）。
 */

import { useAtom } from "jotai";
import toast from "react-hot-toast";

import { logger } from "@renderer/utils/logger";
import {
  offlineModeAtom,
  autoTrackingAtom,
  transferConcurrencyAtom,
} from "@renderer/state/settings";

export function useBehaviorSettings() {
  const [offlineMode, setOfflineMode] = useAtom(offlineModeAtom);
  const [autoTracking, setAutoTracking] = useAtom(autoTrackingAtom);
  const [transferConcurrency, setTransferConcurrency] = useAtom(transferConcurrencyAtom);

  const handleOfflineModeChange = async (enabled: boolean): Promise<void> => {
    // バックエンド ContentSyncService にも反映しないと、process_monitor 経由の
    // 自動同期や直接の cloudSync.Push 呼び出しが UI 設定を無視して S3 にアクセスし続ける。
    const result = await window.api.settings.updateOfflineMode(enabled);
    if (!result.success) {
      logger.error("オフラインモード設定の更新エラー:", {
        component: "useBehaviorSettings",
        function: "handleOfflineModeChange",
        data: result.message,
      });
      toast.error("オフラインモードの更新に失敗しました");
      return;
    }
    setOfflineMode(enabled);
    toast.success(
      enabled ? "オフラインモードを有効にしました" : "オフラインモードを無効にしました",
    );
  };

  const handleAutoTrackingChange = async (enabled: boolean): Promise<void> => {
    try {
      const result = await window.api.settings.updateAutoTracking(enabled);
      if (result.success) {
        setAutoTracking(enabled);
        toast.success(
          enabled ? "自動ゲーム検出を有効にしました" : "自動ゲーム検出を無効にしました",
        );
      } else {
        toast.error("設定の更新に失敗しました");
      }
    } catch (error) {
      logger.error("自動ゲーム検出設定の更新エラー:", {
        component: "useBehaviorSettings",
        function: "handleAutoTrackingChange",
        data: error,
      });
      toast.error("設定の更新に失敗しました");
    }
  };

  const applyTransferConcurrency = async (value: number, showToast: boolean): Promise<void> => {
    try {
      const result = await window.api.settings.updateUploadConcurrency(value);
      if (!result.success) {
        if (showToast) {
          toast.error("同時転送数の更新に失敗しました");
        }
      } else if (showToast) {
        toast.success(`同時転送数を ${value} に設定しました`);
      }
    } catch (error) {
      logger.error("同時転送数設定の更新エラー:", {
        component: "useBehaviorSettings",
        function: "applyTransferConcurrency",
        data: error,
      });
      if (showToast) {
        toast.error("同時転送数の更新に失敗しました");
      }
    }
  };

  const handleTransferConcurrencyChange = async (value: number): Promise<void> => {
    const nextValue = Math.min(32, Math.max(1, value));
    setTransferConcurrency(nextValue);
    await applyTransferConcurrency(nextValue, true);
  };

  return {
    offlineMode,
    autoTracking,
    transferConcurrency,
    setTransferConcurrency,
    handleOfflineModeChange,
    handleAutoTrackingChange,
    handleTransferConcurrencyChange,
  };
}
