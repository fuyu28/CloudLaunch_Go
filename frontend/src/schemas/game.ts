/**
 * @fileoverview ゲーム Zod スキーマ
 *
 * ゲーム登録・編集フォームのバリデーション定義。
 */

import { z } from "zod";

export const gameSchema = z.object({
  title: z.string().min(1, "タイトルは必須です").max(100, "100文字以内で入力してください"),
  publisher: z.string().min(1, "ブランド名は必須です").max(50, "50文字以内で入力してください"),
  imagePath: z.string().optional().or(z.literal("")),
  exePath: z.string().min(1, "実行ファイルのパスは必須です"),
  saveFolderPath: z.string().optional().or(z.literal("")),
});

export const gameFormSchema = gameSchema
  .refine(
    (data) => {
      if (
        data.exePath &&
        ![".exe", ".app"].some((ext) => data.exePath.toLowerCase().endsWith(ext))
      ) {
        return false;
      }
      return true;
    },
    {
      message: "実行ファイル（.exe または .app）を指定してください",
      path: ["exePath"],
    },
  )
  .refine(
    (data) => {
      if (data.imagePath && data.imagePath.trim()) {
        try {
          new URL(data.imagePath);
          const imageExtensions = [".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp"];
          const url = new URL(data.imagePath);
          const pathname = url.pathname.toLowerCase();
          return imageExtensions.some((ext) => pathname.endsWith(ext));
        } catch {
          const imageExtensions = [".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp"];
          if (!imageExtensions.some((ext) => data.imagePath!.toLowerCase().endsWith(ext))) {
            return false;
          }
        }
      }
      return true;
    },
    {
      message: "画像ファイル（PNG、JPG、GIF等）または画像URLを指定してください",
      path: ["imagePath"],
    },
  );

export const monitoringGameStatusSchema = z.object({
  gameId: z.uuid("有効なUUIDを指定してください"),
  gameTitle: z.string().min(1, "ゲームタイトルは必須です"),
  exeName: z.string().min(1, "実行ファイル名は必須です"),
  isPlaying: z.boolean(),
  playTime: z.number().min(0, "プレイ時間は0以上である必要があります"),
  isPaused: z.boolean(),
  needsConfirmation: z.boolean(),
  needsResume: z.boolean(),
});

export type MonitoringGameStatus = z.infer<typeof monitoringGameStatusSchema>;
