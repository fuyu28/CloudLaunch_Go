/**
 * @fileoverview ローカル / クラウドのセーブメタ情報を並べて表示するカード対
 *
 * 同期状態の確認（SyncStatusModal）と競合解決（SyncConflictModal）の双方で使う、
 * デバイス名・更新日時の比較表示を共通化する。
 */

import { FaDesktop, FaCloud } from "react-icons/fa";

import { useTimeFormat } from "@renderer/hooks/useTimeFormat";
import type { SyncMetaSnapshot } from "src/wailsBridge";

type SyncMetaCardPairProps = {
  localMeta: SyncMetaSnapshot | undefined;
  remoteMeta: SyncMetaSnapshot | undefined;
  localIconClassName?: string;
};

export function SyncMetaCardPair({
  localMeta,
  remoteMeta,
  localIconClassName = "text-base-content/70",
}: SyncMetaCardPairProps): React.JSX.Element {
  const { formatDateWithTime } = useTimeFormat();

  const renderCard = (
    icon: React.ReactNode,
    label: string,
    meta: SyncMetaSnapshot | undefined,
    borderClassName: string,
  ): React.JSX.Element => (
    <div className={`rounded-lg border ${borderClassName} bg-base-100 p-3 space-y-2`}>
      <div className="flex items-center gap-2 font-medium text-sm">
        {icon}
        {label}
      </div>
      {meta ? (
        <dl className="text-xs text-base-content/70 space-y-1">
          <div>
            <dt className="inline">デバイス: </dt>
            <dd className="inline font-medium text-base-content">{meta.deviceName}</dd>
          </div>
          <div>
            <dt className="inline">更新日時: </dt>
            <dd className="inline font-medium text-base-content">
              {formatDateWithTime(meta.createdAt)}
            </dd>
          </div>
        </dl>
      ) : (
        <p className="text-xs text-base-content/60">情報なし</p>
      )}
    </div>
  );

  return (
    <div className="grid grid-cols-2 gap-3">
      {renderCard(
        <FaDesktop className={localIconClassName} />,
        "ローカル",
        localMeta,
        "border-base-300",
      )}
      {renderCard(
        <FaCloud className="text-primary" />,
        "クラウド",
        remoteMeta,
        "border-primary/30",
      )}
    </div>
  );
}
