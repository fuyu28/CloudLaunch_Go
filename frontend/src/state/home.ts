/**
 * @fileoverview ホーム画面関連 atoms
 *
 * 検索語・表示中ゲーム・ソート／フィルタ状態など一覧 UI の状態。
 */

import { atom } from "jotai";

import { defaultSortOptionAtom, defaultFilterStateAtom } from "./settings";
import type { GameType } from "src/types/game";
import type { FilterOption, SortOption, SortDirection } from "src/types/menu";

export const searchWordAtom = atom<string>("");

const _filterAtom = atom<FilterOption | null>(null);
const _sortAtom = atom<SortOption | null>(null);
const _sortDirectionAtom = atom<SortDirection>("desc");

export const filterAtom = atom(
  (get) => get(_filterAtom) ?? get(defaultFilterStateAtom),
  (_, set, newValue: FilterOption) => {
    set(_filterAtom, newValue);
  },
);

export const sortAtom = atom(
  (get) => get(_sortAtom) ?? get(defaultSortOptionAtom),
  (_, set, newValue: SortOption) => {
    set(_sortAtom, newValue);
  },
);

export const sortDirectionAtom = atom(
  (get) => get(_sortDirectionAtom),
  (_, set, newValue: SortDirection) => {
    set(_sortDirectionAtom, newValue);
  },
);

export const visibleGamesAtom = atom<GameType[]>([]);

export const currentGameIdAtom = atom<string | null>(null);
