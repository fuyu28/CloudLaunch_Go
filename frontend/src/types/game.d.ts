/**
 * @fileoverview ゲーム関連型定義
 *
 * GameType・PlayStatus・セッションなどゲームドメインの TypeScript 型。
 */

export type PlayStatus = "unplayed" | "playing" | "played";

export type GameType = {
  id: string;
  title: string;
  publisher: string;
  saveFolderPath?: string; // undefined - オプショナル設定
  exePath: string;
  imagePath?: string; // undefined - オプショナル設定
  createdAt: Date;
  localSaveHash?: string;
  localSaveHashUpdatedAt?: Date | null;
  playStatus: PlayStatus;
  totalPlayTime: number;
  lastPlayed: Date | null; // null - 明確な「未プレイ」状態
  clearedAt: Date | null; // null - 明確な「未クリア」状態
  currentRouteId: string | null; // null - 明確な「未選択」状態
};

export type InputGameData = {
  title: string;
  publisher: string;
  imagePath?: string;
  exePath: string;
  saveFolderPath?: string;
};

export type GameImport = {
  erogameScapeId: string;
  title: string;
  brand: string;
  imagePath: string;
  imageUrl?: string;
};

/**
 * 監視中のゲーム情報
 */
export type MonitoringGameStatus = {
  gameId: string;
  gameTitle: string;
  exeName: string;
  isPlaying: boolean;
  playTime: number;
  isPaused: boolean;
  /** 終了確認が必要かどうか */
  needsConfirmation: boolean;
  /** 再開確認が必要かどうか */
  needsResume: boolean;
};

export type PlaySessionType = {
  id: string;
  sessionName?: string; // undefined - オプショナル情報
  playedAt: Date;
  duration: number;
  gameId: string;
};
