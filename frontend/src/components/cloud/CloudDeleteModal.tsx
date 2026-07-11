/**
 * @fileoverview クラウドデータ削除確認モーダル
 *
 * このコンポーネントは、クラウドデータの削除確認ダイアログを
 * 表示し、削除に関する詳細情報を提供します。
 */

import { FiAlertTriangle } from "react-icons/fi";

import { useCloudDeleteConfirm } from "@renderer/hooks/useCloudDeleteConfirm";

import ConfirmModal from "@renderer/components/common/ConfirmModal";
import type { ConfirmDetails, WarningItem } from "@renderer/components/common/ConfirmModal";
import type { CloudDataItem, CloudDirectoryNode } from "src/types/cloud";

type CloudDeleteModalProps = {
  deleteConfirm: CloudDataItem | CloudDirectoryNode | null;
  onCancel: () => void;
  onConfirm: (item: CloudDataItem | CloudDirectoryNode) => void;
  cloudData: CloudDataItem[];
};

export function CloudDeleteModal({
  deleteConfirm,
  onCancel,
  onConfirm,
  cloudData,
}: CloudDeleteModalProps): React.JSX.Element {
  const { deleteMessage, fileCountMessage, subText, sizeMessage, additionalWarnings } =
    useCloudDeleteConfirm(deleteConfirm, cloudData);

  return (
    <ConfirmModal
      id="delete-cloud-data-modal"
      isOpen={!!deleteConfirm}
      onCancel={onCancel}
      onConfirm={() => deleteConfirm && onConfirm(deleteConfirm)}
      title="クラウドデータの削除"
      message={deleteMessage}
      confirmText="削除"
      cancelText="キャンセル"
      confirmVariant="error"
      details={
        {
          icon: <FiAlertTriangle className="text-error" />,
          subText,
          warnings: [
            { text: "削除されたデータは復元できません" },
            {
              text: fileCountMessage,
              highlight: true,
            },
            {
              text: sizeMessage,
            },
            ...additionalWarnings,
          ] as WarningItem[],
        } as ConfirmDetails
      }
    />
  );
}
