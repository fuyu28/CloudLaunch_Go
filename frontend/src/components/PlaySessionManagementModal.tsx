/**
 * @fileoverview セッション管理モーダルコンポーネント
 *
 * このコンポーネントは、特定のゲームに関連するプレイセッション情報を表示し、管理する機能を提供します。
 *
 * 主な機能：
 * - セッション一覧の表示（名前、実行時間）
 * - セッションの削除
 * - セッションの編集（名前）
 * - モーダルの開閉制御
 *
 * @param isOpen - モーダルの開閉状態
 * @param onClose - モーダルを閉じる関数
 * @param gameId - 対象のゲームID
 * @param gameTitle - ゲームタイトル
 * @param onProcessUpdated - セッション情報更新時のコールバック
 */

import { useCallback, useEffect, useState } from "react";
import { RxCross1 } from "react-icons/rx";

import { useTimeFormat } from "@renderer/hooks/useTimeFormat";
import { useToastHandler } from "@renderer/hooks/useToastHandler";

import { logger } from "@renderer/utils/logger";

import ConfirmModal from "./ConfirmModal";
import type { PlaySessionType } from "src/types/game";

/**
 * セッション管理モーダルのProps
 */
type PlaySessionManagementModalProps = {
  /** モーダルの開閉状態 */
  isOpen: boolean;
  /** モーダルを閉じる関数 */
  onClose: () => void;
  /** 対象のゲームID */
  gameId: string;
  /** ゲームタイトル */
  gameTitle: string;
  /** セッション情報更新時のコールバック */
  onProcessUpdated?: () => void;
};

/**
 * セッション管理モーダルコンポーネント
 */
export default function PlaySessionManagementModal({
  isOpen,
  onClose,
  gameId,
  gameTitle,
  onProcessUpdated,
}: PlaySessionManagementModalProps): React.JSX.Element {
  const [processes, setProcesses] = useState<PlaySessionType[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedProcessId, setSelectedProcessId] = useState<string | undefined>(undefined);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const { formatSmart, formatDateWithTime } = useTimeFormat();
  const { showToast } = useToastHandler();

  /**
   * セッション情報を取得
   */
  const fetchProcesses = useCallback(async () => {
    if (!gameId) return;

    setLoading(true);
    try {
      const result = await window.api.database.getPlaySessions(gameId);
      if (result.success && result.data) {
        setProcesses(result.data);
      } else {
        showToast("セッション情報の取得に失敗しました", "error");
      }
    } catch (error) {
      logger.error("セッション情報取得エラー:", {
        component: "PlaySessionManagementModal",
        function: "unknown",
        data: error,
      });
      showToast("セッション情報の取得に失敗しました", "error");
    } finally {
      setLoading(false);
    }
  }, [gameId, showToast]);

  /**
   * セッション削除処理
   */
  const handleDeleteProcess = useCallback(async () => {
    if (!selectedProcessId) return;

    try {
      const result = await window.api.database.deletePlaySession(selectedProcessId);
      if (result.success) {
        showToast("セッションを削除しました", "success");
        await fetchProcesses();
        onProcessUpdated?.();
      } else {
        showToast("セッションの削除に失敗しました", "error");
      }
    } catch (error) {
      logger.error("セッション削除エラー:", {
        component: "PlaySessionManagementModal",
        function: "unknown",
        data: error,
      });
      showToast("セッションの削除に失敗しました", "error");
    } finally {
      setIsDeleteModalOpen(false);
      setSelectedProcessId(undefined);
    }
  }, [selectedProcessId, fetchProcesses, onProcessUpdated, showToast]);

  /**
   * 削除確認モーダルを開く
   */
  const openDeleteModal = useCallback((processId: string) => {
    setSelectedProcessId(processId);
    setIsDeleteModalOpen(true);
  }, []);

  /**
   * 削除確認モーダルを閉じる
   */
  const closeDeleteModal = useCallback(() => {
    setIsDeleteModalOpen(false);
    setSelectedProcessId(undefined);
  }, []);

  /**
   * モーダルが開かれたときにセッション情報を取得
   */
  useEffect(() => {
    if (isOpen) {
      fetchProcesses();
    }
  }, [isOpen, fetchProcesses]);

  /**
   * 選択されたセッションの情報を取得
   */
  const selectedProcess = processes.find((p) => p.id === selectedProcessId);

  return (
    <>
      {/* セッション管理モーダル */}
      <div className={`modal ${isOpen ? "modal-open" : ""}`}>
        <div className="modal-box max-w-4xl max-h-[80vh] flex flex-col">
          <div className="flex justify-between items-center mb-4 flex-shrink-0">
            <h3 className="font-bold text-lg">セッション管理 - {gameTitle}</h3>
            <button className="btn btn-sm btn-circle btn-ghost" onClick={onClose}>
              <RxCross1 />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto scrollbar-thin scrollbar-thumb-base-content/30 scrollbar-track-transparent">
            {loading ? (
              <div className="flex justify-center items-center py-8">
                <span className="loading loading-spinner loading-md"></span>
              </div>
            ) : (
              <div className="space-y-4">
                {processes.length === 0 ? (
                  <div className="text-center py-8 text-base-content/60">
                    このゲームに関連するセッションがありません
                  </div>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="table w-full">
                      <thead>
                        <tr>
                          <th>実行時間</th>
                          <th>プレイ日時</th>
                          <th>操作</th>
                        </tr>
                      </thead>
                      <tbody>
                        {processes.map((process) => (
                          <tr key={process.id}>
                            <td>{formatSmart(process.duration)}</td>
                            <td>{formatDateWithTime(process.playedAt)}</td>
                            <td>
                              <div className="flex gap-2">
                                <button
                                  className="btn btn-sm btn-outline btn-error"
                                  onClick={() => openDeleteModal(process.id)}
                                >
                                  削除
                                </button>
                              </div>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )}
          </div>

          <div className="modal-action flex-shrink-0">
            <button className="btn" onClick={onClose}>
              閉じる
            </button>
          </div>
        </div>
      </div>

      {/* 削除確認モーダル */}
      <ConfirmModal
        id="delete-session-modal"
        isOpen={isDeleteModalOpen}
        message={`このセッションを削除しますか？\nプレイ日時: ${
          selectedProcess ? formatDateWithTime(selectedProcess.playedAt) : "-"
        }\nこの操作は取り消せません。`}
        cancelText="キャンセル"
        confirmText="削除する"
        onConfirm={handleDeleteProcess}
        onCancel={closeDeleteModal}
      />
    </>
  );
}
