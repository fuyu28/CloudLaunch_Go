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
  saveFolderPath?: string;
  exePath: string;
  imagePath?: string;
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

export type MonitoringGameStatus = {
  gameId: string;
  gameTitle: string;
  exeName: string;
  isPlaying: boolean;
  playTime: number;
  isPaused: boolean;
  needsConfirmation: boolean;
  needsResume: boolean;
};

export type PlaySessionType = {
  id: string;
  sessionName?: string;
  playedAt: Date;
  duration: number;
  gameId: string;
};
