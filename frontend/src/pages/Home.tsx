import { useAtom, useAtomValue } from "jotai";
import { useEffect, useState, useCallback } from "react";
import { IoIosAdd } from "react-icons/io";

import ConfirmModal from "@renderer/components/ConfirmModal";
import FloatingButton from "@renderer/components/FloatingButton";
import CloudGameImportModal from "@renderer/components/CloudGameImportModal";
import ErogameScapeImportModal from "@renderer/components/ErogameScapeImportModal";
import GameGrid from "@renderer/components/GameGrid";
import GameFormModal from "@renderer/components/GameModal";
import GameSearchFilter from "@renderer/components/GameSearchFilter";

import { CONFIG, MESSAGES } from "@renderer/constants";
import { UNCONFIGURED_EXE_PATH } from "@renderer/constants/game";
import { useDebounce } from "@renderer/hooks/useDebounce";
import { useGameActions } from "@renderer/hooks/useGameActions";
import { useGameSaveData } from "@renderer/hooks/useGameSaveData";
import { useLoadingState } from "@renderer/hooks/useLoadingState";
import { useOfflineMode } from "@renderer/hooks/useOfflineMode";
import { useValidateCreds } from "@renderer/hooks/useValidCreds";
import { isValidCredsAtom } from "@renderer/state/credentials";
import {
  searchWordAtom,
  filterAtom,
  sortAtom,
  sortDirectionAtom,
  visibleGamesAtom,
} from "@renderer/state/home";
import { autoTrackingAtom } from "@renderer/state/settings";
import type { GameType } from "src/types/game";

