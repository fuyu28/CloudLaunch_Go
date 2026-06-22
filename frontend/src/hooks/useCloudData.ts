/**
 * @fileoverview クラウドデータ管理カスタムフック
 *
 * クラウドデータの取得、削除、ナビゲーション機能を提供します。
 */

import { useState, useCallback, useRef, useMemo } from "react";
import { toast } from "react-hot-toast";

import { logger } from "@renderer/utils/logger";

import type { CloudDataItem, CloudDirectoryNode, CloudFileDetail } from "src/types/cloud";
import {
  countFilesRecursively,
  getNodesByPath,
  latestModifiedRecursively,
  sumSizesRecursively,
} from "@renderer/utils/cloudUtils";

export type { CloudDataItem, CloudDirectoryNode, CloudFileDetail } from "src/types/cloud";

/**
 * useCloudDataフックの戻り値の型定義
 */
export type UseCloudDataReturn = {
  // State
  cloudData: CloudDataItem[];
  directoryTree: CloudDirectoryNode[];
  loading: boolean;
  currentPath: string[];
  currentDirectoryNodes: CloudDirectoryNode[];

  // Actions
  fetchCloudData: (mode?: "cards" | "tree") => Promise<void>;
  navigateToDirectory: (directoryName: string) => void;
  navigateBack: () => void;
  navigateToPath: (newPath: string[]) => void;
  /** ゲーム単位でクラウドデータを削除する（gameId = remotePath または path） */
  deleteGameFromCloud: (gameId: string) => Promise<void>;
  /** 全ゲームのクラウドデータを一括削除する */
  deleteAllGamesFromCloud: () => Promise<void>;
  clearNavigationCache: () => void;
};

function buildCloudDataFromTree(tree: CloudDirectoryNode[]): CloudDataItem[] {
  return tree.map((node) => {
    if (node.isDirectory) {
      return {
        name: node.name,
        totalSize: sumSizesRecursively(node),
        fileCount: countFilesRecursively(node),
        lastModified: latestModifiedRecursively(node),
        remotePath: node.path,
      };
    }
    return {
      name: node.name,
      totalSize: node.size,
      fileCount: 1,
      lastModified: node.lastModified,
      remotePath: node.path,
    };
  });
}

/**
 * クラウドデータ管理フック
 */
