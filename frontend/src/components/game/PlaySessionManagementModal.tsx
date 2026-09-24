/**
 * @fileoverview セッション管理モーダルコンポーネント
 *
 * このコンポーネントは、特定のゲームに関連するプレイセッション情報を表示し、管理する機能を提供します。
 */

import { useCallback, useEffect, useState } from "react";
import { FaEdit } from "react-icons/fa";
import { RxCross1 } from "react-icons/rx";

import { useTimeFormat } from "@renderer/hooks/useTimeFormat";
import { useToastHandler } from "@renderer/hooks/useToastHandler";

import { logger } from "@renderer/utils/logger";

import ConfirmModal from "../common/ConfirmModal";
import type { PlaySessionType } from "src/types/game";

function toDateTimeLocalValue(date: Date): string {
  const offsetDate = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return offsetDate.toISOString().slice(0, 16);
}

type EditFormData = {
  playedAt: string;
  hours: string;
  minutes: string;
  seconds: string;
};

type PlaySessionManagementModalProps = {
  isOpen: boolean;
  onClose: () => void;
  gameId: string;
  gameTitle: string;
  onProcessUpdated?: () => void;
};

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
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [editingProcess, setEditingProcess] = useState<PlaySessionType | undefined>(undefined);
  const [editFormData, setEditFormData] = useState<EditFormData>({
    playedAt: "",
    hours: "",
    minutes: "",
    seconds: "",
  });
  const { formatSmart, formatDateWithTime } = useTimeFormat();
  const { showToast } = useToastHandler();

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

  const openEditModal = useCallback((process: PlaySessionType) => {
    setEditingProcess(process);
    const date = new Date(process.playedAt);
    setEditFormData({
      playedAt: toDateTimeLocalValue(date),
      hours: String(Math.floor(process.duration / 3600)),
      minutes: String(Math.floor((process.duration % 3600) / 60)),
      seconds: String(process.duration % 60),
    });
    setIsEditModalOpen(true);
  }, []);

  const closeEditModal = useCallback(() => {
    setIsEditModalOpen(false);
    setEditingProcess(undefined);
    setEditFormData({ playedAt: "", hours: "", minutes: "", seconds: "" });
  }, []);

  const handleEditSession = useCallback(async () => {
    if (!editingProcess) return;

    try {
      const playedAt = new Date(editFormData.playedAt);
      const hours = Number(editFormData.hours);
      const minutes = Number(editFormData.minutes);
      const seconds = Number(editFormData.seconds);
      if (
        Number.isNaN(playedAt.getTime()) ||
        !Number.isInteger(hours) ||
        hours < 0 ||
        !Number.isInteger(minutes) ||
        minutes < 0 ||
        minutes > 59 ||
        !Number.isInteger(seconds) ||
        seconds < 0 ||
        seconds > 59
      ) {
        showToast("日時とプレイ時間を正しく入力してください", "error");
        return;
      }
      const result = await window.api.database.updateSession(
        editingProcess.id,
        playedAt,
        hours * 3600 + minutes * 60 + seconds,
      );
      if (!result.success) {
        showToast("セッションの更新に失敗しました", "error");
        return;
      }

      showToast("セッションを更新しました", "success");
      await fetchProcesses();
      onProcessUpdated?.();
      closeEditModal();
    } catch (error) {
      logger.error("セッション編集エラー:", {
        component: "PlaySessionManagementModal",
        function: "unknown",
        data: error,
      });
      showToast("セッションの更新に失敗しました", "error");
    }
  }, [editingProcess, editFormData, fetchProcesses, onProcessUpdated, showToast, closeEditModal]);

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

  const openDeleteModal = useCallback((processId: string) => {
    setSelectedProcessId(processId);
    setIsDeleteModalOpen(true);
  }, []);

  const closeDeleteModal = useCallback(() => {
    setIsDeleteModalOpen(false);
    setSelectedProcessId(undefined);
  }, []);

  useEffect(() => {
    if (isOpen) {
      fetchProcesses();
    }
  }, [isOpen, fetchProcesses]);

  return (
    <>
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
                                  className="btn btn-sm btn-outline btn-primary"
                                  onClick={() => openEditModal(process)}
                                >
                                  <FaEdit />
                                  編集
                                </button>
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

      <ConfirmModal
        id="delete-session-modal"
        isOpen={isDeleteModalOpen}
        message={`このセッションを削除しますか？\nこの操作は取り消せません。`}
        cancelText="キャンセル"
        confirmText="削除する"
        onConfirm={handleDeleteProcess}
        onCancel={closeDeleteModal}
      />

      <div className={`modal ${isEditModalOpen ? "modal-open" : ""}`}>
        <div className="modal-box">
          <h3 className="font-bold text-lg mb-4">セッション編集</h3>

          <div className="space-y-4">
            <div>
              <label className="label">
                <span className="label-text">プレイ日時</span>
              </label>
              <input
                type="datetime-local"
                className="input input-bordered w-full"
                value={editFormData.playedAt}
                onChange={(e) => setEditFormData((prev) => ({ ...prev, playedAt: e.target.value }))}
              />
            </div>
            <div className="grid grid-cols-3 gap-3">
              {(
                [
                  ["hours", "時間"],
                  ["minutes", "分"],
                  ["seconds", "秒"],
                ] as const
              ).map(([field, label]) => (
                <label key={field} className="form-control">
                  <span className="label-text mb-1">{label}</span>
                  <input
                    type="number"
                    min="0"
                    max={field === "hours" ? undefined : "59"}
                    className="input input-bordered"
                    value={editFormData[field]}
                    onChange={(e) =>
                      setEditFormData((prev) => ({ ...prev, [field]: e.target.value }))
                    }
                  />
                </label>
              ))}
            </div>
          </div>

          <div className="modal-action">
            <button className="btn btn-ghost" onClick={closeEditModal}>
              キャンセル
            </button>
            <button className="btn btn-primary" onClick={handleEditSession}>
              更新
            </button>
          </div>
        </div>
      </div>
    </>
  );
}
