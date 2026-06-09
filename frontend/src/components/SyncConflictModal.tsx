import { FaDesktop, FaCloud } from "react-icons/fa";

import { BaseModal } from "./BaseModal";
import { useTimeFormat } from "@renderer/hooks/useTimeFormat";
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
  const { formatDateWithTime } = useTimeFormat();

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

        <div className="grid grid-cols-2 gap-3">
          {/* ローカル */}
          <div className="rounded-lg border border-base-300 bg-base-100 p-3 space-y-2">
            <div className="flex items-center gap-2 font-medium text-sm">
              <FaDesktop className="text-warning" />
              ローカル
            </div>
            {localMeta ? (
              <dl className="text-xs text-base-content/70 space-y-1">
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
              <p className="text-xs text-base-content/50">情報なし</p>
            )}
          </div>

          {/* クラウド */}
          <div className="rounded-lg border border-primary/30 bg-base-100 p-3 space-y-2">
            <div className="flex items-center gap-2 font-medium text-sm">
              <FaCloud className="text-primary" />
              クラウド
            </div>
            {remoteMeta ? (
              <dl className="text-xs text-base-content/70 space-y-1">
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
              <p className="text-xs text-base-content/50">情報なし</p>
            )}
          </div>
        </div>

        <div className="alert alert-warning py-2">
          <span className="text-xs">
            選択しなかった側のデータは上書きされます。この操作は取り消せません。
          </span>
        </div>
      </div>
    </BaseModal>
  );
}