export function useCloudData(): UseCloudDataReturn {
  // 状態を統合管理
  const [state, setState] = useState({
    cloudData: [] as CloudDataItem[],
    directoryTree: [] as CloudDirectoryNode[],
    loading: true,
    currentPath: [] as string[],
  });

  // ナビゲーションキャッシュ
  const navigationCacheRef = useRef<Map<string, CloudDirectoryNode[]>>(new Map());

  // 現在のディレクトリノードをメモ化
  const currentDirectoryNodes = useMemo(() => {
    if (state.directoryTree.length === 0) return [];
    return getNodesByPath(state.directoryTree, state.currentPath);
  }, [state.directoryTree, state.currentPath]);

  /**
   * ナビゲーションキャッシュをクリア
   */
  const clearNavigationCache = useCallback((): void => {
    navigationCacheRef.current.clear();
  }, []);

  /**
   * クラウドデータ一覧を取得
   */
  const fetchCloudData = useCallback(
    async (mode?: "cards" | "tree"): Promise<void> => {
      setState((prev) => ({ ...prev, loading: true }));

      try {
        const shouldFetchTree = mode === "tree" || mode === "cards" || mode === undefined;
        const treeResult = shouldFetchTree ? await window.api.cloudData.getDirectoryTree() : null;

        const directoryTree =
          treeResult && treeResult.success && treeResult.data ? treeResult.data : null;
        if (treeResult && !treeResult.success) {
          logger.warn("ディレクトリツリーの取得に失敗しました", {
            component: "useCloudData",
            function: "unknown",
          });
        }

        clearNavigationCache();
        setState((prev) => ({
          cloudData: directoryTree ? buildCloudDataFromTree(directoryTree) : prev.cloudData,
          directoryTree: directoryTree ?? prev.directoryTree,
          loading: false,
          currentPath: prev.currentPath,
        }));
      } catch (error) {
        logger.error("クラウドデータ取得エラー:", {
          component: "useCloudData",
          function: "unknown",
          data: error,
        });
        toast.error("クラウドデータの取得に失敗しました");
        setState((prev) => ({
          cloudData: mode === "cards" || mode === undefined ? [] : prev.cloudData,
          directoryTree: mode === "tree" || mode === undefined ? [] : prev.directoryTree,
          loading: false,
          currentPath: prev.currentPath,
        }));
      }
    },
    [clearNavigationCache],
  );

  /**
   * カードビューでディレクトリに移動
   */
  const navigateToDirectory = useCallback(
    (directoryName: string): void => {
      const trimmed = directoryName.trim();
      if (trimmed === "") {
        return;
      }
      const segments = trimmed.split("/").filter((segment) => segment.length > 0);
      const newPath = segments.length > 1 ? segments : [...state.currentPath, trimmed];
      setState((prev) => ({ ...prev, currentPath: newPath }));
    },
    [state.currentPath],
  );

  /**
   * カードビューで親ディレクトリに戻る
   */
  const navigateBack = useCallback((): void => {
    const newPath = state.currentPath.slice(0, -1);
    setState((prev) => ({ ...prev, currentPath: newPath }));
  }, [state.currentPath]);

  /**
   * 指定パスに直接移動
   */
  const navigateToPath = useCallback((newPath: string[]): void => {
    setState((prev) => ({ ...prev, currentPath: newPath }));
  }, []);

  /**
   * ゲーム単位でクラウドデータを削除する。
   * content-addressed ストレージではブロブ単位の削除は履歴破壊になるため、
   * 必ずゲーム全体を単位として deleteFromCloud を呼ぶ。
   */
  const deleteGameFromCloud = useCallback(
    async (gameId: string): Promise<void> => {
      try {
        const result = await window.api.cloudSync.deleteFromCloud(gameId);
        if (result.success) {
          toast.success("クラウドデータを削除しました");
        } else {
          toast.error(result.message ?? "削除に失敗しました");
          return;
        }

        // 削除後はキャッシュをクリアして最新データを取得
        navigationCacheRef.current.clear();
        await fetchCloudData();
      } catch (error) {
        logger.error("削除エラー:", {
          component: "useCloudData",
          function: "deleteGameFromCloud",
          data: error,
        });
        toast.error("削除に失敗しました");
      }
    },
    [fetchCloudData],
  );

  /**
   * 全ゲームのクラウドデータを一括削除する。
   * cloudData 配列の remotePath（= gameId）を順次 deleteFromCloud に渡す。
   */
  const deleteAllGamesFromCloud = useCallback(async (): Promise<void> => {
    try {
      const gameIds = state.cloudData.map((item) => item.remotePath);
      const results = await Promise.all(
        gameIds.map((gameId) => window.api.cloudSync.deleteFromCloud(gameId)),
      );
      const failedCount = results.filter((r) => !r.success).length;

      if (failedCount === 0) {
        toast.success("全てのクラウドデータを削除しました");
      } else if (failedCount < results.length) {
        toast.success(`一部のデータを削除しました（失敗: ${failedCount}件）`);
      } else {
        toast.error("削除に失敗しました");
      }

      // 削除後はキャッシュをクリアして最新データを取得
      navigationCacheRef.current.clear();
      await fetchCloudData();
    } catch (error) {
      logger.error("一括削除エラー:", {
        component: "useCloudData",
        function: "deleteAllGamesFromCloud",
        data: error,
      });
      toast.error("削除に失敗しました");
    }
  }, [state.cloudData, fetchCloudData]);

  return {
    // State
    cloudData: state.cloudData,
    directoryTree: state.directoryTree,
    loading: state.loading,
    currentPath: state.currentPath,
    currentDirectoryNodes,

    // Actions
    fetchCloudData,
    navigateToDirectory,
    navigateBack,
    navigateToPath,
    deleteGameFromCloud,
    deleteAllGamesFromCloud,
    clearNavigationCache,
  };
}
