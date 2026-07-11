/**
 * @fileoverview 同期管理外ファイルの削除確認モーダル
 *
 * Pull 時に見つかった untracked ファイル削除の可否をユーザーに確認する。
 */

import { FaTrash, FaExclamationTriangle } from "react-icons/fa";

import { BaseModal } from "../common/BaseModal";

type UntrackedDeleteModalProps = {
  isOpen: boolean;
  onClose: () => void;
  gameTitle: string;
  files: string[];
  onConfirm: () => void;
  isProcessing: boolean;
};

/**
 * Pull 時に「同期が一度も認識していないローカル固有ファイル（untracked）」を
 * 削除する必要がある場合に、ユーザーへ明示的な確認を取るモーダル。
 *
 * saveFolderPath の誤設定などで無関係なファイルが巻き込まれて消えるのを防ぐため、
 * これらのファイルは確認なしには削除されない。
 */
export default function UntrackedDeleteModal({
  isOpen,
  onClose,
  gameTitle,
  files,
  onConfirm,
  isProcessing,
}: UntrackedDeleteModalProps): React.JSX.Element {
  return (
    <BaseModal
      id="untracked-delete-modal"
      isOpen={isOpen}
      onClose={onClose}
      title="同期対象外のファイルを削除しますか？"
      size="md"
      footer={
        <div className="flex flex-wrap justify-end gap-2">
          <button type="button" className="btn" onClick={onClose} disabled={isProcessing}>
            キャンセル
          </button>
          <button
            type="button"
            className="btn btn-error"
            onClick={onConfirm}
            disabled={isProcessing}
          >
            {isProcessing ? <span className="loading loading-spinner loading-xs" /> : <FaTrash />}
            削除してダウンロード
          </button>
        </div>
      }
    >
      <div className="space-y-3">
        <p className="text-sm text-base-content/70">
          「{gameTitle}」をクラウドから取得すると、セーブフォルダ内の次のファイルが
          <span className="font-medium text-base-content">クラウド側に存在しない</span>ため
          削除されます。これらは同期がこれまで一度も管理していないファイルです。
        </p>

        <div className="alert alert-warning py-2">
          <FaExclamationTriangle />
          <span className="text-xs">
            セーブフォルダの設定が意図と違う場合、無関係なファイルが含まれていないか確認してください。
            この操作は取り消せません。
          </span>
        </div>

        <ul className="max-h-48 overflow-y-auto rounded-lg border border-base-300 bg-base-100 p-2 text-xs font-mono space-y-1">
          {files.map((file) => (
            <li key={file} className="truncate" title={file}>
              {file}
            </li>
          ))}
        </ul>

        <p className="text-xs text-base-content/50">削除対象: {files.length} 件</p>
      </div>
    </BaseModal>
  );
}
