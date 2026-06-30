import { Link } from "react-router-dom";

import { TabSectionHeader } from "./TabSectionHeader";

type SyncAndLogsTabProps = {
  offlineMode: boolean;
  onSyncAllGames: () => Promise<void>;
  isSyncingAll: boolean;
  onExportGameData: () => Promise<void>;
  isExportingData: boolean;
  onCreateBackup: () => Promise<void>;
  isCreatingBackup: boolean;
  onRestoreBackup: () => Promise<void>;
  isRestoringBackup: boolean;
  onOpenLogsDirectory: () => Promise<void>;
};

export default function SyncAndLogsTab({
  offlineMode,
  onSyncAllGames,
  isSyncingAll,
  onExportGameData,
  isExportingData,
  onCreateBackup,
  isCreatingBackup,
  onRestoreBackup,
  isRestoringBackup,
  onOpenLogsDirectory,
}: SyncAndLogsTabProps): React.JSX.Element {
  return (
    <div className="space-y-6">
      <TabSectionHeader
        title="同期・ログ"
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
            onClick={onSyncAllGames}
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
            onClick={onExportGameData}
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
            onClick={onCreateBackup}
            disabled={isCreatingBackup}
          >
            {isCreatingBackup ? "バックアップ作成中..." : "バックアップを作成"}
          </button>
          <button
            className="btn btn-warning btn-sm w-fit"
            onClick={onRestoreBackup}
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
          <h4 className="font-medium">ログ・デバッグ</h4>
          <p className="text-sm text-base-content/70">トラブルシューティング用</p>
        </div>
        <div className="form-control gap-3">
          <button className="btn btn-outline btn-sm w-fit" onClick={onOpenLogsDirectory}>
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
