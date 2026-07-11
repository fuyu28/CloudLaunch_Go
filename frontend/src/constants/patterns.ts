/**
 * @fileoverview 正規表現パターン定数定義
 *
 * このファイルは、アプリケーション全体で使用される正規表現パターンを定数として定義します。
 */

export const PATTERNS = {
  BUCKET_NAME: /^[a-z0-9][a-z0-9.-]*[a-z0-9]$/,

  URL_VALIDATION:
    /^(https?:\/\/)?(?:localhost|(?:[a-zA-Z0-9-]+\.)+[a-zA-Z]{2,6})(?::[0-9]{1,5})?(?:\/[^\s]*)?$/,

  IMAGE_FILE_EXTENSIONS: /\.(jpg|jpeg|png|gif|bmp|webp)$/i,
  EXE_FILE_EXTENSIONS: /\.(exe|msi)$/i,

  /** Steam URLパターン（steam:\/\/rungameid\/123456 形式） */
  STEAM_URL: /^steam:\/\/rungameid\/([0-9]+)$/,

  INVALID_FILENAME_CHARS: /[<>:"/\\|?*]/g,
} as const;

export type Patterns = typeof PATTERNS;
