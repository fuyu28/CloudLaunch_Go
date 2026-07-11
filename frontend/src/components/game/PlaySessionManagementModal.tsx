/**
 * @fileoverview セッション管理モーダルコンポーネント
 *
 * このコンポーネントは、特定のゲームに関連するプレイセッション情報を表示し、管理する機能を提供します。
 */

import { useCallback, useEffect, useState, useMemo } from "react";
import { FaEdit } from "react-icons/fa";
import { RxCross1 } from "react-icons/rx";

import { useTimeFormat } from "@renderer/hooks/useTimeFormat";
import { useToastHandler } from "@renderer/hooks/useToastHandler";

import { logger } from "@renderer/utils/logger";

import ConfirmModal from "../common/ConfirmModal";
import { playSessionEditSchema } from "@renderer/schemas/playSession";
import type { PlaySessionType } from "src/types/game";
import { useZodValidation } from "../../hooks/useZodValidation";

/**
 * 編集用のフォームデータ
 */
type EditFormData = Record<string, unknown> & {
  sessionName: string;
};

/**
 * 編集フォームのフィールド名の型
 */
type EditFormFields = keyof Pick<EditFormData, "sessionName">;

/**
 * セッション管理モーダルのProps
 */
type PlaySessionManagementModalProps = {
  isOpen: boolean;
  onClose: () => void;
  gameId: string;
  gameTitle: string;
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
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [editingProcess, setEditingProcess] = useState<PlaySessionType | undefined>(undefined);
  const [editFormData, setEditFormData] = useState<EditFormData>({
    sessionName: "",
  });

  const memoizedEditFormData = useMemo(() => editFormData, [editFormData]);

  const validation = useZodValidation(playSessionEditSchema, memoizedEditFormData);
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
   * 編集モーダルを開く
   */
  const openEditModal = useCallback(
    (process: PlaySessionType) => {
      setEditingProcess(process);
      // フォームの初期値には表示用フォールバック "未設定" を含めず、空文字を入れる。
      // 表示側のフォールバックは placeholder（下記 input）に分離しており、
      // ユーザが編集せずに更新を押しても "未設定" 文字列が保存されない。
      setEditFormData({
        sessionName: process.sessionName ?? "",
      });
      validation.resetTouched(); // タッチ状態をリセット
      setIsEditModalOpen(true);
    },
    [validation],
  );

  /**
   * 編集モーダルを閉じる
   */
  const closeEditModal = useCallback(() => {
    setIsEditModalOpen(false);
    setEditingProcess(undefined);
    setEditFormData({
      sessionName: "",
    });
    validation.resetTouched(); // タッチ状態をリセット
  }, [validation]);

  /**
   * フォーム入力変更処理
   */
  const handleFormChange = useCallback(
    (field: EditFormFields, value: string | null) => {
      setEditFormData((prev) => ({ ...prev, [field]: value ?? "" }));
      validation.touch(field);
    },
    [validation],
  );

  /**
   * セッション編集処理
   */
  const handleEditSession = useCallback(async () => {
    if (!editingProcess) return;

    const validationResult = validation.validate();
    if (!validationResult.isValid) {
      showToast("入力内容に問題があります", "error");
      return;
    }

    try {
      // セッション名を更新
      // 既存の未設定（null / undefined）と空文字は同一視し、変更がなければ API を叩かない。
      const nextName = memoizedEditFormData.sessionName;
      const prevName = editingProcess.sessionName ?? "";
      if (nextName !== prevName) {
        const nameResult = await window.api.database.updateSessionName(editingProcess.id, nextName);
        if (!nameResult.success) {
          showToast("セッション名の更新に失敗しました", "error");
          return;
        }
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
  }, [
    editingProcess,
    memoizedEditFormData,
    validation,
    fetchProcesses,
    onProcessUpdated,
    showToast,
    closeEditModal,
  ]);

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
                          <th>セッション名</th>
                          <th>実行時間</th>
                          <th>プレイ日時</th>
                          <th>操作</th>
                        </tr>
                      </thead>
                      <tbody>
                        {processes.map((process) => (
                          <tr key={process.id}>
                            <td>
                              <div className="font-medium">{process.sessionName ?? "未設定"}</div>
                            </td>
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
        message={`セッション「${selectedProcess?.sessionName || "未設定"}」を削除しますか？\nこの操作は取り消せません。`}
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
                <span className="label-text">セッション名</span>
              </label>
              <input
                type="text"
                className={`input input-bordered w-full ${
                  validation.hasError("sessionName") ? "input-error" : ""
                }`}
                value={editFormData.sessionName}
                onChange={(e) => handleFormChange("sessionName", e.target.value)}
                placeholder="未設定"
              />
              {validation.getError("sessionName") && (
                <div className="text-error text-sm mt-1">{validation.getError("sessionName")}</div>
              )}
            </div>
          </div>

          <div className="modal-action">
            <button className="btn btn-ghost" onClick={closeEditModal}>
              キャンセル
            </button>
            <button
              className="btn btn-primary"
              onClick={handleEditSession}
              disabled={validation.hasError("sessionName")}
            >
              更新
            </button>
          </div>
        </div>
      </div>
    </>
  );
}
