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
  /** 現在ファイル一覧を遅延取得中のゲームID（remotePath）集合 */
  loadingGameIds: Set<string>;

  // Actions
  fetchCloudData: () => Promise<void>;
  /** 指定ゲームのファイル一覧を遅延取得し、ディレクトリツリーへマージする */
  ensureGameLoaded: (gameId: string) => Promise<void>;
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
    loadingGameIds: new Set<string>(),
  });

  // ナビゲーションキャッシュ
  const navigationCacheRef = useRef<Map<string, CloudDirectoryNode[]>>(new Map());
  // 詳細を取得済みのゲームID（再ナビゲーション時の二重取得を防ぐ）
  const loadedGamesRef = useRef<Set<string>>(new Set());
  // 取得中のゲームID（同時呼び出しによる二重取得を防ぐ）
  const loadingGamesRef = useRef<Set<string>>(new Set());

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
   * クラウドデータ一覧（タイトルのみ）を取得する。
   * 初期表示ではファイル数・サイズは取得せず、ゲームを開いたときに ensureGameLoaded で遅延取得する。
   */
  const fetchCloudData = useCallback(async (): Promise<void> => {
    setState((prev) => ({ ...prev, loading: true }));

    try {
      const result = await window.api.cloudData.getCloudGameSummaries();
      if (!result.success) {
        logger.warn("クラウドサマリの取得に失敗しました", {
          component: "useCloudData",
          function: "fetchCloudData",
        });
        toast.error("クラウドデータの取得に失敗しました");
        setState((prev) => ({ ...prev, loading: false }));
        return;
      }

      const summaries = result.data ?? [];
      // 各ゲームを「未取得（children: undefined）」のトップノードとして表示する。
      const topNodes: CloudDirectoryNode[] = summaries.map((summary) => ({
        name: summary.name,
        path: summary.remotePath,
        isDirectory: true,
        size: 0,
        lastModified: summary.lastModified,
        children: undefined,
      }));

      loadedGamesRef.current.clear();
      loadingGamesRef.current.clear();
      clearNavigationCache();
      setState({
        cloudData: buildCloudDataFromTree(topNodes),
        directoryTree: topNodes,
        loading: false,
        currentPath: [],
        loadingGameIds: new Set(),
      });
    } catch (error) {
      logger.error("クラウドデータ取得エラー:", {
        component: "useCloudData",
        function: "fetchCloudData",
        data: error,
      });
      toast.error("クラウドデータの取得に失敗しました");
      setState((prev) => ({ ...prev, loading: false }));
    }
  }, [clearNavigationCache]);

  /**
   * 指定ゲームのファイル一覧（サイズ付き）を遅延取得し、ディレクトリツリーへマージする。
   * 取得済み・取得中のゲームは再取得しない。
   */
  const ensureGameLoaded = useCallback(
    async (gameId: string): Promise<void> => {
      if (!gameId) {
        return;
      }
      if (loadedGamesRef.current.has(gameId) || loadingGamesRef.current.has(gameId)) {
        return;
      }
      loadingGamesRef.current.add(gameId);
      setState((prev) => {
        const next = new Set(prev.loadingGameIds);
        next.add(gameId);
        return { ...prev, loadingGameIds: next };
      });

      try {
        const result = await window.api.cloudData.getGameDirectoryNode(gameId);
        if (!result.success || !result.data) {
          logger.warn("ゲームのディレクトリ取得に失敗しました", {
            component: "useCloudData",
            function: "ensureGameLoaded",
            data: gameId,
          });
          toast.error("ゲームのデータ取得に失敗しました");
          return;
        }

        // children が undefined（=ファイルなし）でも「取得済み」と区別できるよう空配列にする。
        const loadedNode: CloudDirectoryNode = {
          ...result.data,
          children: result.data.children ?? [],
        };
        loadedGamesRef.current.add(gameId);
        clearNavigationCache();
        setState((prev) => {
          const directoryTree = prev.directoryTree.map((node) =>
            node.path === gameId ? loadedNode : node,
          );
          return {
            ...prev,
            directoryTree,
            cloudData: buildCloudDataFromTree(directoryTree),
          };
        });
      } catch (error) {
        logger.error("ゲームのデータ取得エラー:", {
          component: "useCloudData",
          function: "ensureGameLoaded",
          data: error,
        });
        toast.error("ゲームのデータ取得に失敗しました");
      } finally {
        loadingGamesRef.current.delete(gameId);
        setState((prev) => {
          const next = new Set(prev.loadingGameIds);
          next.delete(gameId);
          return { ...prev, loadingGameIds: next };
        });
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
    loadingGameIds: state.loadingGameIds,

    // Actions
    fetchCloudData,
    ensureGameLoaded,
    navigateToDirectory,
    navigateBack,
    navigateToPath,
    deleteGameFromCloud,
    deleteAllGamesFromCloud,
    clearNavigationCache,
  };
}
