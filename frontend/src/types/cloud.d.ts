/**
 * @fileoverview クラウドデータ関連型定義
 *
 * このファイルは、クラウドデータ管理で使用される型を統一的に定義します。
 */

import type { PlayStatus } from "./game";

export type CloudDataItem = {
  name: string;
  totalSize: number;
  fileCount: number;
  lastModified: Date;
  remotePath: string;
};

export type CloudFileDetail = {
  name: string;
  size: number;
  lastModified: Date;
  key: string;
  relativePath: string;
};

export type CloudDirectoryNode = {
  name: string;
  path: string;
  isDirectory: boolean;
  size: number;
  lastModified: Date;
  children?: CloudDirectoryNode[];
  objectKey?: string;
  /**
   * トップレベル（ゲーム）の遅延取得状態でも commit メタから持ち越したファイル数。
   * children を取得済みのノードでは undefined（countFilesRecursively で算出する）。
   */
  fileCount?: number;
};

export type CloudGameMetadata = {
  id: string;
  title: string;
  publisher: string;
  imageHash?: string;
  playStatus: PlayStatus;
  totalPlayTime: number;
  lastPlayed?: Date | string | null;
  clearedAt?: Date | string | null;
  currentRouteId?: string | null;
  createdAt: Date | string;
  updatedAt: Date | string;
};

export type CloudMetadata = {
  version: number;
  updatedAt: Date | string;
  games: CloudGameMetadata[];
};
