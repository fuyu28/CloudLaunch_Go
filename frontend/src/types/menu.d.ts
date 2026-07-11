/**
 * @fileoverview 一覧メニュー用型定義
 *
 * ソート項目・方向・プレイ状態フィルタの TypeScript 型。
 */

export type SortOption = "title" | "lastPlayed" | "totalPlayTime" | "publisher" | "lastRegistered";

export type SortDirection = "asc" | "desc";

export type FilterOption = "all" | "unplayed" | "playing" | "played";
