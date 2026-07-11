/**
 * @fileoverview メモ管理機能の型定義
 *
 * メモに関連するすべての型定義を集約し、フロントエンドとバックエンド間で
 * 一貫した型安全性を提供します。
 */

/**
 * メモのデータ型
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

/**
 * メモ作成時のデータ型
 */
export type CreateMemoData = {
  title: string;
  content: string;
  gameId: string;
};

/**
 * メモ更新時のデータ型
 */
export type UpdateMemoData = {
  title: string;
  content: string;
};

/**
 * メモファイル操作の結果型
 */
export type MemoFileOperationResult = {
  success: boolean;
  filePath?: string;
  error?: string;
};

/**
 * メモディレクトリ情報型
 */
export type MemoDirectoryInfo = {
  baseDir: string;
  gameDir: string;
  fileCount: number;
};

/**
 * クラウドメモ情報型
 */
export type CloudMemoInfo = {
  key: string;
  fileName: string;
  gameId: string;
  memoTitle: string;
  memoId: string;
  lastModified: Date;
  size: number;
};

/**
 * メモ同期結果型
 */
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
