import { isValidCredsAtom } from "@renderer/state/credentials";
import { visibleGamesAtom, currentGameIdAtom } from "@renderer/state/home";
import { useAtomValue, useSetAtom } from "jotai";
import { useCallback, useEffect, useState } from "react";
import { useParams, useNavigate, Navigate } from "react-router-dom";

import CloudDataCard from "@renderer/components/CloudDataCard";
import ConfirmModal from "@renderer/components/ConfirmModal";
import SyncConflictModal from "@renderer/components/SyncConflictModal";
import UntrackedDeleteModal from "@renderer/components/UntrackedDeleteModal";
import GameInfo from "@renderer/components/GameInfo";
import GameFormModal from "@renderer/components/GameModal";
import MemoCard from "@renderer/components/MemoCard";
import PlaySessionManagementModal from "@renderer/components/PlaySessionManagementModal";
import PlaySessionModal from "@renderer/components/PlaySessionModal";
import PlayStatistics from "@renderer/components/PlayStatistics";

import { useGameEdit } from "@renderer/hooks/useGameEdit";
import { useGameSaveData } from "@renderer/hooks/useGameSaveData";
import { useOfflineMode } from "@renderer/hooks/useOfflineMode";
import { useToastHandler } from "@renderer/hooks/useToastHandler";
import { useTimeFormat } from "@renderer/hooks/useTimeFormat";
import { useValidateCreds } from "@renderer/hooks/useValidCreds";

import { logger } from "@renderer/utils/logger";

import type { GameType } from "src/types/game";
import type { SyncMetaSnapshot } from "src/wailsBridge";

