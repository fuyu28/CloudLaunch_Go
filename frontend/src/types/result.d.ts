/**
 * @fileoverview API 結果型定義
 *
 * フロントが扱う ApiResult 成功／失敗ユニオン型。
 */

export type ApiResult<T = void> = { success: true; data?: T } | { success: false; message: string };
