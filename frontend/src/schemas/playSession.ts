import { z } from "zod";

/**
 * プレイセッション編集用のバリデーションスキーマ
 * セッション名の編集データ検証に使用
 */
export const playSessionEditSchema = z.object({
  sessionName: z
    .string()
    .min(1, "セッション名は必須です")
    .max(200, "セッション名は200文字以内で入力してください")
    .trim(),
});
