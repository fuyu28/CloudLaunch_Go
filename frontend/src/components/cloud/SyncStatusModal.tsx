/**
 * @fileoverview 同期状態の確認モーダル
 *
 * 「同期確認」押下時に、いきなりアップロード/ダウンロードするのではなく、
 * 現在の同期状態（最新 / アップロード可 / ダウンロード可 / 未同期）と
 */

import { FaCloud, FaCloudUploadAlt, FaCloudDownloadAlt, FaCheckCircle } from "react-icons/fa";

import { BaseModal } from "../common/BaseModal";
import { SyncMetaCardPair } from "./SyncMetaCardPair";
import type { SyncStatus, SyncMetaSnapshot } from "src/wailsBridge";

type SyncStatusModalProps = {
  isOpen: boolean;
  onClose: () => void;
  gameTitle: string;
  status: SyncStatus;
  localMeta?: SyncMetaSnapshot;
  remoteMeta?: SyncMetaSnapshot;
  hasSaveFolder: boolean;
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

  // 競合は SyncConflictModal 側。ここに載せると上書き操作と混ざる。
  const view = status === "conflict" ? views.in_sync : views[status];

  const showUpload = status === "push_needed" || (status === "never_synced" && hasSaveFolder);
  const showDownload = status === "pull_needed";

  const actionButton = (
    icon: React.ReactNode,
    label: string,
    onClick: () => void,
  ): React.JSX.Element => (
    <button type="button" className="btn btn-primary" onClick={onClick} disabled={isProcessing}>
      {isProcessing ? <span className="loading loading-spinner loading-xs" /> : icon}
      {label}
    </button>
  );

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
          {showUpload && actionButton(<FaCloudUploadAlt />, "アップロード", onUpload)}
          {showDownload && actionButton(<FaCloudDownloadAlt />, "ダウンロード", onDownload)}
        </div>
      }
    >
      <div className="space-y-4">
        <div className="flex items-start gap-3 rounded-lg border border-base-300 bg-base-200 p-3">
          <span className="text-2xl">{view.icon}</span>
          <div>
            <p className="font-semibold">{view.title}</p>
            <p className="text-sm text-base-content/75">{view.description}</p>
          </div>
        </div>

        <SyncMetaCardPair localMeta={localMeta} remoteMeta={remoteMeta} />

        <p className="text-xs text-base-content/60">対象: {gameTitle}</p>
      </div>
    </BaseModal>
  );
}
