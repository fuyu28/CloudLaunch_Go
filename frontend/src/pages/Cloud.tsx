/**
 * @fileoverview クラウドデータ管理ページ
 *
 * このコンポーネントは、R2/S3クラウドストレージ上のデータを
 * エクスプローラー形式で閲覧・管理する機能を提供します。
 *
 * 主な機能：
 * - クラウドデータ一覧表示（フォルダビュー）
 * - ゲーム/フォルダの詳細情報表示
 * - データ削除機能（確認ダイアログ付き）
 * - ビュー切り替え（カード/ツリー）
 * - ナビゲーション機能
 */

import { useCallback, useEffect, useState } from "react";
import { useAtomValue } from "jotai";

import { CloudBreadcrumb } from "@renderer/components/CloudBreadcrumb";
import { CloudContent } from "@renderer/components/CloudContent";
import { CloudDeleteModal } from "@renderer/components/CloudDeleteModal";
import { CloudFileDetailsModal } from "@renderer/components/CloudFileDetailsModal";
import { CloudHeader, type ViewMode } from "@renderer/components/CloudHeader";

import { isValidCredsAtom } from "@renderer/state/credentials";

import {
  useCloudData,
  type CloudDataItem,
  type CloudFileDetail,
} from "@renderer/hooks/useCloudData";
import { useOfflineMode } from "@renderer/hooks/useOfflineMode";
import { useToastHandler } from "@renderer/hooks/useToastHandler";
import { useValidateCreds } from "@renderer/hooks/useValidCreds";

import { logger } from "@renderer/utils/logger";

import { countFilesRecursively, sumSizesRecursively } from "@renderer/utils/cloudUtils";
import type { CloudDirectoryNode } from "@renderer/utils/cloudUtils";
import type { GameType } from "src/types/game";

/**
 * クラウドデータ管理ページメインコンポーネント
 */
