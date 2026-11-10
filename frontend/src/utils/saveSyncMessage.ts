/**
 * @fileoverview 起動前のセーブデータのダウンロード確認メッセージを組み立てるヘルパ。
 * Home / GameDetail で共通化し、同期判定（Status）が返した実際のフォルダ・コミット情報だけを載せる。
 */

import type { SyncStatusDetail } from "src/wailsBridge";

type DateFormatter = (date: Date | string | number | null | undefined) => string;

export function buildSaveSyncMessage(
  formatDateWithTime: DateFormatter,
  title: string,
  detail: Pick<SyncStatusDetail, "hasLocalBaseline" | "local" | "remoteMeta">,
): string {
  // 基準なしの pull_needed は「クラウドで変更あり」ではないので、同じ文言にしない。
  const lead = detail.hasLocalBaseline
    ? `${title} は前回の同期以降、別のPCからセーブデータがアップロードされています。`
    : `${title} はこのPCでまだ同期しておらず、セーブデータがクラウドと異なります。`;
  const localLine = detail.local?.lastModified
    ? formatDateWithTime(detail.local.lastModified)
    : "ファイルなし";
  const remoteLine = detail.remoteMeta
    ? `${formatDateWithTime(detail.remoteMeta.createdAt)}（${detail.remoteMeta.deviceName}）`
    : "不明";
  return [
    lead,
    `このPCのセーブ最終更新: ${localLine}`,
    `クラウドへのアップロード: ${remoteLine}`,
    "ダウンロードすると、このPCのセーブデータはクラウドの内容に置き換わります。ダウンロードしますか？",
  ].join("\n");
}
