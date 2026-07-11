/**
 * @fileoverview セーブデータのクラウド同期確認メッセージ用ヘルパ。
 * Home / GameDetail 双方で重複していた確認ダイアログ文言の組み立てを共通化する。
 */

export function toValidDate(value: Date | string | number | null | undefined): Date | null {
  if (!value) return null;
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? null : parsed;
}

export function buildSaveSyncMessage(
  formatDateWithTime: (date: Date | string | number | null | undefined) => string,
  title: string,
  localUpdatedAt: Date | string | number | null | undefined,
  cloudUpdatedAt: Date | string | number | null | undefined,
): string {
  const localDate = toValidDate(localUpdatedAt);
  const cloudDate = toValidDate(cloudUpdatedAt);
  return `${title} のセーブデータがクラウドと異なります。\nローカル最終更新: ${formatDateWithTime(
    localDate,
  )}\nクラウド最終更新: ${formatDateWithTime(cloudDate)}\nダウンロードしますか？`;
}
