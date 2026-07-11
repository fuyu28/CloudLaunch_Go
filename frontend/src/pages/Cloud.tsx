/**
 * @fileoverview クラウドデータ管理ページ
 *
 * このコンポーネントは、R2/S3クラウドストレージ上のデータを
 * エクスプローラー形式で閲覧・管理する機能を提供します。
 */

import { useCallback, useEffect, useState } from "react";
import { useAtomValue } from "jotai";

import { CloudBreadcrumb } from "@renderer/components/cloud/CloudBreadcrumb";
import { CloudContent } from "@renderer/components/cloud/CloudContent";
import { CloudDeleteModal } from "@renderer/components/cloud/CloudDeleteModal";
import { CloudFileDetailsModal } from "@renderer/components/cloud/CloudFileDetailsModal";
import { CloudHeader, type ViewMode } from "@renderer/components/cloud/CloudHeader";

import { isValidCredsAtom } from "@renderer/state/credentials";

import { useCloudData } from "@renderer/hooks/useCloudData";
import { useLatestRequestId } from "@renderer/hooks/useLatestRequestId";
import { useOfflineMode } from "@renderer/hooks/useOfflineMode";
import { useValidateCreds } from "@renderer/hooks/useValidCreds";

import { logger } from "@renderer/utils/logger";

import { countFilesRecursively, sumSizesRecursively } from "@renderer/utils/cloudUtils";
import type { CloudDataItem, CloudDirectoryNode, CloudFileDetail } from "src/types/cloud";

export default function Cloud(): React.JSX.Element {
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
  const isValidCreds = useAtomValue(isValidCredsAtom);

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
   * ナビゲーションは表示名ではなく node（実体）で辿るため、同名ゲームが 2 件あっても
   * 1 件に集約されない。
   */
  const handleNavigateToDirectory = useCallback(
    (node: CloudDirectoryNode): void => {
      if (currentPath.length === 0) {
        // ルート node.path はゲームID。配下パスと混同しない。
        void ensureGameLoaded(node.path);
      }
      navigateToDirectory(node);
    },
    [currentPath.length, ensureGameLoaded, navigateToDirectory],
  );

  // ルートで開いているゲームノード（カードビューのサブ階層表示用）。
  // 表示名ではなくパス（一意）で解決する。
  const openGameNode =
    currentPath.length > 0
      ? directoryTree.find((node) => node.path === currentPath[0].id)
      : undefined;
  // 遅延取得中は空一覧を「0ファイル」と誤表示しないためのフラグ。
  const isOpenGameLoading = openGameNode ? loadingGameIds.has(openGameNode.path) : false;

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
      // path==="*" は全削除センチネル。通常パスと分岐を分ける。
      if ("path" in item && (item as CloudDirectoryNode).path === "*") {
        await deleteAllGamesFromCloud();
        return;
      }

      // カード/ツリーでキー名が違うので remotePath と path の両方を見る。
      const gameId = "remotePath" in item ? item.remotePath : (item as CloudDirectoryNode).path;
      await deleteGameFromCloud(gameId);
    } finally {
      setDeleteConfirm(null);
    }
  };

  // 詳細モーダルの最新リクエストID。連続クリック時に古い resolve が
  // 新しく開いた対象のファイル一覧を上書きしないようガードするため使う。
  const detailsRequest = useLatestRequestId();

  const handleViewDetails = async (node: CloudDirectoryNode): Promise<void> => {
    const detailItem: CloudDataItem = {
      name: node.name,
      totalSize: node.isDirectory ? sumSizesRecursively(node) : node.size,
      fileCount: node.isDirectory ? countFilesRecursively(node) : 1,
      lastModified: node.lastModified,
      remotePath: node.path,
    };

    // 古い応答で state を上書きしないよう最新リクエスト ID を検証する。
    const reqId = detailsRequest.next();
    setDetailsModal({ item: detailItem, files: [], loading: true });

    try {
      const result = await window.api.cloudData.getCloudFileDetails(detailItem.remotePath);
      if (!detailsRequest.isLatest(reqId)) {
        // 別のノードのクリックで置き換わっているので何もしない
        return;
      }
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
      if (!detailsRequest.isLatest(reqId)) {
        return;
      }
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
      <CloudHeader
        viewMode={viewMode}
        onViewModeChange={handleViewModeChange}
        cloudData={cloudData}
        directoryTree={directoryTree}
        loading={loading}
        onRefresh={() => fetchCloudData()}
        onDeleteAll={handleDeleteAll}
      />

      {/* 取得不能状態の案内: リストが空に見える理由を明示する */}
      {isOfflineMode ? (
        <div className="alert alert-warning text-sm mb-4">
          オフラインモードのため、クラウドデータを取得できません
        </div>
      ) : !isValidCreds ? (
        <div className="alert alert-warning text-sm mb-4">
          クラウド認証情報が未設定のため、クラウドデータを取得できません
        </div>
      ) : null}

      <CloudBreadcrumb
        currentPath={currentPath}
        onNavigateToPath={navigateToPath}
        onNavigateBack={navigateBack}
      />

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

      <CloudDeleteModal
        deleteConfirm={deleteConfirm}
        onCancel={() => setDeleteConfirm(null)}
        onConfirm={handleDelete}
        cloudData={cloudData}
      />

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
