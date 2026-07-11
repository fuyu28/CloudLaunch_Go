/**
 * @fileoverview 批評空間連携型定義
 *
 * ErogameScape 検索結果・取得結果の TypeScript 型。
 */

export type ErogameScapeSearchItem = {
  erogameScapeId: string;
  title: string;
  brand?: string;
  gameUrl: string;
};

export type ErogameScapeSearchResult = {
  items: ErogameScapeSearchItem[];
  nextPageUrl?: string;
};
