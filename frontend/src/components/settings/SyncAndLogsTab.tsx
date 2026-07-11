import { Link } from "react-router-dom";
import { useState } from "react";
import toast from "react-hot-toast";

import { useSyncAndLogsActions } from "@renderer/hooks/useSyncAndLogsActions";
import { logLevelManager, type LogLevel } from "@renderer/utils/logLevel";
import { logger } from "@renderer/utils/logger";

import { TabSectionHeader } from "./TabSectionHeader";

export default function SyncAndLogsTab(): React.JSX.Element {
  const {
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
  } = useSyncAndLogsActions();

  const [frontendLogLevel, setFrontendLogLevel] = useState<LogLevel>(() =>
    logLevelManager.getCurrentLevel(),
  );

  const handleFrontendLogLevelChange = async (level: LogLevel): Promise<void> => {
    try {
      logLevelManager.setLevel(level);
      setFrontendLogLevel(level);
      if (level === "off") {
        toast.success("フロントエンドのログを無効にしました");
        return;
      }
      const result = await window.api.settings.updateLogLevel(level);
      if (!result.success) {
        toast.error("バックエンドのログレベル更新に失敗しました");
        return;
      }
      toast.success(`ログレベルを ${level} に設定しました`);
    } catch (error) {
      logger.error("ログレベル更新エラー:", {
        component: "SyncAndLogsTab",
        function: "handleFrontendLogLevelChange",
        data: error,
      });
      toast.error("ログレベルの更新に失敗しました");
    }
  };

  return (
    <div className="space-y-6">
      <TabSectionHeader
        title="データ・ログ"
        description="クラウド同期とトラブルシューティング"
        color="info"
      />

      <div className="bg-base-200 p-4 rounded-lg">
        <div className="mb-3">
          <h4 className="font-medium">クラウド同期</h4>
          <p className="text-sm text-base-content/70">ゲーム情報とセッションを同期します</p>
        </div>
        <div className="form-control">
          <button
            className="btn btn-outline btn-sm w-fit"
            onClick={() => void handleSyncAllGames()}
            disabled={isSyncingAll || offlineMode}
          >
            {isSyncingAll ? "同期中..." : "全ゲームを同期"}
          </button>
          <p className="text-xs text-base-content/50 mt-2">
            変更があったゲームのみクラウドと同期します
          </p>
        </div>
      </div>

      <div className="bg-base-200 p-4 rounded-lg">
        <div className="mb-3">
          <h4 className="font-medium">データエクスポート</h4>
          <p className="text-sm text-base-content/70">ゲーム情報と統計をCSV/JSONで保存します</p>
        </div>
        <div className="form-control">
          <button
            className="btn btn-outline btn-sm w-fit"
            onClick={() => void handleExportGameData()}
            disabled={isExportingData}
          >
            {isExportingData ? "エクスポート中..." : "CSV/JSONを出力"}
          </button>
          <p className="text-xs text-base-content/50 mt-2">
            出力先フォルダにタイムスタンプ付きファイルを生成します
          </p>
        </div>
      </div>

      <div className="bg-base-200 p-4 rounded-lg">
        <div className="mb-3">
          <h4 className="font-medium">バックアップ・復元</h4>
          <p className="text-sm text-base-content/70">全データをZIPで保存し、後で復元できます</p>
        </div>
        <div className="form-control gap-3">
          <button
            className="btn btn-outline btn-sm w-fit"
            onClick={() => void handleCreateBackup()}
            disabled={isCreatingBackup}
          >
            {isCreatingBackup ? "バックアップ作成中..." : "バックアップを作成"}
          </button>
          <button
            className="btn btn-warning btn-sm w-fit"
            onClick={() => void handleRestoreBackup()}
            disabled={isRestoringBackup}
          >
            {isRestoringBackup ? "復元中..." : "バックアップを復元"}
          </button>
          <p className="text-xs text-base-content/50 mt-1">
            復元時は現在のローカルデータを上書きします（認証情報はOS管理のため対象外）
          </p>
        </div>
      </div>

      <div className="bg-base-200 p-4 rounded-lg">
        <div className="mb-3">
          <h4 className="font-medium">ログレベル</h4>
          <p className="text-sm text-base-content/70">
            フロントエンドの出力レベル。off 以外はバックエンドにも反映します
          </p>
        </div>
        <select
          className="select select-bordered select-sm w-full max-w-xs"
          value={frontendLogLevel}
          onChange={(e) => void handleFrontendLogLevelChange(e.target.value as LogLevel)}
        >
          {logLevelManager.getAvailableLevels().map((level) => (
            <option key={level} value={level}>
              {level} — {logLevelManager.getLevelDescription(level)}
            </option>
          ))}
        </select>
      </div>

      <div className="bg-base-200 p-4 rounded-lg">
        <div className="mb-3">
          <h4 className="font-medium">ログ・デバッグ</h4>
          <p className="text-sm text-base-content/70">トラブルシューティング用</p>
        </div>
        <div className="form-control gap-3">
          <button
            className="btn btn-outline btn-sm w-fit"
            onClick={() => void handleOpenLogsDirectory()}
          >
            <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2H5a2 2 0 00-2-2z"
              />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 1v6" />
            </svg>
            ログフォルダを開く
          </button>
          <p className="text-xs text-base-content/50 mt-2">
            アプリケーションのログファイルが保存されているフォルダを開きます
          </p>

          <Link to="/debug/process" className="btn btn-outline btn-sm w-fit mt-4">
            プロセス監視デバッグを開く
          </Link>
          <p className="text-xs text-base-content/50 mt-2">プロセス監視の取得結果を確認します</p>
        </div>
      </div>
    </div>
  );
}
