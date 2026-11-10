/**
 * @fileoverview 同期状態の確認モーダル
 *
 * 「同期確認」押下時に、いきなりアップロード/ダウンロードするのではなく、
 * 前回の同期以降どちら側で何が変わったか（バックエンドの3者比較そのまま）と、
 * 判定に使ったセーブフォルダ・クラウドの情報を表示し、必要な操作だけをユーザーに選ばせる。
 */

import { useState } from "react";

import {
  FaCloud,
  FaCloudUploadAlt,
  FaCloudDownloadAlt,
  FaCheckCircle,
  FaExclamationTriangle,
} from "react-icons/fa";

import { BaseModal } from "../common/BaseModal";
import { SyncMetaCardPair } from "./SyncMetaCardPair";
import type { SyncStatusDetail } from "src/wailsBridge";

type SyncStatusModalProps = {
  isOpen: boolean;
  onClose: () => void;
  gameTitle: string;
  detail: SyncStatusDetail;
  hasSaveFolder: boolean;
  isProcessing: boolean;
  onUpload: () => void;
  onDownload: () => void;
  onForceUpload: () => void;
  onForceDownload: () => void;
};

type StatusView = {
  icon: React.ReactNode;
  title: string;
  description: string;
};

// 「新しい／古い」は使わない。判定は時刻ではなく「前回の同期から変わったか」なので、
// 日時比較の言葉を出すと表示中の日時と矛盾して見える。
function describeStatus(detail: SyncStatusDetail, hasSaveFolder: boolean): StatusView {
  const { status, savesDiffer, hasLocalBaseline } = detail;
  switch (status) {
    case "in_sync":
    case "conflict":
      return {
        icon: <FaCheckCircle className="text-success" />,
        title: "クラウドと一致しています",
        description: "前回の同期以降、このPCでもクラウドでも変更はありません。",
      };
    case "push_needed":
      return savesDiffer
        ? {
            icon: <FaCloudUploadAlt className="text-primary" />,
            title: "このPCに未アップロードの変更があります",
            description:
              "前回の同期以降、このPCのセーブデータが変更されています。クラウド側は変わっていません。",
          }
        : {
            icon: <FaCloudUploadAlt className="text-primary" />,
            title: "プレイ記録だけが未アップロードです",
            description:
              "セーブデータはクラウドと同じです。ゲーム情報やプレイ履歴だけがこのPCで変更されています。",
          };
    case "pull_needed":
      if (!hasLocalBaseline) {
        return savesDiffer
          ? {
              icon: <FaExclamationTriangle className="text-warning" />,
              title: "このPCではまだ同期していません",
              description:
                "このPCのセーブデータはクラウドと異なります。ダウンロードするとクラウドの内容に置き換わります。このPCの内容を残すなら強制同期でクラウドを上書きしてください。",
            }
          : {
              icon: <FaCloudDownloadAlt className="text-primary" />,
              title: "このPCではまだ同期していません",
              description:
                "セーブデータはクラウドと同じです。ダウンロードすると、ゲーム情報とプレイ履歴をクラウドに合わせます。",
            };
      }
      return savesDiffer
        ? {
            icon: <FaCloudDownloadAlt className="text-primary" />,
            title: "クラウドに未取得の変更があります",
            description:
              "前回の同期以降、別のPCからアップロードされています。このPCのセーブデータは変わっていません。",
          }
        : {
            icon: <FaCloudDownloadAlt className="text-primary" />,
            title: "クラウドのプレイ記録だけが更新されています",
            description:
              "セーブデータは同じです。別のPCで更新されたゲーム情報やプレイ履歴を取り込めます。",
          };
    case "never_synced":
      return {
        icon: <FaCloud className="text-base-content/50" />,
        title: "クラウドにデータがありません",
        description: hasSaveFolder
          ? "まだアップロードされていません。アップロードするとクラウドに保存されます。"
          : "まだアップロードされていません。セーブ保存先を設定するとアップロードできます。",
      };
  }
}

export default function SyncStatusModal({
  isOpen,
  onClose,
  gameTitle,
  detail,
  hasSaveFolder,
  isProcessing,
  onUpload,
  onDownload,
  onForceUpload,
  onForceDownload,
}: SyncStatusModalProps): React.JSX.Element {
  const [forceAction, setForceAction] = useState<"push" | "pull" | null>(null);
  const close = (): void => {
    if (isProcessing) return;
    setForceAction(null);
    onClose();
  };

  // 競合は SyncConflictModal 側。ここに載せると上書き操作と混ざる。
  const { status } = detail;
  const view = describeStatus(detail, hasSaveFolder);

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
      onClose={close}
      title="同期状態の確認"
      size="md"
      footer={
        <div className="flex flex-wrap justify-end gap-2">
          {forceAction ? (
            <>
              <button
                type="button"
                className="btn"
                disabled={isProcessing}
                onClick={() => setForceAction(null)}
              >
                戻る
              </button>
              <button
                type="button"
                className="btn btn-error"
                disabled={isProcessing}
                onClick={() => {
                  const action = forceAction;
                  setForceAction(null);
                  if (action === "push") onForceUpload();
                  else onForceDownload();
                }}
              >
                上書きする
              </button>
            </>
          ) : (
            <>
              <button type="button" className="btn" onClick={close} disabled={isProcessing}>
                閉じる
              </button>
              {showUpload && actionButton(<FaCloudUploadAlt />, "アップロード", onUpload)}
              {showDownload && actionButton(<FaCloudDownloadAlt />, "ダウンロード", onDownload)}
            </>
          )}
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

        <SyncMetaCardPair local={detail.local} remoteMeta={detail.remoteMeta} />

        {forceAction ? (
          <div role="alert" className="rounded-lg border border-error p-3 text-sm">
            {forceAction === "push"
              ? "クラウドのセーブデータ・ゲーム情報・プレイ履歴を、このPCの内容で上書きします。クラウド側の変更は失われます。"
              : "このPCのセーブデータ・ゲーム情報・プレイ履歴を、クラウドの内容で上書きします。このPC側の変更は失われます。"}
          </div>
        ) : (
          <details>
            <summary className="cursor-pointer text-sm">強制同期</summary>
            <div className="mt-2 flex flex-wrap gap-2">
              <button
                type="button"
                className="btn btn-outline btn-error btn-sm"
                disabled={isProcessing || !hasSaveFolder}
                onClick={() => setForceAction("push")}
              >
                Force Push（クラウドを上書き）
              </button>
              <button
                type="button"
                className="btn btn-outline btn-error btn-sm"
                disabled={isProcessing || !hasSaveFolder || status === "never_synced"}
                onClick={() => setForceAction("pull")}
              >
                Force Pull（このPCを上書き）
              </button>
            </div>
          </details>
        )}

        <p className="text-xs text-base-content/60">対象: {gameTitle}</p>
      </div>
    </BaseModal>
  );
}
