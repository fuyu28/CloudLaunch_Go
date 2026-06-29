/**
 * @fileoverview セーブデータのクラウド同期確認メッセージ用ヘルパ。
 *
 * Home / GameDetail 双方で繰り返されていた
 *   - `toValidDate`: 任意の値を有効な Date / null に正規化
 *   - `buildSaveSyncMessage`: ローカル/クラウドの更新日時を併記したメッセージ生成
 * を集約する。`formatDateWithTime` はページ側で `useTimeFormat()` から受け取って渡す。
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
