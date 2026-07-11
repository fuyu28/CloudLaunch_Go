/**
 * @fileoverview ゲーム編集操作フック
 *
 * このフックは、ゲームの編集・削除・起動機能を提供します。
 */

import { useState, useCallback, useMemo } from "react";

import { handleApiError, showSuccessToast } from "@renderer/utils/errorHandler";
import { UNCONFIGURED_EXE_PATH } from "@renderer/constants/game";

import type { GameType, InputGameData } from "src/types/game";
import type { ApiResult } from "src/types/result";
import type { NavigateFunction } from "react-router-dom";

type SetterOrUpdater<Value> = (value: Value | ((prev: Value) => Value)) => void;

export type GameEditResult = {
  editData: InputGameData | undefined;
  isEditModalOpen: boolean;
  isDeleteModalOpen: boolean;
  isLaunching: boolean;
  openEdit: () => void;
  closeEdit: () => void;
  onEditClosed: () => void;
  openDelete: () => void;
  closeDelete: () => void;
  handleUpdateGame: (values: InputGameData) => Promise<ApiResult<void>>;
  handleDeleteGame: () => Promise<void>;
  handleLaunchGame: () => Promise<void>;
};

export function useGameEdit(
  game: GameType | undefined,
  navigate: NavigateFunction,
  setFilteredGames: SetterOrUpdater<GameType[]>,
): GameEditResult {
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [isLaunching, setIsLaunching] = useState(false);

  const editData = useMemo(() => {
    if (!game) return undefined;
    const { title, publisher, imagePath, exePath, saveFolderPath } = game;
    return {
      title,
      publisher,
      imagePath,
      exePath: exePath === UNCONFIGURED_EXE_PATH ? "" : exePath,
      saveFolderPath,
    };
  }, [game]);

  const openEdit = useCallback(() => {
    if (!editData) return;
    setIsEditModalOpen(true);
  }, [editData]);

  const closeEdit = useCallback(() => {
    setIsEditModalOpen(false);
  }, []);

  const onEditClosed = useCallback(() => {
    // メモ化されたeditDataを使用するため、特別な処理は不要
  }, []);

  const openDelete = useCallback(() => {
    setIsDeleteModalOpen(true);
  }, []);

  const closeDelete = useCallback(() => {
    setIsDeleteModalOpen(false);
  }, []);

  const handleUpdateGame = useCallback(
    async (values: InputGameData): Promise<ApiResult<void>> => {
      if (!game) {
        return { success: false, message: "ゲームが見つかりません。" };
      }

      const result = await window.api.database.updateGame(game.id, values);

      if (result.success) {
        showSuccessToast("ゲーム情報を更新しました。");

        setFilteredGames((list) => list.map((g) => (g.id === game.id ? { ...g, ...values } : g)));
      } else {
        handleApiError(result);
      }

      return result;
    },
    [game, setFilteredGames],
  );

  const handleDeleteGame = useCallback(async (): Promise<void> => {
    if (!game) return;

    const result = await window.api.database.deleteGame(game.id);

    if (result.success) {
      showSuccessToast("ゲームを削除しました。");

      setFilteredGames((g) => g.filter((x) => x.id !== game.id));

      navigate("/", { replace: true });
    } else {
      handleApiError(result);
    }

    closeDelete();
  }, [game, navigate, setFilteredGames, closeDelete]);

  const handleLaunchGame = useCallback(async (): Promise<void> => {
    if (!game) return;

    if (!game.exePath || game.exePath === UNCONFIGURED_EXE_PATH) {
      handleApiError({ success: false, message: "実行ファイルのパスが未設定です" });
      return;
    }

    setIsLaunching(true);

    try {
      const result = await window.api.game.launchGame(game.exePath);

      if (result.success) {
        showSuccessToast("ゲームを起動しました。");
      } else {
        handleApiError(result);
      }
    } finally {
      setIsLaunching(false);
    }
  }, [game]);

  return {
    editData,
    isEditModalOpen,
    isDeleteModalOpen,
    isLaunching,
    openEdit,
    closeEdit,
    onEditClosed,
    openDelete,
    closeDelete,
    handleUpdateGame,
    handleDeleteGame,
    handleLaunchGame,
  };
}

export default useGameEdit;
