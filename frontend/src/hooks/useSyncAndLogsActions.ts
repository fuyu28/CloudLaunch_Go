/**
 * @fileoverview データ・ログタブの同期 / バックアップ / エクスポート操作。
 */

import { useAtomValue } from "jotai";
import { useState } from "react";
import toast from "react-hot-toast";

import { useCloudSync } from "@renderer/hooks/useCloudSync";
import { offlineModeAtom } from "@renderer/state/settings";
import { logger } from "@renderer/utils/logger";

export function useSyncAndLogsActions() {
  const offlineMode = useAtomValue(offlineModeAtom);
  const { getStatus, push, pull } = useCloudSync(offlineMode);
  const [isSyncingAll, setIsSyncingAll] = useState(false);
  const [isExportingData, setIsExportingData] = useState(false);
  const [isCreatingBackup, setIsCreatingBackup] = useState(false);
  const [isRestoringBackup, setIsRestoringBackup] = useState(false);

  const handleOpenLogsDirectory = async (): Promise<void> => {
    try {
      const result = await window.api.file.openLogsDirectory();
      if (result.success) {
        toast.success("ログフォルダを開きました");
      } else {
        toast.error(result.message || "ログフォルダを開くことができませんでした");
      }
    } catch (error) {
      logger.error("ログフォルダを開くエラー:", {
        component: "useSyncAndLogsActions",
        function: "handleOpenLogsDirectory",
        data: error,
      });
      toast.error("ログフォルダを開くことができませんでした");
    }
  };

  const handleSyncAllGames = async (): Promise<void> => {
    if (offlineMode) {
      toast.error("オフラインモードでは同期できません");
      return;
    }
    setIsSyncingAll(true);
    const toastId = toast.loading("全ゲームを同期中…");
    try {
      const games = await window.api.database.listGames("", "all", "title", "asc");
      let uploaded = 0;
      let downloaded = 0;
      let failed = 0;
      let skipped = 0;

      for (const game of games) {
        if (!game.saveFolderPath) continue;

        const statusResult = await getStatus(game.id);
        if (!statusResult.success || !statusResult.data) {
          failed++;
          continue;
        }

        const { status } = statusResult.data;
        if (status === "push_needed") {
          const op = await push(game.id);
          if (op.ok) uploaded++;
          else failed++;
        } else if (status === "pull_needed") {
          const op = await pull(game.id);
          // 同期管理外ファイルの削除確認が必要な場合は破壊を避けてスキップ（詳細画面で確認）
          if (op.ok && op.applied === false) skipped++;
          else if (op.ok) downloaded++;
          else failed++;
        } else if (status === "conflict") {
          skipped++;
        }
      }

      const parts: string[] = [];
      if (uploaded > 0) parts.push(`アップロード${uploaded}件`);
      if (downloaded > 0) parts.push(`ダウンロード${downloaded}件`);
      const suffix =
        (failed > 0 ? `（${failed}件失敗）` : "") + (skipped > 0 ? `（${skipped}件要確認）` : "");
      const message =
        parts.length > 0 ? `同期完了: ${parts.join(" / ")}${suffix}` : "すべて最新の状態です";
      toast.success(message, { id: toastId });
    } catch (error) {
      logger.error("全ゲーム同期エラー:", {
        component: "useSyncAndLogsActions",
        function: "handleSyncAllGames",
        data: error,
      });
      toast.error("クラウド同期に失敗しました", { id: toastId });
    } finally {
      setIsSyncingAll(false);
    }
  };

  const handleExportGameData = async (): Promise<void> => {
    const selected = await window.api.file.selectFolder();
    if (!selected.success || !selected.data) {
      return;
    }
    setIsExportingData(true);
    try {
      const result = await window.api.maintenance.exportGameData(selected.data);
      if (!result.success || !result.data) {
        toast.error((!result.success && result.message) || "データエクスポートに失敗しました");
        return;
      }
      toast.success("CSV/JSONのエクスポートが完了しました");
      await window.api.window.openFolder(selected.data);
    } catch (error) {
      logger.error("データエクスポートエラー:", {
        component: "useSyncAndLogsActions",
        function: "handleExportGameData",
        data: error,
      });
      toast.error("データエクスポートに失敗しました");
    } finally {
      setIsExportingData(false);
    }
  };

  const handleCreateBackup = async (): Promise<void> => {
    const selected = await window.api.file.selectFolder();
    if (!selected.success || !selected.data) {
      return;
    }
    setIsCreatingBackup(true);
    try {
      const result = await window.api.maintenance.createFullBackup(selected.data);
      if (!result.success || !result.data) {
        toast.error((!result.success && result.message) || "バックアップ作成に失敗しました");
        return;
      }
      toast.success("バックアップを作成しました");
      await window.api.window.openFolder(selected.data);
    } catch (error) {
      logger.error("バックアップ作成エラー:", {
        component: "useSyncAndLogsActions",
        function: "handleCreateBackup",
        data: error,
      });
      toast.error("バックアップ作成に失敗しました");
    } finally {
      setIsCreatingBackup(false);
    }
  };

  const handleRestoreBackup = async (): Promise<void> => {
    const selected = await window.api.file.selectFile([
      { name: "CloudLaunch backup", extensions: ["zip"] },
    ]);
    if (!selected.success || !selected.data) {
      return;
    }
    const accepted = window.confirm(
      "バックアップ復元を実行します。現在のローカルデータは上書きされます。続行しますか？",
    );
    if (!accepted) {
      return;
    }

    setIsRestoringBackup(true);
    try {
      const result = await window.api.maintenance.restoreFullBackup(selected.data);
      if (!result.success) {
        toast.error(result.message || "バックアップ復元に失敗しました");
        return;
      }
      toast.success("バックアップを復元しました");
      window.setTimeout(() => {
        window.location.reload();
      }, 300);
    } catch (error) {
      logger.error("バックアップ復元エラー:", {
        component: "useSyncAndLogsActions",
        function: "handleRestoreBackup",
        data: error,
      });
      toast.error("バックアップ復元に失敗しました");
    } finally {
      setIsRestoringBackup(false);
    }
  };

  return {
    offlineMode,
    isSyncingAll,
    isExportingData,
    isCreatingBackup,
    isRestoringBackup,
    handleSyncAllGames,
    handleExportGameData,
    handleCreateBackup,
    handleRestoreBackup,
    handleOpenLogsDirectory,
  };
}
