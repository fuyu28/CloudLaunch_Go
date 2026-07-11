/**
 * @fileoverview 設定値定数定義
 *
 * このファイルは、アプリケーション全体で使用される設定値を定数として定義します。
 */

export const CONFIG = {
  // タイミング関連（ミリ秒）
  TIMING: {
    SEARCH_DEBOUNCE_MS: 300,
  },

  // バリデーション関連
  VALIDATION: {
    ACCESS_KEY_MIN_LENGTH: 10,
    SECRET_KEY_MIN_LENGTH: 20,
    TITLE_MAX_LENGTH: 100,
    PUBLISHER_MAX_LENGTH: 100,
  },

  // デフォルト値
  DEFAULTS: {
    REGION: "auto",
    PLAY_STATUS: "unplayed" as const,
  },

  // UI関連
  UI: {
    CARD_WIDTH: "220px",
    FLOATING_BUTTON_POSITION: "bottom-16 right-6",
    ICON_SIZE: 28,
  },

  // ファイル関連
  FILE: {
    IMAGE_EXTENSIONS: ["png", "jpg", "jpeg", "gif"] as const,
    EXECUTABLE_EXTENSIONS: ["exe", "app"] as const,
  },

  // ファイルサイズ関連
  FILE_SIZE: {
    MAX_UPLOAD_SIZE_MB: 100,
    MAX_IMAGE_SIZE_MB: 10,
  },

  // AWS S3/R2関連
  AWS: {
    DEFAULT_REGION: "auto",
    DELETE_BATCH_SIZE: 1000,
    REQUEST_TIMEOUT_MS: 30000,
    MAX_LIST_ITERATIONS: 1000,
  },

  // Steam関連
  STEAM: {
    APPLAUNCH_FLAG: "-applaunch",
    NO_VR_FLAG: "--no-vr",
  },

  // Prisma関連
  PRISMA: {
    UNIQUE_CONSTRAINT_ERROR: "P2002",
  },

  // パス関連
  PATH: {
    REMOTE_PATH_TEMPLATE: (gameId: string) => `games/${gameId}/save_data`,
  },
} as const;

export type Config = typeof CONFIG;
