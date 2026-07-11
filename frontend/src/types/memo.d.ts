/**
 * @fileoverview メモ管理機能の型定義
 *
 * メモに関連するすべての型定義を集約し、フロントエンドとバックエンド間で
 * 一貫した型安全性を提供します。
 */

export type MemoType = {
  id: string;
  title: string;
  content: string;
  gameId: string;
  gameTitle?: string;
  createdAt: Date;
  updatedAt: Date;
};

export type CreateMemoData = {
  title: string;
  content: string;
  gameId: string;
};

export type UpdateMemoData = {
  title: string;
  content: string;
};

export type MemoFileOperationResult = {
  success: boolean;
  filePath?: string;
  error?: string;
};

export type MemoDirectoryInfo = {
  baseDir: string;
  gameDir: string;
  fileCount: number;
};

export type CloudMemoInfo = {
  key: string;
  fileName: string;
  gameId: string;
  memoTitle: string;
  memoId: string;
  lastModified: Date;
  size: number;
};

export type MemoSyncResult = {
  success: boolean;
  uploaded: number;
  /** ローカルで上書きされたメモ数（クラウド→ローカル） */
  localOverwritten: number;
  /** クラウドで上書きされたメモ数（ローカル→クラウド） */
  cloudOverwritten: number;
  created: number;
  updated: number;
  skipped: number;
  error?: string;
  details: string[];
};
