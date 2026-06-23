/**
 * @fileoverview 同期状態の確認モーダル
 *
 * 「同期確認」押下時に、いきなりアップロード/ダウンロードするのではなく、
 * 現在の同期状態（最新 / アップロード可 / ダウンロード可 / 未同期）と
 * ローカル・クラウドの情報を表示し、ユーザーが操作を選べるようにする。
 * 競合（conflict）は専用の SyncConflictModal で扱うため、ここでは扱わない。
 */

import {
  FaDesktop,
  FaCloud,
  FaCloudUploadAlt,
  FaCloudDownloadAlt,
  FaCheckCircle,
} from "react-icons/fa";

import { BaseModal } from "../common/BaseModal";
import { useTimeFormat } from "@renderer/hooks/useTimeFormat";
import type { SyncStatus, SyncMetaSnapshot } from "src/wailsBridge";

type SyncStatusModalProps = {
  isOpen: boolean;
  onClose: () => void;
  gameTitle: string;
  /** 競合以外の同期状態 */
  status: SyncStatus;
  localMeta?: SyncMetaSnapshot;
  remoteMeta?: SyncMetaSnapshot;
  /** セーブ保存先が設定されているか（未同期時のアップロード可否判定に使う） */
  hasSaveFolder: boolean;
  /** アップロード/ダウンロード実行中か */
  isProcessing: boolean;
  onUpload: () => void;
  onDownload: () => void;
};

type StatusView = {
  icon: React.ReactNode;
  title: string;
  description: string;
};

export default function SyncStatusModal({
  isOpen,
  onClose,
  gameTitle,
  status,
  localMeta,
  remoteMeta,
  hasSaveFolder,
  isProcessing,
  onUpload,
  onDownload,
}: SyncStatusModalProps): React.JSX.Element {
  const { formatDateWithTime } = useTimeFormat();

  const views: Record<Exclude<SyncStatus, "conflict">, StatusView> = {
    in_sync: {
      icon: <FaCheckCircle className="text-success" />,
      title: "最新の状態です",
      description: "ローカルとクラウドのセーブデータは同期されています。",
    },
    push_needed: {
      icon: <FaCloudUploadAlt className="text-primary" />,
      title: "アップロードできます",
      description: "ローカルのセーブデータがクラウドより新しい状態です。",
    },
    pull_needed: {
      icon: <FaCloudDownloadAlt className="text-primary" />,
      title: "ダウンロードできます",
      description: "クラウドのセーブデータがローカルより新しい状態です。",
    },
    never_synced: {
      icon: <FaCloud className="text-base-content/50" />,
      title: "クラウドにデータがありません",
      description: hasSaveFolder
        ? "まだアップロードされていません。アップロードするとクラウドに保存されます。"
        : "まだアップロードされていません。セーブ保存先を設定するとアップロードできます。",
    },
  };

  const view = views[status as Exclude<SyncStatus, "conflict">] ?? views.in_sync;

  const showUpload = status === "push_needed" || (status === "never_synced" && hasSaveFolder);
  const showDownload = status === "pull_needed";

  return (
    <BaseModal
      id="sync-status-modal"
      isOpen={isOpen}
      onClose={onClose}
      title="同期状態の確認"
      size="md"
      footer={
        <div className="flex flex-wrap justify-end gap-2">
          <button type="button" className="btn" onClick={onClose} disabled={isProcessing}>
            閉じる
          </button>
          {showUpload && (
            <button
              type="button"
              className="btn btn-primary"
              onClick={onUpload}
              disabled={isProcessing}
            >
              {isProcessing ? (
                <span className="loading loading-spinner loading-xs" />
              ) : (
                <FaCloudUploadAlt />
              )}
              アップロード
            </button>
          )}
          {showDownload && (
            <button
              type="button"
              className="btn btn-primary"
              onClick={onDownload}
              disabled={isProcessing}
            >
              {isProcessing ? (
                <span className="loading loading-spinner loading-xs" />
              ) : (
                <FaCloudDownloadAlt />
              )}
              ダウンロード
            </button>
          )}
        </div>
      }
    >
      <div className="space-y-4">
        {/* 状態サマリ */}
        <div className="flex items-start gap-3 rounded-lg border border-base-300 bg-base-200 p-3">
          <span className="text-2xl">{view.icon}</span>
          <div>
            <p className="font-semibold">{view.title}</p>
            <p className="text-sm text-base-content/75">{view.description}</p>
          </div>
        </div>

        {/* ローカル / クラウドの情報 */}
        <div className="grid grid-cols-2 gap-3">
          <div className="rounded-lg border border-base-300 bg-base-100 p-3 space-y-2">
            <div className="flex items-center gap-2 font-medium text-sm">
              <FaDesktop className="text-base-content/70" />
              ローカル
            </div>
            {localMeta ? (
              <dl className="text-xs text-base-content/75 space-y-1">
                <div>
                  <dt className="inline">デバイス: </dt>
                  <dd className="inline font-medium text-base-content">{localMeta.deviceName}</dd>
                </div>
                <div>
                  <dt className="inline">更新日時: </dt>
                  <dd className="inline font-medium text-base-content">
                    {formatDateWithTime(localMeta.createdAt)}
                  </dd>
                </div>
              </dl>
            ) : (
              <p className="text-xs text-base-content/60">情報なし</p>
            )}
          </div>

          <div className="rounded-lg border border-primary/30 bg-base-100 p-3 space-y-2">
            <div className="flex items-center gap-2 font-medium text-sm">
              <FaCloud className="text-primary" />
              クラウド
            </div>
            {remoteMeta ? (
              <dl className="text-xs text-base-content/75 space-y-1">
                <div>
                  <dt className="inline">デバイス: </dt>
                  <dd className="inline font-medium text-base-content">{remoteMeta.deviceName}</dd>
                </div>
                <div>
                  <dt className="inline">更新日時: </dt>
                  <dd className="inline font-medium text-base-content">
                    {formatDateWithTime(remoteMeta.createdAt)}
                  </dd>
                </div>
              </dl>
            ) : (
              <p className="text-xs text-base-content/60">情報なし</p>
            )}
          </div>
        </div>

        <p className="text-xs text-base-content/60">対象: {gameTitle}</p>
      </div>
    </BaseModal>
  );
}