export default function GameDetail(): React.JSX.Element {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const setVisibleGames = useSetAtom(visibleGamesAtom);
  const setCurrentGameId = useSetAtom(currentGameIdAtom);
  const visibleGames = useAtomValue(visibleGamesAtom);
  const isValidCreds = useAtomValue(isValidCredsAtom);
  const validateCreds = useValidateCreds();
  const [game, setGame] = useState<GameType | undefined>(undefined);
  const [isLoadingGame, setIsLoadingGame] = useState(true);
  const [isPlaySessionModalOpen, setIsPlaySessionModalOpen] = useState(false);
  const [isProcessModalOpen, setIsProcessModalOpen] = useState(false);
  const [refreshKey, setRefreshKey] = useState(0);
  const [isUpdatingStatus, setIsUpdatingStatus] = useState(false);
  const [isDownloadConfirmOpen, setIsDownloadConfirmOpen] = useState(false);
  const [saveSyncMessage, setSaveSyncMessage] = useState("");
  const [isConflictModalOpen, setIsConflictModalOpen] = useState(false);
  const [conflictMeta, setConflictMeta] = useState<{
    localMeta?: SyncMetaSnapshot;
    remoteMeta?: SyncMetaSnapshot;
  } | null>(null);
  const [isResolvingConflict, setIsResolvingConflict] = useState(false);
  const [isSyncing, setIsSyncing] = useState(false);
  // 同期管理外（untracked）ファイルの削除確認モーダル状態
  const [untrackedDeletes, setUntrackedDeletes] = useState<string[] | null>(null);
  // 確認後に再実行する操作の種別（通常 pull か、コンフリクトのリモート採用か）
  const [untrackedConfirmKind, setUntrackedConfirmKind] = useState<"pull" | "resolveRemote" | null>(
    null,
  );
  const [isDeletingUntracked, setIsDeletingUntracked] = useState(false);
  const { showToast } = useToastHandler();
  const { isOfflineMode, checkNetworkFeature } = useOfflineMode();
  const { formatDateWithTime } = useTimeFormat();

  // ゲームデータを取得
  useEffect(() => {
    if (!id) return;

    const fetchGame = async (): Promise<void> => {
      setIsLoadingGame(true);
      try {
        // まずvisibleGamesから検索
        const existingGame = visibleGames.find((g) => g.id === id);
        if (existingGame) {
          setGame(existingGame);
          setCurrentGameId(id);
          setIsLoadingGame(false);
          return;
        }

        // visibleGamesにない場合は直接データベースから取得
        const fetchedGame = await window.api.database.getGameById(id);
        if (fetchedGame) {
          // APIから返されたデータは既にtransformされているのでそのまま使用
          const transformedGame = fetchedGame as GameType;
          setGame(transformedGame);
          setCurrentGameId(id);
          // visibleGamesも更新
          setVisibleGames((prev) => {
            const exists = prev.find((g) => g.id === id);
            return exists ? prev : [...prev, transformedGame];
          });
        } else {
          setGame(undefined);
        }
      } catch (error) {
        logger.error("ゲームデータの取得に失敗", {
          component: "GameDetail",
          function: "loadGame",
          error: error instanceof Error ? error : new Error(String(error)),
          data: { gameId: id },
        });
        setGame(undefined);
      } finally {
        setIsLoadingGame(false);
      }
    };

    fetchGame();
  }, [id, visibleGames, setCurrentGameId, setVisibleGames]);

  // コンポーネントのアンマウント時にIDをクリア
  useEffect(() => {
    return () => {
      setCurrentGameId(null);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // カスタムフック
  const { uploadSaveData, downloadSaveData, isUploading, isDownloading } = useGameSaveData();
  const {
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
  } = useGameEdit(game, navigate, setVisibleGames);

  useEffect(() => {
    if (!isOfflineMode) {
      validateCreds();
    }
  }, [validateCreds, isOfflineMode]);

  // セーブデータ操作のコールバック
  const handleUploadSaveData = useCallback(async (): Promise<void> => {
    if (!checkNetworkFeature("セーブデータアップロード")) {
      return;
    }
    if (game) {
      await uploadSaveData(game);
    }
  }, [game, uploadSaveData, checkNetworkFeature]);

  const handleDownloadSaveData = useCallback(async (): Promise<void> => {
    if (!checkNetworkFeature("セーブデータダウンロード")) {
      return;
    }
    if (game) {
      await downloadSaveData(game);
    }
  }, [game, downloadSaveData, checkNetworkFeature]);

  const toValidDate = useCallback(
    (value: Date | string | number | null | undefined): Date | null => {
      if (!value) return null;
      const parsed = new Date(value);
      return Number.isNaN(parsed.getTime()) ? null : parsed;
    },
    [],
  );

  const buildSaveSyncMessage = useCallback(
    (
      title: string,
      localUpdatedAt: Date | string | number | null | undefined,
      cloudUpdatedAt: Date | string | number | null | undefined,
    ) => {
      const localDate = toValidDate(localUpdatedAt);
      const cloudDate = toValidDate(cloudUpdatedAt);
      return `${title} のセーブデータがクラウドと異なります。\nローカル最終更新: ${formatDateWithTime(localDate)}\nクラウド最終更新: ${formatDateWithTime(cloudDate)}\nダウンロードしますか？`;
    },
    [formatDateWithTime, toValidDate],
  );

  // プレイセッション追加関連のコールバック
  const handleOpenPlaySessionModal = (): void => {
    setIsPlaySessionModalOpen(true);
  };

  const handleClosePlaySessionModal = (): void => {
    setIsPlaySessionModalOpen(false);
  };

  // 全データを再取得する関数
  const refreshGameData = useCallback(async () => {
    if (!game?.id) return;

    try {
      // ゲームデータを再取得
      const updatedGame = await window.api.database.getGameById(game.id);

      if (updatedGame) {
        // ローカルの状態を更新（APIから返されたデータは既にtransformされている）
        const transformedGame = updatedGame as GameType;
        setGame(transformedGame);
        // visibleGamesも更新
        setVisibleGames((prev) => prev.map((g) => (g.id === game.id ? transformedGame : g)));
      }
      // リフレッシュキーを更新してコンポーネントの再レンダリングを促す
      setRefreshKey((prev) => prev + 1);
    } catch (error) {
      logger.error("ゲームデータの更新に失敗", {
        component: "GameDetail",
        function: "refreshGameData",
        error: error instanceof Error ? error : new Error(String(error)),
        data: { gameId: game?.id },
      });
    }
  }, [game?.id, setVisibleGames]);

  const handleAddPlaySession = useCallback(
    async (duration: number, sessionName?: string): Promise<void> => {
      if (!game) return;

      try {
        const result = await window.api.database.createSession(duration, game.id, sessionName);
        if (result.success) {
          showToast("プレイセッションを追加しました", "success");
          // 全データを再取得
          await refreshGameData();
        } else {
          showToast(result.message || "プレイセッションの追加に失敗しました", "error");
        }
      } catch {
        showToast("プレイセッションの追加に失敗しました", "error");
      }
    },
    [game, showToast, refreshGameData],
  );

  const handleSyncGame = useCallback(
    async (showResult = true): Promise<boolean> => {
      if (!game) return false;
      if (isOfflineMode) {
        if (showResult) {
          showToast("オフラインモードでは同期できません", "error");
        }
        return false;
      }
      try {
        const statusResult = await window.api.cloudSync.status(game.id);
        if (!statusResult.success || !statusResult.data) {
          if (showResult)
            showToast(
              (!statusResult.success && statusResult.message) || "同期状態の取得に失敗しました",
              "error",
            );
          return false;
        }
        const { status } = statusResult.data;
        if (status === "in_sync") {
          if (showResult) showToast("すでに最新の状態です", "success");
          return true;
        }
        if (status === "never_synced") {
          if (showResult) showToast("クラウドにデータがありません", "error");
          return false;
        }
        if (status === "push_needed") {
          const pushResult = await window.api.cloudSync.push(game.id);
          if (!pushResult.success) {
            if (showResult) showToast(pushResult.message || "アップロードに失敗しました", "error");
            return false;
          }
          if (showResult) showToast("クラウドにアップロードしました", "success");
          return true;
        }
        if (status === "pull_needed") {
          const pullResult = await window.api.cloudSync.pull(game.id);
          if (!pullResult.success) {
            if (showResult) showToast(pullResult.message || "ダウンロードに失敗しました", "error");
            return false;
          }
          if (pullResult.data && !pullResult.data.applied) {
            // 同期管理外ファイルの削除確認が必要（ここまでローカル無変更）
            if (showResult) {
              setUntrackedDeletes(pullResult.data.untrackedDeletes ?? []);
              setUntrackedConfirmKind("pull");
            }
            return false;
          }
          if (showResult) showToast("クラウドからダウンロードしました", "success");
          return true;
        }
        // conflict
        if (showResult) {
          setConflictMeta({
            localMeta: statusResult.data.localMeta,
            remoteMeta: statusResult.data.remoteMeta,
          });
          setIsConflictModalOpen(true);
        }
        return false;
      } catch (error) {
        logger.error("ゲーム同期エラー:", {
          component: "GameDetail",
          function: "handleSyncGame",
          data: error,
        });
        if (showResult) {
          showToast("クラウド同期に失敗しました", "error");
        }
        return false;
      }
    },
    [game, isOfflineMode, showToast],
  );

  const launchGameDirect = useCallback(async (): Promise<void> => {
    if (!game) return;
    if (!isOfflineMode && isValidCreds) {
      await handleSyncGame(false);
    }
    await handleLaunchGame();
  }, [game, isOfflineMode, isValidCreds, handleSyncGame, handleLaunchGame]);

  const handleLaunchGameWithSync = useCallback(async (): Promise<void> => {
    if (!game) return;
    if (!game.saveFolderPath || isOfflineMode || !isValidCreds) {
      await launchGameDirect();
      return;
    }
    const statusResult = await window.api.cloudSync.status(game.id);
    if (!statusResult.success || !statusResult.data) {
      await launchGameDirect();
      return;
    }
    const { status, remoteMeta } = statusResult.data;
    if (status === "pull_needed" || status === "conflict") {
      setSaveSyncMessage(
        buildSaveSyncMessage(
          game.title,
          game.localSaveHashUpdatedAt,
          remoteMeta?.createdAt ?? null,
        ),
      );
      setIsDownloadConfirmOpen(true);
      return;
    }
    await launchGameDirect();
  }, [game, isOfflineMode, isValidCreds, launchGameDirect]);

  const handleDownloadAndLaunch = useCallback(async (): Promise<void> => {
    setIsDownloadConfirmOpen(false);
    setSaveSyncMessage("");
    await handleDownloadSaveData();
    await launchGameDirect();
  }, [handleDownloadSaveData, launchGameDirect]);

  const handleSkipDownloadAndLaunch = useCallback(async (): Promise<void> => {
    setIsDownloadConfirmOpen(false);
    setSaveSyncMessage("");
    await launchGameDirect();
  }, [launchGameDirect]);

  const handleResolveConflict = useCallback(
    async (useLocal: boolean): Promise<void> => {
      if (!game) return;
      setIsResolvingConflict(true);
      try {
        const result = await window.api.cloudSync.resolveConflict(game.id, useLocal);
        if (result.success) {
          if (!useLocal && result.data && !result.data.applied) {
            // リモート採用だが、同期管理外ファイルの削除確認が必要（ローカル無変更）
            setIsConflictModalOpen(false);
            setConflictMeta(null);
            setUntrackedDeletes(result.data.untrackedDeletes ?? []);
            setUntrackedConfirmKind("resolveRemote");
            return;
          }
          showToast(
            useLocal
              ? "ローカルデータをクラウドに反映しました"
              : "クラウドデータをローカルに適用しました",
            "success",
          );
          setIsConflictModalOpen(false);
          setConflictMeta(null);
        } else {
          showToast(result.message || "コンフリクト解決に失敗しました", "error");
        }
      } catch {
        showToast("コンフリクト解決に失敗しました", "error");
      } finally {
        setIsResolvingConflict(false);
      }
    },
    [game, showToast],
  );

  // untracked 削除の確認後、deleteUntracked=true で再実行する。
  const handleConfirmUntrackedDelete = useCallback(async (): Promise<void> => {
    if (!game || !untrackedConfirmKind) return;
    setIsDeletingUntracked(true);
    try {
      const result =
        untrackedConfirmKind === "pull"
          ? await window.api.cloudSync.pull(game.id, true)
          : await window.api.cloudSync.resolveConflict(game.id, false, true);
      if (result.success && result.data?.applied) {
        showToast("クラウドからダウンロードしました", "success");
        setUntrackedDeletes(null);
        setUntrackedConfirmKind(null);
        setRefreshKey((k) => k + 1);
      } else {
        showToast((!result.success && result.message) || "ダウンロードに失敗しました", "error");
      }
    } catch {
      showToast("ダウンロードに失敗しました", "error");
    } finally {
      setIsDeletingUntracked(false);
    }
  }, [game, untrackedConfirmKind, showToast]);

  const handleCancelUntrackedDelete = useCallback((): void => {
    setUntrackedDeletes(null);
    setUntrackedConfirmKind(null);
  }, []);

  const handleSyncCheck = useCallback(async (): Promise<void> => {
    if (!game) return;
    if (!checkNetworkFeature("同期確認")) return;
    setIsSyncing(true);
    try {
      await handleSyncGame(true);
    } finally {
      setIsSyncing(false);
    }
  }, [game, handleSyncGame, checkNetworkFeature]);

  // プレイステータス変更のハンドラー
  const handleStatusChange = useCallback(
    async (newStatus: "unplayed" | "playing" | "played"): Promise<void> => {
      if (!game) return;

      setIsUpdatingStatus(true);
      try {
        const result = await window.api.database.updatePlayStatus(game.id, newStatus);

        if (result.success) {
          showToast("プレイステータスを更新しました", "success");
          // 全データを再取得
          await refreshGameData();
        } else {
          showToast(result.message || "プレイステータスの更新に失敗しました", "error");
        }
      } catch (error) {
        logger.error("プレイステータスの更新エラー", {
          component: "GameDetail",
          function: "handleStatusChange",
          error: error instanceof Error ? error : new Error(String(error)),
          data: { gameId: game.id, newStatus },
        });
        showToast("プレイステータスの更新に失敗しました", "error");
      } finally {
        setIsUpdatingStatus(false);
      }
    },
    [game, showToast, refreshGameData],
  );

  if (!id) {
    return <Navigate to="/" replace />;
  }

  // ローディング中の表示
  if (isLoadingGame) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="loading loading-spinner loading-lg"></div>
      </div>
    );
  }

  // ゲームが見つからない場合のみリダイレクト
  if (!game) {
    return <Navigate to="/" replace />;
  }

  return (
    <div className="bg-base-200 px-6 ">
      {/* 上段：ゲーム情報カード */}
      <div className="mb-5">
        <GameInfo
          game={game}
          isUpdatingStatus={isUpdatingStatus}
          isLaunching={isLaunching}
          onStatusChange={(status) =>
            handleStatusChange(status as "unplayed" | "playing" | "played")
          }
          onLaunchGame={handleLaunchGameWithSync}
          onEditGame={openEdit}
          onDeleteGame={openDelete}
        />
      </div>

      {/* 中段：プレイ統計 */}
      <div className="mb-5">
        <PlayStatistics
          game={game}
          refreshKey={refreshKey}
          onAddPlaySession={handleOpenPlaySessionModal}
          onOpenProcessManagement={() => setIsProcessModalOpen(true)}
        />
      </div>

      {/* 下段：その他の管理機能 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
        {/* クラウドデータ管理カード */}
        <CloudDataCard
          gameId={game.id}
          gameTitle={game.title}
          hasSaveFolder={!!game.saveFolderPath}
          isValidCreds={isValidCreds}
          isUploading={isUploading}
          isDownloading={isDownloading}
          isSyncing={isSyncing}
          onUpload={handleUploadSaveData}
          onDownload={handleDownloadSaveData}
          onSync={handleSyncCheck}
        />

        {/* メモ管理カード */}
        <MemoCard gameId={game.id} />
      </div>

      {/* モーダル */}

      {/* 起動前セーブデータ同期 */}
      <ConfirmModal
        id="download-save-before-launch-modal"
        isOpen={isDownloadConfirmOpen}
        title="セーブデータの同期"
        message={
          saveSyncMessage ||
          `${game.title} のセーブデータがクラウドと異なります。\nダウンロードしますか？`
        }
        cancelText="しない"
        confirmText="ダウンロードする"
        onConfirm={handleDownloadAndLaunch}
        onCancel={handleSkipDownloadAndLaunch}
      />

      {/* 削除 */}
      <ConfirmModal
        id="delete-game-modal"
        isOpen={isDeleteModalOpen}
        message={`${game.title} を削除しますか？\nこの操作は取り消せません`}
        cancelText="キャンセル"
        confirmText="削除する"
        onConfirm={handleDeleteGame}
        onCancel={closeDelete}
      />

      {/* 編集 */}
      <GameFormModal
        mode="edit"
        initialData={editData}
        isOpen={isEditModalOpen}
        onSubmit={handleUpdateGame}
        onClose={closeEdit}
        onClosed={onEditClosed}
      />

      {/* プレイセッション追加 */}
      <PlaySessionModal
        isOpen={isPlaySessionModalOpen}
        onClose={handleClosePlaySessionModal}
        onSubmit={handleAddPlaySession}
        gameTitle={game.title}
      />

      {/* コンフリクト解決 */}
      <SyncConflictModal
        isOpen={isConflictModalOpen}
        onClose={() => {
          setIsConflictModalOpen(false);
          setConflictMeta(null);
        }}
        gameTitle={game.title}
        localMeta={conflictMeta?.localMeta}
        remoteMeta={conflictMeta?.remoteMeta}
        onUseLocal={() => handleResolveConflict(true)}
        onUseRemote={() => handleResolveConflict(false)}
        isResolving={isResolvingConflict}
      />

      {/* 同期管理外ファイルの削除確認 */}
      <UntrackedDeleteModal
        isOpen={untrackedDeletes !== null}
        onClose={handleCancelUntrackedDelete}
        gameTitle={game.title}
        files={untrackedDeletes ?? []}
        onConfirm={handleConfirmUntrackedDelete}
        isProcessing={isDeletingUntracked}
      />

      {/* プロセス管理 */}
      <PlaySessionManagementModal
        isOpen={isProcessModalOpen}
        gameId={game.id}
        gameTitle={game.title}
        onClose={() => setIsProcessModalOpen(false)}
        onProcessUpdated={refreshGameData}
      />
    </div>
  );
}
