/**
 * @fileoverview プレイセッション Zod スキーマ
 *
 * セッション名編集などのバリデーション定義。
 */

import { z } from "zod";

export const playSessionEditSchema = z.object({
  // 空文字は「セッション名をクリアして未設定に戻す」意味を持つため許可する（バックエンドで NULL 化される）。
  sessionName: z.string().max(200, "セッション名は200文字以内で入力してください").trim(),
});
