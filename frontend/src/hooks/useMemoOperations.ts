/**
 * @fileoverview メモ操作フック
 *
 * メモの削除、編集、閲覧などの共通操作を提供します。
 * ゲーム詳細ページとメモ一覧ページで共通使用されます。
 */

import { useNavigate } from "react-router-dom";

import { logger } from "@renderer/utils/logger";

import { useToastHandler } from "./useToastHandler";

type UseMemoOperationsProps = {
  gameId?: string;
  onDeleteSuccess?: (deletedMemoId: string) => void;
  closeDropdown: () => void;
  openDeleteModal: (memoId: string) => void;
  onSyncSuccess?: () => void;
};

type UseMemoOperationsReturn = {
  handleDeleteMemo: (memoId: string) => Promise<void>;
  handleEditMemo: (memoId: string, event: React.MouseEvent) => void;
  handleViewMemo: (memoId: string) => void;
  handleDeleteConfirm: (memoId: string, event: React.MouseEvent) => void;
  handleSyncFromCloud: (event: React.MouseEvent) => Promise<void>;
};

/**
 * 以前はハンドラを useCallback でメモ化していたが、以下の理由により生の関数に戻した:
 *   - MemoCardBase の memo 比較関数がハンドラを比較対象外にしているためメモ化の意味がない
 *   - 呼び出し元の MemoList/MemoCard が結局インライン矢印関数を渡すので参照は毎回変わる
 * 依存配列の管理コストに対してリターンがないため撤去している。
 */
export function useMemoOperations({
  gameId,
  onDeleteSuccess,
  closeDropdown,
  openDeleteModal,
  onSyncSuccess,
}: UseMemoOperationsProps): UseMemoOperationsReturn {
  const navigate = useNavigate();
  const { showToast } = useToastHandler();

  const handleDeleteMemo = async (memoId: string): Promise<void> => {
    try {
      const result = await window.api.memo.deleteMemo(memoId);
      if (result.success) {
        showToast("メモを削除しました", "success");
        onDeleteSuccess?.(memoId);
      } else {
        showToast("メモの削除に失敗しました", "error");
      }
    } catch (error) {
      logger.error("メモ削除エラー:", {
        component: "useMemoOperations",
        function: "unknown",
        data: error,
      });
      showToast("メモの削除中にエラーが発生しました", "error");
    }
  };

  const handleEditMemo = (memoId: string, event: React.MouseEvent): void => {
    event.stopPropagation();
    closeDropdown();

    if (gameId) {
      // ゲーム詳細起点なら from=game を付け、戻り先を失わないようにする。
      navigate(`/memo/edit/${memoId}?from=game&gameId=${gameId}`);
    } else {
      // 一覧起点では from を付けない（詳細文脈を捏造しない）。
      navigate(`/memo/edit/${memoId}`);
    }
  };

  const handleViewMemo = (memoId: string): void => {
    if (gameId) {
      // ゲーム詳細起点なら from=game を付け、戻り先を失わないようにする。
      navigate(`/memo/view/${memoId}?from=game&gameId=${gameId}`);
    } else {
      // 一覧起点では from を付けない（詳細文脈を捏造しない）。
      navigate(`/memo/view/${memoId}`);
    }
  };

  const handleDeleteConfirm = (memoId: string, event: React.MouseEvent): void => {
    event.stopPropagation();
    closeDropdown();
    openDeleteModal(memoId);
  };

  const handleSyncFromCloud = async (event: React.MouseEvent): Promise<void> => {
    event.stopPropagation();
    closeDropdown();

    try {
      const result = await window.api.memo.syncMemosFromCloud(gameId);
      if (result.success && result.data) {
        logger.debug("同期結果:", {
          component: "useMemoOperations",
          function: "unknown",
          data: result.data,
        });
        const { uploaded, created, localOverwritten, cloudOverwritten, skipped } = result.data;
        showToast(
          `同期完了: 新規アップロード${uploaded ?? 0}件、作成${created}件、ローカル更新${localOverwritten}件、クラウド更新${cloudOverwritten}件、スキップ${skipped}件`,
          "success",
        );
        onSyncSuccess?.();
      } else {
        showToast("メモの同期に失敗しました", "error");
      }
    } catch (error) {
      logger.error("メモ同期エラー:", {
        component: "useMemoOperations",
        function: "unknown",
        data: error,
      });
      showToast("メモの同期中にエラーが発生しました", "error");
    }
  };

  return {
    handleDeleteMemo,
    handleEditMemo,
    handleViewMemo,
    handleDeleteConfirm,
    handleSyncFromCloud,
  };
}
