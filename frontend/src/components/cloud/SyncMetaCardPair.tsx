/**
 * @fileoverview このPCのセーブフォルダとクラウドの最新アップロードを並べて表示するカード対
 *
 * 同期状態の確認（SyncStatusModal）と競合解決（SyncConflictModal）の双方で、
 * 同期判定が実際に読んだフォルダと日時を見せる。
 */

import { FaDesktop, FaCloud } from "react-icons/fa";

import { useTimeFormat } from "@renderer/hooks/useTimeFormat";
import { formatFileSize } from "@renderer/utils/cloudUtils";
import type { LocalSaveInfo, SyncMetaSnapshot } from "src/wailsBridge";

type SyncMetaCardPairProps = {
  local: LocalSaveInfo | undefined;
  remoteMeta: SyncMetaSnapshot | undefined;
  localIconClassName?: string;
};

function Row({ label, value, title }: { label: string; value: string; title?: string }) {
  return (
    <div className="flex gap-1 min-w-0">
      <dt className="shrink-0">{label}:</dt>
      <dd className="font-medium text-base-content truncate" title={title}>
        {value}
      </dd>
    </div>
  );
}

export function SyncMetaCardPair({
  local,
  remoteMeta,
  localIconClassName = "text-base-content/70",
}: SyncMetaCardPairProps): React.JSX.Element {
  const { formatDateWithTime } = useTimeFormat();

  const card = (
    icon: React.ReactNode,
    label: string,
    borderClassName: string,
    body: React.ReactNode,
  ): React.JSX.Element => (
    <div className={`rounded-lg border ${borderClassName} bg-base-100 p-3 space-y-2 min-w-0`}>
      <div className="flex items-center gap-2 font-medium text-sm">
        {icon}
        {label}
      </div>
      {body}
    </div>
  );

  return (
    <div className="grid grid-cols-2 gap-3">
      {card(
        <FaDesktop className={localIconClassName} />,
        "このPC",
        "border-base-300",
        local ? (
          <dl className="text-xs text-base-content/70 space-y-1">
            <Row label="フォルダ" value={local.folderPath} title={local.folderPath} />
            <Row
              label="ファイル"
              value={
                local.fileCount > 0
                  ? `${local.fileCount}件（${formatFileSize(local.totalSize)}）`
                  : "なし"
              }
            />
            <Row
              label="最終更新"
              value={local.lastModified ? formatDateWithTime(local.lastModified) : "—"}
            />
          </dl>
        ) : (
          <p className="text-xs text-base-content/60">セーブフォルダが未設定です</p>
        ),
      )}
      {card(
        <FaCloud className="text-primary" />,
        "クラウド",
        "border-primary/30",
        remoteMeta ? (
          <dl className="text-xs text-base-content/70 space-y-1">
            <Row label="アップロード元" value={remoteMeta.deviceName} />
            {remoteMeta.fileCount > 0 && (
              <Row
                label="ファイル"
                value={`${remoteMeta.fileCount}件（${formatFileSize(remoteMeta.totalSize)}）`}
              />
            )}
            <Row label="アップロード日時" value={formatDateWithTime(remoteMeta.createdAt)} />
          </dl>
        ) : (
          <p className="text-xs text-base-content/60">まだアップロードされていません</p>
        ),
      )}
    </div>
  );
}
