/**
 * @fileoverview ホーム（ゲーム一覧）ページ
 *
 * 検索・フィルタ・ソート付きのゲーム一覧と追加／インポート操作を提供する。
 */

import { useAtom, useAtomValue } from "jotai";
import { useEffect, useState, useCallback } from "react";
import { IoIosAdd } from "react-icons/io";

import ConfirmModal from "@renderer/components/common/ConfirmModal";
import FloatingButton from "@renderer/components/common/FloatingButton";
import CloudGameImportModal from "@renderer/components/cloud/CloudGameImportModal";
import SyncConflictModal from "@renderer/components/cloud/SyncConflictModal";
import ErogameScapeImportModal from "@renderer/components/game/ErogameScapeImportModal";
import GameGrid from "@renderer/components/game/GameGrid";
import GameFormModal from "@renderer/components/game/GameModal";
import GameSearchFilter from "@renderer/components/game/GameSearchFilter";

import { CONFIG, MESSAGES } from "@renderer/constants";
import { UNCONFIGURED_EXE_PATH } from "@renderer/constants/game";
import { useDebounce } from "@renderer/hooks/useDebounce";
import { useGameActions } from "@renderer/hooks/useGameActions";
import { useGameSaveData } from "@renderer/hooks/useGameSaveData";
import { useCloudSync } from "@renderer/hooks/useCloudSync";
import { useLoadingState } from "@renderer/hooks/useLoadingState";
import { useOfflineMode } from "@renderer/hooks/useOfflineMode";
import { useTimeFormat } from "@renderer/hooks/useTimeFormat";
import { useToastHandler } from "@renderer/hooks/useToastHandler";
import { useValidateCreds } from "@renderer/hooks/useValidCreds";
import { isValidCredsAtom } from "@renderer/state/credentials";
import { buildSaveSyncMessage } from "@renderer/utils/saveSyncMessage";
import { checkDirectoryExists, checkFileExists } from "@renderer/utils/fileValidation";
import {
  searchWordAtom,
  filterAtom,
  sortAtom,
  sortDirectionAtom,
  visibleGamesAtom,
} from "@renderer/state/home";
import { autoTrackingAtom } from "@renderer/state/settings";
import type { GameType } from "src/types/game";
import type { SyncMetaSnapshot } from "src/wailsBridge";

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
  const [saveSyncMessage, setSaveSyncMessage] = useState("");
  const [isConflictModalOpen, setIsConflictModalOpen] = useState(false);
  const [isResolvingConflict, setIsResolvingConflict] = useState(false);
  const [conflictMeta, setConflictMeta] = useState<{
    localMeta?: SyncMetaSnapshot;
    remoteMeta?: SyncMetaSnapshot;
  } | null>(null);
  const [warningGameIds, setWarningGameIds] = useState<Set<string>>(new Set());
  const isValidCreds = useAtomValue(isValidCredsAtom);
  const validateCreds = useValidateCreds();
  const { isOfflineMode } = useOfflineMode();
  const { getStatus, resolveConflict } = useCloudSync(isOfflineMode);
  const { downloadSaveData } = useGameSaveData();
  const { formatDateWithTime } = useTimeFormat();
  const { showToast } = useToastHandler();

  const debouncedSearchWord = useDebounce(searchWord, CONFIG.TIMING.SEARCH_DEBOUNCE_MS);

  const gameListLoading = useLoadingState();
  const gameActionLoading = useLoadingState();

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

  useEffect(() => {
    let cancelled = false;

    const resolveWarnings = async (): Promise<void> => {
      const entries = await Promise.all(
        visibleGames.map(async (game) => {
          const hasUnconfiguredExe = !game.exePath || game.exePath === UNCONFIGURED_EXE_PATH;
          const exeMissing = hasUnconfiguredExe ? true : !(await checkFileExists(game.exePath));
          const savePath = (game.saveFolderPath || "").trim();
          const saveMissing = savePath !== "" ? !(await checkDirectoryExists(savePath)) : false;
          return [game.id, exeMissing || saveMissing] as const;
        }),
      );

      if (cancelled) {
        return;
      }

      const next = new Set<string>();
      for (const [gameID, hasWarning] of entries) {
        if (hasWarning) {
          next.add(gameID);
        }
      }
      setWarningGameIds(next);
    };

    void resolveWarnings();
    return () => {
      cancelled = true;
    };
  }, [visibleGames]);

  const buildSyncMessage = useCallback(
    (
      title: string,
      localUpdatedAt: Date | string | number | null | undefined,
      cloudUpdatedAt: Date | string | number | null | undefined,
    ) => buildSaveSyncMessage(formatDateWithTime, title, localUpdatedAt, cloudUpdatedAt),
    [formatDateWithTime],
  );

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

      const exeExists = await checkFileExists(game.exePath);
      if (!exeExists) {
        await gameActionLoading.executeWithLoading(
          async () => {
            throw new Error("実行ファイルが見つかりません");
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

      const statusResult = await getStatus(game.id);
      if (!statusResult.success || !statusResult.data) {
        await launchGameDirect(game);
        return;
      }

      const { status, remoteMeta } = statusResult.data;
      if (status === "conflict") {
        setConflictMeta({
          localMeta: statusResult.data.localMeta,
          remoteMeta: statusResult.data.remoteMeta,
        });
        setPendingLaunchGame(game);
        setIsConflictModalOpen(true);
        return;
      }
      if (status === "pull_needed") {
        setSaveSyncMessage(
          buildSyncMessage(game.title, game.localSaveHashUpdatedAt, remoteMeta?.createdAt ?? null),
        );
        setPendingLaunchGame(game);
        setIsDownloadConfirmOpen(true);
        return;
      }

      await launchGameDirect(game);
    },
    [gameActionLoading, isOfflineMode, isValidCreds, launchGameDirect, getStatus, buildSyncMessage],
  );

  const handleDownloadAndLaunch = useCallback(async (): Promise<void> => {
    setIsDownloadConfirmOpen(false);
    setSaveSyncMessage("");
    if (!pendingLaunchGame) {
      return;
    }
    const downloaded = await downloadSaveData(pendingLaunchGame);
    if (downloaded) {
      await launchGameDirect(pendingLaunchGame);
    }
    setPendingLaunchGame(null);
  }, [downloadSaveData, launchGameDirect, pendingLaunchGame]);

  const handleSkipDownloadAndLaunch = useCallback(async (): Promise<void> => {
    setIsDownloadConfirmOpen(false);
    setSaveSyncMessage("");
    if (!pendingLaunchGame) {
      return;
    }
    await launchGameDirect(pendingLaunchGame);
    setPendingLaunchGame(null);
  }, [launchGameDirect, pendingLaunchGame]);

  const handleResolveConflict = useCallback(
    async (useLocal: boolean): Promise<void> => {
      if (!pendingLaunchGame) return;
      setIsResolvingConflict(true);
      try {
        const op = await resolveConflict(pendingLaunchGame.id, useLocal);
        if (op.ok && !(op.applied === false)) {
          showToast(
            useLocal
              ? "ローカルデータをクラウドに反映しました"
              : "クラウドデータをローカルに適用しました",
            "success",
          );
          setIsConflictModalOpen(false);
          setConflictMeta(null);
          await launchGameDirect(pendingLaunchGame);
          setPendingLaunchGame(null);
          return;
        }
        if (op.ok && op.applied === false) {
          showToast(
            "同期対象外のローカルファイルがあるため、ゲーム詳細の「同期」から確認してください。",
            "error",
          );
          setIsConflictModalOpen(false);
          setConflictMeta(null);
          setPendingLaunchGame(null);
          return;
        }
        showToast(op.message || "コンフリクト解決に失敗しました", "error");
      } catch {
        showToast("コンフリクト解決に失敗しました", "error");
      } finally {
        setIsResolvingConflict(false);
      }
    },
    [pendingLaunchGame, resolveConflict, showToast, launchGameDirect],
  );

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
      <GameGrid
        games={visibleGames}
        onLaunchGame={handleLaunchGame}
        warningGameIds={warningGameIds}
      />

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
        message={
          saveSyncMessage ||
          `${pendingLaunchGame?.title ?? "このゲーム"} のセーブデータがクラウドと異なります。\nダウンロードしますか？`
        }
        cancelText="しない"
        confirmText="ダウンロードする"
        onConfirm={handleDownloadAndLaunch}
        onCancel={handleSkipDownloadAndLaunch}
      />

      <SyncConflictModal
        isOpen={isConflictModalOpen}
        onClose={() => {
          setIsConflictModalOpen(false);
          setConflictMeta(null);
          setPendingLaunchGame(null);
        }}
        gameTitle={pendingLaunchGame?.title ?? ""}
        localMeta={conflictMeta?.localMeta}
        remoteMeta={conflictMeta?.remoteMeta}
        onUseLocal={() => void handleResolveConflict(true)}
        onUseRemote={() => void handleResolveConflict(false)}
        isResolving={isResolvingConflict}
      />
    </div>
  );
}
