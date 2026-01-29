import { z } from "zod"

/**
 * プレイセッション編集用のバリデーションスキーマ
 * セッション名の編集データ検証に使用
 */
export const playSessionEditSchema = z.object({
  sessionName: z
    .string()
    .min(1, "セッション名は必須です")
    .max(200, "セッション名は200文字以内で入力してください")
    .trim()
})

/**
 * セッション名更新用のバリデーションスキーマ
 */
export const sessionNameUpdateSchema = z.object({
  sessionName: z
    .string()
    .min(1, "セッション名は必須です")
    .max(200, "セッション名は200文字以内で入力してください")
    .trim()
})

/**
 * セッションID検証用スキーマ
 */
export const sessionIdSchema = z.string().uuid("有効なセッションIDを指定してください")

/**
 * プレイセッション編集データの型定義（zodスキーマから自動生成）
 */
export type PlaySessionEditData = z.infer<typeof playSessionEditSchema>

/**
 * セッション名更新データの型定義
 */
export type SessionNameUpdateData = z.infer<typeof sessionNameUpdateSchema>