export default function Cloud(): React.JSX.Element {
  // 状態管理
  const [viewMode, setViewMode] = useState<ViewMode>("cards");
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
  const [deleteConfirm, setDeleteConfirm] = useState<CloudDataItem | CloudDirectoryNode | null>(
    null,
  );
  const [detailsModal, setDetailsModal] = useState<{
    item: CloudDataItem | null;
    files: CloudFileDetail[];
    loading: boolean;
  }>({
    item: null,
    files: [],
    loading: false,
  });
  const [games, setGames] = useState<GameType[]>([]);
  const [selectedGameId, setSelectedGameId] = useState<string>("");
  const [isSyncingGame, setIsSyncingGame] = useState(false);
  const [isLoadingGames, setIsLoadingGames] = useState(false);

  const isValidCreds = useAtomValue(isValidCredsAtom);
  const validateCreds = useValidateCreds();
  const { showToast } = useToastHandler();
  const { isOfflineMode } = useOfflineMode();

  // クラウドデータ管理フック
  const {
    cloudData,
    directoryTree,
    loading,
    currentPath,
    currentDirectoryNodes,
    fetchCloudData,
    navigateToDirectory,
    navigateBack,
    navigateToPath,
    deleteCloudData,
  } = useCloudData();

  const fetchGames = useCallback(async (): Promise<void> => {
    setIsLoadingGames(true);
    try {
      const gameList = await window.api.database.listGames("", "all", "title", "asc");
      setGames(gameList);
      if (gameList.length > 0) {
        setSelectedGameId((prev) =>
          gameList.some((game) => game.id === prev) ? prev : gameList[0].id,
        );
      } else {
        setSelectedGameId("");
      }
    } catch (error) {
      logger.error("ゲーム一覧の取得に失敗しました:", {
        component: "Cloud",
        function: "fetchGames",
        data: error,
      });
      showToast("ゲーム一覧の取得に失敗しました", "error");
      setGames([]);
      setSelectedGameId("");
    } finally {
      setIsLoadingGames(false);
    }
  }, [showToast]);

  const handleSyncSelectedGame = useCallback(async (): Promise<void> => {
    if (!selectedGameId) {
      showToast("同期するゲームを選択してください", "error");
      return;
    }
    if (isOfflineMode) {
      showToast("オフラインモードでは同期できません", "error");
      return;
    }
    setIsSyncingGame(true);
    try {
      const result = await window.api.cloudSync.syncGame(selectedGameId);
      if (!result.success || !result.data) {
        showToast(result.message || "クラウド同期に失敗しました", "error");
        return;
      }
      showToast(
        `同期完了: アップロード${result.data.uploadedGames}件 / ダウンロード${result.data.downloadedGames}件`,
        "success",
      );
    } catch (error) {
      logger.error("ゲーム同期エラー:", {
        component: "Cloud",
        function: "handleSyncSelectedGame",
        data: error,
      });
      showToast("クラウド同期に失敗しました", "error");
    } finally {
      setIsSyncingGame(false);
    }
  }, [selectedGameId, isOfflineMode, showToast]);

  /**
   * ツリーノードの展開・折りたたみ
   */
  const handleToggleExpand = (path: string): void => {
    const newExpanded = new Set(expandedNodes);
    if (newExpanded.has(path)) {
      newExpanded.delete(path);
    } else {
      newExpanded.add(path);
    }
    setExpandedNodes(newExpanded);
  };

  /**
   * ツリーノード選択
   */
  const handleSelectNode = (node: CloudDirectoryNode): void => {
    if (!node.isDirectory) {
      logger.debug("ファイルが選択されました:", {
        component: "Cloud",
        function: "unknown",
        data: node.name,
      });
    } else {
      handleToggleExpand(node.path);
    }
  };

  /**
   * 全削除処理
   */
  const handleDeleteAll = (): void => {
    const allDeleteItem = {
      name: "全てのクラウドデータ",
      path: "*",
      isDirectory: true,
      size: cloudData.reduce((sum, item) => sum + item.totalSize, 0),
      lastModified: new Date(),
      children: [],
    } as CloudDirectoryNode;
    setDeleteConfirm(allDeleteItem);
  };

  /**
   * クラウドデータを削除
   */
  const handleDelete = async (item: CloudDataItem | CloudDirectoryNode): Promise<void> => {
    try {
      await deleteCloudData(item);
    } finally {
      setDeleteConfirm(null);
    }
  };

  /**
   * ファイル詳細を表示
   */
  const handleViewDetails = async (node: CloudDirectoryNode): Promise<void> => {
    const detailItem: CloudDataItem = {
      name: node.name,
      totalSize: node.isDirectory ? sumSizesRecursively(node) : node.size,
      fileCount: node.isDirectory ? countFilesRecursively(node) : 1,
      lastModified: node.lastModified,
      remotePath: node.path,
    };

    setDetailsModal({ item: detailItem, files: [], loading: true });

    try {
      const result = await window.api.cloudData.getCloudFileDetails(detailItem.remotePath);
      if (result.success && result.data) {
        setDetailsModal((prev) => ({
          ...prev,
          files: result.data!,
          loading: false,
        }));
      } else {
        import("react-hot-toast").then(({ toast }) => {
          toast.error("ファイル詳細の取得に失敗しました");
        });
        setDetailsModal((prev) => ({ ...prev, loading: false }));
      }
    } catch (error) {
      logger.error("ファイル詳細取得エラー:", {
        component: "Cloud",
        function: "unknown",
        data: error,
      });
      import("react-hot-toast").then(({ toast }) => {
        toast.error("ファイル詳細の取得に失敗しました");
      });
      setDetailsModal((prev) => ({ ...prev, loading: false }));
    }
  };

  // コンポーネントマウント時にデータを取得
  useEffect(() => {
    fetchCloudData();
  }, [fetchCloudData]);

  useEffect(() => {
    if (!isOfflineMode) {
      validateCreds();
    }
  }, [validateCreds, isOfflineMode]);

  useEffect(() => {
    fetchGames();
  }, [fetchGames]);

  return (
    <div className="container mx-auto px-4 py-6">
      {/* ヘッダー */}
      <CloudHeader
        viewMode={viewMode}
        onViewModeChange={setViewMode}
        cloudData={cloudData}
        directoryTree={directoryTree}
        loading={loading}
        onRefresh={fetchCloudData}
        onDeleteAll={handleDeleteAll}
      />

      {/* ゲーム情報同期 */}
      <div className="card bg-base-100 shadow-xl mb-4">
        <div className="card-body">
          <h3 className="font-semibold text-lg">ゲーム情報の同期</h3>
          <p className="text-sm text-base-content/60 mb-4">
            タイトル・プレイ情報・セッションをクラウドと同期します
          </p>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <select
              className="select select-bordered w-full sm:max-w-xs"
              value={selectedGameId}
              onChange={(event) => setSelectedGameId(event.target.value)}
              disabled={isLoadingGames || games.length === 0}
            >
              {games.length === 0 ? (
                <option value="">ゲームがありません</option>
              ) : (
                <option value="">ゲームを選択</option>
              )}
              {games.map((game) => (
                <option key={game.id} value={game.id}>
                  {game.title}
                </option>
              ))}
            </select>
            <button
              className="btn btn-outline btn-sm w-fit"
              onClick={handleSyncSelectedGame}
              disabled={isSyncingGame || isOfflineMode || !selectedGameId}
            >
              {isSyncingGame ? "同期中..." : "選択したゲームを同期"}
            </button>
            <button
              className="btn btn-ghost btn-sm w-fit"
              onClick={fetchGames}
              disabled={isLoadingGames}
            >
              {isLoadingGames ? "更新中..." : "一覧を更新"}
            </button>
          </div>
          {!isOfflineMode && !isValidCreds && (
            <p className="text-xs text-error mt-3">クラウド認証情報が未設定です</p>
          )}
          {isOfflineMode && (
            <p className="text-xs text-warning mt-3">オフラインモードのため同期できません</p>
          )}
        </div>
      </div>

      {/* パンくずリスト */}
      <CloudBreadcrumb
        currentPath={currentPath}
        onNavigateToPath={navigateToPath}
        onNavigateBack={navigateBack}
      />

      {/* コンテンツ */}
      <CloudContent
        viewMode={viewMode}
        loading={loading}
        directoryTree={directoryTree}
        currentPath={currentPath}
        currentDirectoryNodes={currentDirectoryNodes}
        expandedNodes={expandedNodes}
        onToggleExpand={handleToggleExpand}
        onSelectNode={handleSelectNode}
        onDelete={(item) => setDeleteConfirm(item)}
        onNavigateToDirectory={navigateToDirectory}
        onViewDetails={handleViewDetails}
      />

      {/* 削除確認ダイアログ */}
      <CloudDeleteModal
        deleteConfirm={deleteConfirm}
        onCancel={() => setDeleteConfirm(null)}
        onConfirm={handleDelete}
        cloudData={cloudData}
      />

      {/* ファイル詳細モーダル */}
      <CloudFileDetailsModal
        isOpen={!!detailsModal.item}
        onClose={() => setDetailsModal({ item: null, files: [], loading: false })}
        item={detailsModal.item}
        files={detailsModal.files}
        loading={detailsModal.loading}
      />
    </div>
  );
}
