/**
 * @fileoverview 確認ダイアログコンポーネント
 *
 * このコンポーネントは、ユーザーのアクション確認を行うためのモーダルダイアログを提供します。
 * シンプルなメッセージ表示から、詳細情報・注意事項付きの高度な確認まで対応します。
 */

import { FiAlertTriangle } from "react-icons/fi";

import { BaseModal } from "./BaseModal";

export type WarningItem = {
  text: string;
  highlight?: boolean;
};

export type ConfirmDetails = {
  icon?: React.ReactNode;
  subText?: string;
  warnings?: WarningItem[];
  customContent?: React.ReactNode;
};

type ConfirmModalProps = {
  id: string;
  isOpen: boolean;
  title?: string;
  message: string;
  details?: ConfirmDetails;
  cancelText?: string;
  confirmText?: string;
  confirmVariant?: "primary" | "error";
  onConfirm: () => void;
  onCancel: () => void;
};

export default function ConfirmModal({
  id,
  isOpen,
  title = "確認",
  message,
  details,
  cancelText = "いいえ",
  confirmText = "はい",
  confirmVariant = "primary",
  onConfirm,
  onCancel,
}: ConfirmModalProps): React.JSX.Element {
  const footer = (
    <div className="justify-end space-x-2">
      <button className="btn btn-outline" onClick={onCancel}>
        {cancelText}
      </button>
      <button
        className={`btn ${confirmVariant === "error" ? "btn-error" : "btn-primary"}`}
        onClick={onConfirm}
      >
        {confirmText}
      </button>
    </div>
  );

  return (
    <BaseModal
      id={id}
      isOpen={isOpen}
      title={title}
      onClose={onCancel}
      size="lg"
      showCloseButton={false}
      footer={footer}
    >
      <div className="mb-4">
        {details?.icon ? (
          <div className="flex items-center gap-3">
            <div className="text-4xl flex-shrink-0">{details.icon}</div>
            <div>
              <p className="mb-2 whitespace-pre-line">{message}</p>
              {details.subText && <p className="text-sm text-base-content/70">{details.subText}</p>}
            </div>
          </div>
        ) : (
          <p className="mb-2 whitespace-pre-line">{message}</p>
        )}
      </div>

      {details?.customContent && <div className="mb-4">{details.customContent}</div>}

      {details?.warnings && details.warnings.length > 0 && (
        <div className="bg-error/10 border border-error/20 rounded-lg p-3 text-sm">
          <div className="flex items-start gap-2">
            <FiAlertTriangle className="text-error mt-0.5 flex-shrink-0" />
            <div>
              <div className="font-medium text-error mb-1">注意事項</div>
              <ul className="text-error/80 space-y-1 text-xs">
                {details.warnings.map((warning, index) => (
                  <li key={index} className={warning.highlight ? "font-medium" : ""}>
                    • {warning.text}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      )}
    </BaseModal>
  );
}
