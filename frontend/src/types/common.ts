/**
 * @fileoverview 共通型定義のエクスポート
 *
 * このファイルは、types ディレクトリ内の共通的に使用される型をまとめてエクスポートします。
 */

export * from "./validation";

export * from "./path";

export * from "./result";
export * from "./error";

/**
 * 汎用的なオプション型
 */
export type SelectOption<T = string> = {
  label: string;
  value: T;
  disabled?: boolean;
};

/**
 * ページネーション情報
 */
export type PaginationInfo = {
  currentPage: number;
  totalPages: number;
  itemsPerPage: number;
  totalItems: number;
};

/**
 * ソート情報
 */
export type SortInfo<T extends string = string> = {
  field: T;
  direction: "asc" | "desc";
};

/**
 * 検索フィルター
 */
export type SearchFilter = {
  query?: string;
  category?: string;
  dateRange?: {
    from?: Date;
    to?: Date;
  };
};

/**
 * ローディング状態
 */
export type LoadingState = {
  isLoading: boolean;
  error?: string;
  lastUpdated?: Date;
};

/**
 * 非同期操作の状態
 */
export type AsyncStatus = "idle" | "loading" | "success" | "error";

/**
 * 非同期操作の状態管理
 */
export type AsyncState<T = unknown> = {
  status: AsyncStatus;
  data?: T;
  error?: string;
};

/**
 * キーと値のペア（汎用）
 */
export type KeyValuePair<K = string, V = unknown> = {
  key: K;
  value: V;
};

/**
 * 範囲指定
 */
export type Range<T = number> = {
  min: T;
  max: T;
};

/**
 * 座標情報
 */
export type Coordinates = {
  x: number;
  y: number;
};

/**
 * サイズ情報
 */
export type Size = {
  width: number;
  height: number;
};
