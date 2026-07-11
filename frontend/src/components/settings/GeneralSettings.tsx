/**
 * @fileoverview 一般設定コンポーネント
 *
 * アプリケーションの一般的な設定項目を管理するコンポーネントです。
 *
 * 主な機能：
 * - DaisyUIテーマの選択・変更
 * - デフォルトソート順の設定
 * - デフォルトフィルター状態の設定
 * - オフラインモードの設定
 * - 自動ゲーム検出の設定
 * - 設定の永続化
 * - リアルタイムでの変更反映
 *
 * 使用技術：
 * - Jotai atoms（状態管理）
 * - DaisyUI コンポーネント
 */

import { useAtom } from "jotai";
import { useEffect, useState } from "react";
import toast from "react-hot-toast";

import { logger } from "@renderer/utils/logger";

import { useCloudSync } from "@renderer/hooks/useCloudSync";
import { offlineModeAtom, autoTrackingAtom, transferConcurrencyAtom } from "../../state/settings";

import AppearanceTab from "./AppearanceTab";
import ScreenshotSettingsTab from "./ScreenshotSettingsTab";
import BehaviorTab from "./BehaviorTab";
import DefaultsTab from "./DefaultsTab";
import SyncAndLogsTab from "./SyncAndLogsTab";

/**
 * 一般設定コンポーネント
 *
 * テーマ選択、デフォルトソート順、デフォルトフィルター状態、
 * オフラインモード、自動ゲーム検出など、アプリケーションの一般的な設定を提供します。
 *
 * @returns 一般設定コンポーネント要素
 */
export default function GeneralSettings(): React.JSX.Element {
  type GeneralTab = "appearance" | "screenshot" | "behavior" | "defaults" | "maintenance";

  const [offlineMode, setOfflineMode] = useAtom(offlineModeAtom);
  const [autoTracking, setAutoTracking] = useAtom(autoTrackingAtom);
  const [transferConcurrency, setTransferConcurrency] = useAtom(transferConcurrencyAtom);
  const { getStatus, push, pull } = useCloudSync(offlineMode);
  const [isSyncingAll, setIsSyncingAll] = useState(false);
  const [isExportingData, setIsExportingData] = useState(false);
  const [isCreatingBackup, setIsCreatingBackup] = useState(false);
  const [isRestoringBackup, setIsRestoringBackup] = useState(false);
  const [activeTab, setActiveTab] = useState<GeneralTab>("appearance");

  // オフラインモード変更ハンドラー
  const handleOfflineModeChange = async (enabled: boolean): Promise<void> => {
    // バックエンド ContentSyncService にも反映しないと、process_monitor 経由の
    // 自動同期や直接の cloudSync.Push 呼び出しが UI 設定を無視して S3 にアクセスし続ける。
    const result = await window.api.settings.updateOfflineMode(enabled);
    if (!result.success) {
      logger.error("オフラインモード設定の更新エラー:", {
        component: "GeneralSettings",
        function: "handleOfflineModeChange",
        data: result.message,
      });
      toast.error("オフラインモードの更新に失敗しました");
      return;
    }
    setOfflineMode(enabled);
    if (enabled) {
      toast.success("オフラインモードを有効にしました");
    } else {
      toast.success("オフラインモードを無効にしました");
    }
  };

  // 自動ゲーム検出変更ハンドラー
  const handleAutoTrackingChange = async (enabled: boolean): Promise<void> => {
    try {
      const result = await window.api.settings.updateAutoTracking(enabled);
      if (result.success) {
        setAutoTracking(enabled);
        if (enabled) {
          toast.success("自動ゲーム検出を有効にしました");
        } else {
          toast.success("自動ゲーム検出を無効にしました");
        }
      } else {
        toast.error("設定の更新に失敗しました");
      }
    } catch (error) {
      logger.error("自動ゲーム検出設定の更新エラー:", {
        component: "GeneralSettings",
        function: "unknown",
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
        component: "GeneralSettings",
        function: "unknown",
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

  // ログフォルダを開くハンドラー
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
        component: "GeneralSettings",
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
        component: "GeneralSettings",
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
        component: "GeneralSettings",
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
        component: "GeneralSettings",
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
        component: "GeneralSettings",
        function: "handleRestoreBackup",
        data: error,
      });
      toast.error("バックアップ復元に失敗しました");
    } finally {
      setIsRestoringBackup(false);
    }
  };

  useEffect(() => {
    void applyTransferConcurrency(transferConcurrency, false);
  }, []);

  return (
    <div className="w-full">
      <h2 className="text-xl font-semibold mb-6">一般設定</h2>

      <div role="tablist" className="tabs tabs-boxed mb-6 overflow-x-auto">
        <button
          className={`tab ${activeTab === "appearance" ? "tab-active" : ""}`}
          onClick={() => setActiveTab("appearance")}
        >
          外観
        </button>
        <button
          className={`tab ${activeTab === "screenshot" ? "tab-active" : ""}`}
          onClick={() => setActiveTab("screenshot")}
        >
          スクリーンショット
        </button>
        <button
          className={`tab ${activeTab === "behavior" ? "tab-active" : ""}`}
          onClick={() => setActiveTab("behavior")}
        >
          動作
        </button>
        <button
          className={`tab ${activeTab === "defaults" ? "tab-active" : ""}`}
          onClick={() => setActiveTab("defaults")}
        >
          初期表示
        </button>
        <button
          className={`tab ${activeTab === "maintenance" ? "tab-active" : ""}`}
          onClick={() => setActiveTab("maintenance")}
        >
          同期・ログ
        </button>
      </div>

      {activeTab === "appearance" && <AppearanceTab />}

      {activeTab === "screenshot" && <ScreenshotSettingsTab />}

      {activeTab === "behavior" && (
        <BehaviorTab
          offlineMode={offlineMode}
          onOfflineModeChange={handleOfflineModeChange}
          autoTracking={autoTracking}
          onAutoTrackingChange={handleAutoTrackingChange}
          transferConcurrency={transferConcurrency}
          onTransferConcurrencyInputChange={setTransferConcurrency}
          onTransferConcurrencyBlur={handleTransferConcurrencyChange}
        />
      )}

      {activeTab === "defaults" && <DefaultsTab />}

      {activeTab === "maintenance" && (
        <SyncAndLogsTab
          offlineMode={offlineMode}
          onSyncAllGames={handleSyncAllGames}
          isSyncingAll={isSyncingAll}
          onExportGameData={handleExportGameData}
          isExportingData={isExportingData}
          onCreateBackup={handleCreateBackup}
          isCreatingBackup={isCreatingBackup}
          onRestoreBackup={handleRestoreBackup}
          isRestoringBackup={isRestoringBackup}
          onOpenLogsDirectory={handleOpenLogsDirectory}
        />
      )}
    </div>
  );
}
