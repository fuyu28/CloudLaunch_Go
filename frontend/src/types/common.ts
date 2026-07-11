/**
 * @fileoverview 共通型定義のエクスポート
 *
 * このファイルは、types ディレクトリ内の共通的に使用される型をまとめてエクスポートします。
 */

export * from "./validation";

export * from "./path";

export * from "./result";
export * from "./error";

export type SelectOption<T = string> = {
  label: string;
  value: T;
  disabled?: boolean;
};

export type PaginationInfo = {
  currentPage: number;
  totalPages: number;
  itemsPerPage: number;
  totalItems: number;
};

export type SortInfo<T extends string = string> = {
  field: T;
  direction: "asc" | "desc";
};

export type SearchFilter = {
  query?: string;
  category?: string;
  dateRange?: {
    from?: Date;
    to?: Date;
  };
};

export type LoadingState = {
  isLoading: boolean;
  error?: string;
  lastUpdated?: Date;
};

export type AsyncStatus = "idle" | "loading" | "success" | "error";

export type AsyncState<T = unknown> = {
  status: AsyncStatus;
  data?: T;
  error?: string;
};

export type KeyValuePair<K = string, V = unknown> = {
  key: K;
  value: V;
};

export type Range<T = number> = {
  min: T;
  max: T;
};

export type Coordinates = {
  x: number;
  y: number;
};

export type Size = {
  width: number;
  height: number;
};
