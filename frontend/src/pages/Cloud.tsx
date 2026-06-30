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

import { CloudBreadcrumb } from "@renderer/components/cloud/CloudBreadcrumb";
import { CloudContent } from "@renderer/components/cloud/CloudContent";
import { CloudDeleteModal } from "@renderer/components/cloud/CloudDeleteModal";
import { CloudFileDetailsModal } from "@renderer/components/cloud/CloudFileDetailsModal";
import { CloudHeader, type ViewMode } from "@renderer/components/cloud/CloudHeader";

import { useCloudData } from "@renderer/hooks/useCloudData";
import { useOfflineMode } from "@renderer/hooks/useOfflineMode";
import { useValidateCreds } from "@renderer/hooks/useValidCreds";

import { logger } from "@renderer/utils/logger";

import { countFilesRecursively, sumSizesRecursively } from "@renderer/utils/cloudUtils";
import type { CloudDataItem, CloudDirectoryNode, CloudFileDetail } from "src/types/cloud";

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

  const validateCreds = useValidateCreds();
  const { isOfflineMode } = useOfflineMode();

  // クラウドデータ管理フック
  const {
    cloudData,
    directoryTree,
    loading,
    currentPath,
    currentDirectoryNodes,
    loadingGameIds,
    fetchCloudData,
    ensureGameLoaded,
    navigateToDirectory,
    navigateBack,
    navigateToPath,
    deleteGameFromCloud,
    deleteAllGamesFromCloud,
  } = useCloudData();

  /**
   * カード ⇄ ツリーの切替時は、もう一方のビュー由来の閲覧状態（カードビューの
   * currentPath とツリービューの expandedNodes）が見えないところで残らないように
   * リセットする。残したままだとパンくずやツリーの展開状態が不整合な見え方になる。
   *
   * CloudHeader のビュー切替ボタンは押すたびに同じモードでも onViewModeChange を
   * 発火するため、アクティブな側を再クリックしたときに閲覧位置をリセットしないよう
   * mode === viewMode で早期 return する。
   */
  const handleViewModeChange = useCallback(
    (mode: ViewMode): void => {
      if (mode === viewMode) {
        return;
      }
      setViewMode(mode);
      setExpandedNodes(new Set());
      navigateToPath([]);
    },
    [navigateToPath, viewMode],
  );

  /**
   * ディレクトリ（ゲームまたはサブフォルダ）を開く。
   * ルートレベル（=ゲーム）を開くときは、そのゲームのファイル一覧を遅延取得する。
   * currentPath は表示名ベースで管理するため、ナビゲーションには node.name を使う。
   */
  const handleNavigateToDirectory = useCallback(
    (node: CloudDirectoryNode): void => {
      if (currentPath.length === 0) {
        // node.path はルートではゲームID（remotePath）
        void ensureGameLoaded(node.path);
      }
      navigateToDirectory(node.name);
    },
    [currentPath.length, ensureGameLoaded, navigateToDirectory],
  );

  // ルートで開いているゲームノード（カードビューのサブ階層表示用）
  const openGameNode =
    currentPath.length > 0 ? directoryTree.find((node) => node.name === currentPath[0]) : undefined;
  // 開いているゲームのファイル一覧をまだ取得中かどうか
  const isOpenGameLoading = openGameNode ? loadingGameIds.has(openGameNode.path) : false;

  /**
   * ツリーノードの展開・折りたたみ
   */
  const handleToggleExpand = (path: string): void => {
    const newExpanded = new Set(expandedNodes);
    if (newExpanded.has(path)) {
      newExpanded.delete(path);
    } else {
      newExpanded.add(path);
      // トップレベル（=ゲーム）を展開するときにファイル一覧を遅延取得する。
      // ゲームノードの path は remotePath（ゲームID）と一致する。
      if (directoryTree.some((node) => node.path === path)) {
        void ensureGameLoaded(path);
      }
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
   * 全削除処理 - 全ゲームの削除確認モーダルを表示するためのセンチネルをセット
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
   * ゲーム単位のクラウドデータ削除（確認モーダルからコールバック）
   */
  const handleDelete = async (item: CloudDataItem | CloudDirectoryNode): Promise<void> => {
    try {
      // 全削除センチネル（path === "*"）
      if ("path" in item && (item as CloudDirectoryNode).path === "*") {
        await deleteAllGamesFromCloud();
        return;
      }

      // ゲーム単位削除：remotePath（CloudDataItem）または path（CloudDirectoryNode）をゲームIDとして使用
      const gameId = "remotePath" in item ? item.remotePath : (item as CloudDirectoryNode).path;
      await deleteGameFromCloud(gameId);
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

  // コンポーネントマウント時にタイトル一覧を取得（カード/ツリーで共通）
  useEffect(() => {
    fetchCloudData();
  }, [fetchCloudData]);

  useEffect(() => {
    if (!isOfflineMode) {
      validateCreds();
    }
  }, [validateCreds, isOfflineMode]);

  return (
    <div className="container mx-auto px-4 py-6">
      {/* ヘッダー */}
      <CloudHeader
        viewMode={viewMode}
        onViewModeChange={handleViewModeChange}
        cloudData={cloudData}
        directoryTree={directoryTree}
        loading={loading}
        onRefresh={() => fetchCloudData()}
        onDeleteAll={handleDeleteAll}
      />

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
        gameLoading={isOpenGameLoading}
        loadingGameIds={loadingGameIds}
        directoryTree={directoryTree}
        currentPath={currentPath}
        currentDirectoryNodes={currentDirectoryNodes}
        expandedNodes={expandedNodes}
        onToggleExpand={handleToggleExpand}
        onSelectNode={handleSelectNode}
        onDelete={(item) => setDeleteConfirm(item)}
        onNavigateToDirectory={handleNavigateToDirectory}
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