export default function Home(): React.ReactElement {
  const [searchWord, setSearchWord] = useAtom(searchWordAtom);
  const [filter, setFilter] = useAtom(filterAtom);
  const [sort, setSort] = useAtom(sortAtom);
  const [sortDirection, setSortDirection] = useAtom(sortDirectionAtom);
  const [visibleGames, setVisibleGames] = useAtom(visibleGamesAtom);
  const [autoTracking] = useAtom(autoTrackingAtom);
  const [isGameFormOpen, setIsGameFormOpen] = useState(false);
  const [isImportOpen, setIsImportOpen] = useState(false);
  const [isErogameScapeImportOpen, setIsErogameScapeImportOpen] = useState(false);
  const [isDownloadConfirmOpen, setIsDownloadConfirmOpen] = useState(false);
  const [pendingLaunchGame, setPendingLaunchGame] = useState<GameType | null>(null);
  const isValidCreds = useAtomValue(isValidCredsAtom);
  const validateCreds = useValidateCreds();
  const { isOfflineMode } = useOfflineMode();
  const { downloadSaveData } = useGameSaveData();

  // 検索語をデバウンス
  const debouncedSearchWord = useDebounce(searchWord, CONFIG.TIMING.SEARCH_DEBOUNCE_MS);

  // ローディング状態管理
  const gameListLoading = useLoadingState();
  const gameActionLoading = useLoadingState();

  // ゲーム操作フック
  const { createGameAndRefreshList } = useGameActions({
    searchWord: debouncedSearchWord,
    filter,
    sort,
    sortDirection,
    onGamesUpdate: setVisibleGames,
    onModalClose: () => setIsGameFormOpen(false),
  });

  const refreshGameList = useCallback(async (): Promise<void> => {
    const games = await gameListLoading.executeWithLoading(
      () => window.api.database.listGames(debouncedSearchWord, filter, sort, sortDirection),
      {
        errorMessage: MESSAGES.GAME.LIST_FETCH_FAILED,
        showToast: true,
      },
    );

    if (games) {
      setVisibleGames(games as GameType[]);
    }
  }, [debouncedSearchWord, filter, gameListLoading, setVisibleGames, sort, sortDirection]);

  useEffect(() => {
    let cancelled = false;

    const fetchGames = async (): Promise<void> => {
      const games = await gameListLoading.executeWithLoading(
        () => window.api.database.listGames(debouncedSearchWord, filter, sort, sortDirection),
        {
          errorMessage: MESSAGES.GAME.LIST_FETCH_FAILED,
          showToast: true,
        },
      );

      if (!cancelled && games) {
        setVisibleGames(games as GameType[]);
      }
    };

    fetchGames();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debouncedSearchWord, filter, sort, sortDirection]);

  useEffect(() => {
    if (!isOfflineMode) {
      validateCreds();
    }
  }, [validateCreds, isOfflineMode]);

  const handleAddGame = createGameAndRefreshList;

  const launchGameDirect = useCallback(
    async (game: GameType): Promise<void> => {
      await gameActionLoading.executeWithLoading(
        async () => {
          const result = await window.api.game.launchGame(game.exePath);
          if (!result.success) {
            throw new Error(result.message);
          }
          return result;
        },
        {
          loadingMessage: MESSAGES.GAME.LAUNCHING,
          successMessage: MESSAGES.GAME.LAUNCHED,
          errorMessage: MESSAGES.GAME.LAUNCH_FAILED,
          showToast: true,
        },
      );
    },
    [gameActionLoading],
  );

  const handleLaunchGame = useCallback(
    async (game: GameType) => {
      if (!game.exePath || game.exePath === UNCONFIGURED_EXE_PATH) {
        await gameActionLoading.executeWithLoading(
          async () => {
            throw new Error("実行ファイルのパスが未設定です");
          },
          {
            loadingMessage: MESSAGES.GAME.LAUNCHING,
            errorMessage: MESSAGES.GAME.LAUNCH_FAILED,
            showToast: true,
          },
        );
        return;
      }

      if (!game.saveFolderPath || isOfflineMode || !isValidCreds) {
        await launchGameDirect(game);
        return;
      }

      const cloudHashResult = await window.api.saveData.hash.getCloudHash(game.id);
      if (!cloudHashResult.success || !cloudHashResult.data?.hash) {
        await launchGameDirect(game);
        return;
      }

      const localHashResult = await window.api.saveData.hash.computeLocalHash(game.saveFolderPath);
      if (
        localHashResult.success &&
        localHashResult.data &&
        localHashResult.data !== cloudHashResult.data.hash
      ) {
        setPendingLaunchGame(game);
        setIsDownloadConfirmOpen(true);
        return;
      }

      await launchGameDirect(game);
    },
    [gameActionLoading, isOfflineMode, isValidCreds, launchGameDirect],
  );

  const handleDownloadAndLaunch = useCallback(async (): Promise<void> => {
    setIsDownloadConfirmOpen(false);
    if (!pendingLaunchGame) {
      return;
    }
    await downloadSaveData(pendingLaunchGame);
    await launchGameDirect(pendingLaunchGame);
    setPendingLaunchGame(null);
  }, [downloadSaveData, launchGameDirect, pendingLaunchGame]);

  const handleSkipDownloadAndLaunch = useCallback(async (): Promise<void> => {
    setIsDownloadConfirmOpen(false);
    if (!pendingLaunchGame) {
      return;
    }
    await launchGameDirect(pendingLaunchGame);
    setPendingLaunchGame(null);
  }, [launchGameDirect, pendingLaunchGame]);

  return (
    <div className="flex flex-col h-full min-h-0 relative">
      {/* 検索・フィルタ領域 */}
      <GameSearchFilter
        searchWord={searchWord}
        sort={sort}
        sortDirection={sortDirection}
        filter={filter}
        onSearchWordChange={setSearchWord}
        onSortChange={setSort}
        onSortDirectionChange={setSortDirection}
        onFilterChange={setFilter}
      />

      {/* ゲーム一覧 */}
      <GameGrid games={visibleGames} onLaunchGame={handleLaunchGame} />

      {/* ゲーム追加ボタン */}
      <FloatingButton
        onClick={() => setIsGameFormOpen(true)}
        ariaLabel="ゲームを追加"
        positionClass={autoTracking ? "bottom-16 right-6" : "bottom-6 right-6"}
      >
        <IoIosAdd size={28} />
      </FloatingButton>

      {/* ゲーム登録モーダル */}
      <GameFormModal
        mode="add"
        isOpen={isGameFormOpen}
        onClose={() => setIsGameFormOpen(false)}
        onSubmit={handleAddGame}
        onOpenCloudImport={() => setIsImportOpen(true)}
        onOpenErogameScapeImport={() => setIsErogameScapeImportOpen(true)}
      />

      <ErogameScapeImportModal
        isOpen={isErogameScapeImportOpen}
        onClose={() => setIsErogameScapeImportOpen(false)}
        onSubmit={handleAddGame}
      />

      <CloudGameImportModal
        isOpen={isImportOpen}
        onClose={() => setIsImportOpen(false)}
        localGames={visibleGames}
        onImported={refreshGameList}
      />

      <ConfirmModal
        id="download-save-before-launch-modal"
        isOpen={isDownloadConfirmOpen}
        title="セーブデータの同期"
        message={`${pendingLaunchGame?.title ?? "このゲーム"} のセーブデータがクラウドと異なります。\nダウンロードしますか？`}
        cancelText="しない"
        confirmText="ダウンロードする"
        onConfirm={handleDownloadAndLaunch}
        onCancel={handleSkipDownloadAndLaunch}
      />
    </div>
  );
}
