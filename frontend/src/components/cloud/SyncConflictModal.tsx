import { FaDesktop, FaCloud } from "react-icons/fa";

import { BaseModal } from "../common/BaseModal";
import { SyncMetaCardPair } from "./SyncMetaCardPair";
import type { SyncMetaSnapshot } from "src/wailsBridge";

type SyncConflictModalProps = {
  isOpen: boolean;
  onClose: () => void;
  gameTitle: string;
  localMeta: SyncMetaSnapshot | undefined;
  remoteMeta: SyncMetaSnapshot | undefined;
  onUseLocal: () => void;
  onUseRemote: () => void;
  isResolving: boolean;
};

export default function SyncConflictModal({
  isOpen,
  onClose,
  gameTitle,
  localMeta,
  remoteMeta,
  onUseLocal,
  onUseRemote,
  isResolving,
}: SyncConflictModalProps): React.JSX.Element {
  return (
    <BaseModal
      id="sync-conflict-modal"
      isOpen={isOpen}
      onClose={onClose}
      title="セーブデータが競合しています"
      size="md"
      footer={
        <div className="flex flex-wrap justify-end gap-2">
          <button type="button" className="btn" onClick={onClose} disabled={isResolving}>
            キャンセル
          </button>
          <button
            type="button"
            className="btn btn-outline btn-warning"
            onClick={onUseLocal}
            disabled={isResolving}
          >
            {isResolving ? <span className="loading loading-spinner loading-xs" /> : <FaDesktop />}
            ローカルを使う
          </button>
          <button
            type="button"
            className="btn btn-primary"
            onClick={onUseRemote}
            disabled={isResolving}
          >
            {isResolving ? <span className="loading loading-spinner loading-xs" /> : <FaCloud />}
            クラウドを使う
          </button>
        </div>
      }
    >
      <div className="space-y-4">
        <p className="text-sm text-base-content/70">
          「{gameTitle}」のセーブデータがローカルとクラウドの両方で変更されています。
          どちらのデータを使用するか選択してください。
        </p>

        <SyncMetaCardPair
          localMeta={localMeta}
          remoteMeta={remoteMeta}
          localIconClassName="text-warning"
        />

        <div className="alert alert-warning py-2">
          <span className="text-xs">
            選択しなかった側のデータは上書きされます。この操作は取り消せません。
          </span>
        </div>
      </div>
    </BaseModal>
  );
}
