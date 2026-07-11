/**
 * @fileoverview ゲーム操作フック
 *
 * このファイルは、ゲームの追加・編集・削除に関する共通ロジックを提供します。
 */

import { useCallback } from "react";

import { useLoadingState } from "./useLoadingState";
import { MESSAGES } from "@renderer/constants";
import type { InputGameData, GameType } from "src/types/game";
import type { SortOption, FilterOption, SortDirection } from "src/types/menu";
import type { ApiResult } from "src/types/result";

export type UseGameActionsProps = {
  searchWord: string;
  filter: FilterOption;
  sort: SortOption;
  sortDirection: SortDirection;
  onGamesUpdate: (games: GameType[]) => void;
  onModalClose: () => void;
};

export function useGameActions({
  searchWord,
  filter,
  sort,
  sortDirection,
  onGamesUpdate,
  onModalClose,
}: UseGameActionsProps): {
  createGameAndRefreshList: (values: InputGameData) => Promise<ApiResult<void>>;
  isLoading: boolean;
} {
  const gameActionLoading = useLoadingState();

  const createGameAndRefreshList = useCallback(
    async (values: InputGameData): Promise<ApiResult<void>> => {
      const result = await gameActionLoading.executeWithLoading(
        async () => {
          const createResult = await window.api.database.createGame(values);
          if (!createResult.success) {
            throw new Error((createResult as { success: false; message: string }).message);
          }

          const games = await window.api.database.listGames(
            searchWord,
            filter,
            sort,
            sortDirection,
          );
          onGamesUpdate(games as GameType[]);
          onModalClose();

          return createResult;
        },
        {
          loadingMessage: MESSAGES.GAME.ADDING,
          successMessage: MESSAGES.GAME.ADDED,
          showToast: true,
        },
      );

      return result || { success: false, message: MESSAGES.GAME.ADD_FAILED };
    },
    [searchWord, filter, sort, sortDirection, onGamesUpdate, onModalClose, gameActionLoading],
  );

  return {
    createGameAndRefreshList,
    isLoading: gameActionLoading.isLoading,
  };
}
