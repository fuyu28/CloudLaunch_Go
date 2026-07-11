/**
 * @fileoverview ゲーム詳細ページ
 *
 * 個別ゲームの情報表示、起動、同期、メモ、セッション管理のハブ。
 */

import { isValidCredsAtom } from "@renderer/state/credentials";
import { visibleGamesAtom, currentGameIdAtom } from "@renderer/state/home";
import { useAtomValue, useSetAtom } from "jotai";
import { useCallback, useEffect, useState } from "react";
import { useParams, useNavigate, Navigate } from "react-router-dom";

import CloudDataCard from "@renderer/components/cloud/CloudDataCard";
import ConfirmModal from "@renderer/components/common/ConfirmModal";
import SyncConflictModal from "@renderer/components/cloud/SyncConflictModal";
import SyncStatusModal from "@renderer/components/cloud/SyncStatusModal";
import UntrackedDeleteModal from "@renderer/components/cloud/UntrackedDeleteModal";
import GameInfo from "@renderer/components/game/GameInfo";
import GameFormModal from "@renderer/components/game/GameModal";
import MemoCard from "@renderer/components/memo/MemoCard";
import PlaySessionManagementModal from "@renderer/components/game/PlaySessionManagementModal";
import PlaySessionModal from "@renderer/components/game/PlaySessionModal";
import PlayStatistics from "@renderer/components/game/PlayStatistics";

import { useGameEdit } from "@renderer/hooks/useGameEdit";
import { useGameSaveData } from "@renderer/hooks/useGameSaveData";
import { useCloudSync } from "@renderer/hooks/useCloudSync";
import { useOfflineMode } from "@renderer/hooks/useOfflineMode";
import { useToastHandler } from "@renderer/hooks/useToastHandler";
import { useTimeFormat } from "@renderer/hooks/useTimeFormat";
import { useValidateCreds } from "@renderer/hooks/useValidCreds";

import { logger } from "@renderer/utils/logger";
import { buildSaveSyncMessage } from "@renderer/utils/saveSyncMessage";

import type { GameType } from "src/types/game";
import type { SyncMetaSnapshot, SyncStatusDetail } from "src/wailsBridge";

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
  const [pendingLaunchAfterConflict, setPendingLaunchAfterConflict] = useState(false);
  const [conflictMeta, setConflictMeta] = useState<{
    localMeta?: SyncMetaSnapshot;
    remoteMeta?: SyncMetaSnapshot;
  } | null>(null);
  const [isResolvingConflict, setIsResolvingConflict] = useState(false);
  const [isSyncing, setIsSyncing] = useState(false);
  // syncStatusDetail が null のあいだは確認モーダルを出さない。
  const [syncStatusDetail, setSyncStatusDetail] = useState<SyncStatusDetail | null>(null);
  const [isSyncActionRunning, setIsSyncActionRunning] = useState(false);
  // Pull が Applied=false を返したときの削除候補。確認前はローカル無変更。
  const [untrackedDeletes, setUntrackedDeletes] = useState<string[] | null>(null);
  // 確認後に pull か resolveRemote のどちらを deleteUntracked=true で再実行するか。
  const [untrackedConfirmKind, setUntrackedConfirmKind] = useState<"pull" | "resolveRemote" | null>(
    null,
  );
  const [isDeletingUntracked, setIsDeletingUntracked] = useState(false);
  const { showToast } = useToastHandler();
  const { isOfflineMode, checkNetworkFeature } = useOfflineMode();
  const { getStatus, push, pull, resolveConflict } = useCloudSync(isOfflineMode);
  const { formatDateWithTime } = useTimeFormat();

  useEffect(() => {
    if (!id) return;

    const fetchGame = async (): Promise<void> => {
      setIsLoadingGame(true);
      try {
        // 一覧キャッシュにあれば DB 往復を避ける。
        const existingGame = visibleGames.find((g) => g.id === id);
        if (existingGame) {
          setGame(existingGame);
          setCurrentGameId(id);
          setIsLoadingGame(false);
          return;
        }

        // 直リンクやフィルタ外は visibleGames に無いので DB から取る。
        const fetchedGame = await window.api.database.getGameById(id);
        if (fetchedGame) {
          const transformedGame = fetchedGame as GameType;
          setGame(transformedGame);
          setCurrentGameId(id);
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

  // 詳細離脱後に PlayStatusBar が古い currentGameId を見ないようクリア。
  useEffect(() => {
    return () => {
      setCurrentGameId(null);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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

  const handleUploadSaveData = useCallback(async (): Promise<void> => {
    if (!checkNetworkFeature("セーブデータアップロード")) {
      return;
    }
    if (game) {
      await uploadSaveData(game);
    }
  }, [game, uploadSaveData, checkNetworkFeature]);

  const handleDownloadSaveData = useCallback(async (): Promise<boolean> => {
    if (!checkNetworkFeature("セーブデータダウンロード")) {
      return false;
    }
    if (!game) {
      return false;
    }
    return downloadSaveData(game);
  }, [game, downloadSaveData, checkNetworkFeature]);

  const buildSyncMessage = useCallback(
    (
      title: string,
      localUpdatedAt: Date | string | number | null | undefined,
      cloudUpdatedAt: Date | string | number | null | undefined,
    ) => buildSaveSyncMessage(formatDateWithTime, title, localUpdatedAt, cloudUpdatedAt),
    [formatDateWithTime],
  );

  const handleOpenPlaySessionModal = (): void => {
    setIsPlaySessionModalOpen(true);
  };

  const handleClosePlaySessionModal = (): void => {
    setIsPlaySessionModalOpen(false);
  };

  const refreshGameData = useCallback(async () => {
    if (!game?.id) return;

    try {
      const updatedGame = await window.api.database.getGameById(game.id);

      if (updatedGame) {
        const transformedGame = updatedGame as GameType;
        setGame(transformedGame);
        setVisibleGames((prev) => prev.map((g) => (g.id === game.id ? transformedGame : g)));
      }
      // 子が props 同一でも再マウント／再取得するよう refreshKey を進める。
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
        const statusResult = await getStatus(game.id);
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
          const op = await push(game.id);
          if (!op.ok) {
            if (showResult) showToast(op.message || "アップロードに失敗しました", "error");
            return false;
          }
          if (showResult) showToast("クラウドにアップロードしました", "success");
          return true;
        }
        if (status === "pull_needed") {
          const op = await pull(game.id);
          if (!op.ok) {
            if (showResult) showToast(op.message || "ダウンロードに失敗しました", "error");
            return false;
          }
          if (op.ok && op.applied === false) {
            // untracked 削除確認が必要（ここまでローカル無変更）。
            if (showResult) {
              setUntrackedDeletes(op.untrackedDeletes ?? []);
              setUntrackedConfirmKind("pull");
            }
            return false;
          }
          if (showResult) showToast("クラウドからダウンロードしました", "success");
          return true;
        }
        // conflict は自動解決せず専用モーダルへ。
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
    [game, isOfflineMode, showToast, getStatus, push, pull],
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
    const statusResult = await getStatus(game.id);
    if (!statusResult.success || !statusResult.data) {
      await launchGameDirect();
      return;
    }
    const { status, remoteMeta } = statusResult.data;
    if (status === "conflict") {
      // conflict を pull 確認に流すとローカルを黙って上書きしてしまう。
      setConflictMeta({
        localMeta: statusResult.data.localMeta,
        remoteMeta: statusResult.data.remoteMeta,
      });
      setPendingLaunchAfterConflict(true);
      setIsConflictModalOpen(true);
      return;
    }
    if (status === "pull_needed") {
      setSaveSyncMessage(
        buildSyncMessage(game.title, game.localSaveHashUpdatedAt, remoteMeta?.createdAt ?? null),
      );
      setIsDownloadConfirmOpen(true);
      return;
    }
    await launchGameDirect();
  }, [game, isOfflineMode, isValidCreds, launchGameDirect, getStatus, buildSyncMessage]);

  const handleDownloadAndLaunch = useCallback(async (): Promise<void> => {
    setIsDownloadConfirmOpen(false);
    setSaveSyncMessage("");
    const downloaded = await handleDownloadSaveData();
    if (!downloaded) {
      return;
    }
    await handleLaunchGame();
  }, [handleDownloadSaveData, handleLaunchGame]);

  const handleSkipDownloadAndLaunch = useCallback(async (): Promise<void> => {
    setIsDownloadConfirmOpen(false);
    setSaveSyncMessage("");
    // スキップ時は同期せず起動する（launchGameDirect は裏で sync するため使わない）。
    await handleLaunchGame();
  }, [handleLaunchGame]);

  const handleResolveConflict = useCallback(
    async (useLocal: boolean): Promise<void> => {
      if (!game) return;
      setIsResolvingConflict(true);
      try {
        const op = await resolveConflict(game.id, useLocal);
        if (op.ok) {
          if (!useLocal && op.applied === false) {
            // リモート採用でも untracked 削除確認が必要（ローカル無変更）。
            setIsConflictModalOpen(false);
            setConflictMeta(null);
            setUntrackedDeletes(op.untrackedDeletes ?? []);
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
          if (pendingLaunchAfterConflict) {
            setPendingLaunchAfterConflict(false);
            await handleLaunchGame();
          }
        } else {
          showToast(op.message || "コンフリクト解決に失敗しました", "error");
        }
      } catch {
        showToast("コンフリクト解決に失敗しました", "error");
      } finally {
        setIsResolvingConflict(false);
      }
    },
    [game, showToast, resolveConflict, pendingLaunchAfterConflict, handleLaunchGame],
  );

  // 確認後に deleteUntracked=true で再実行する（未確認のまま消さない）。
  const handleConfirmUntrackedDelete = useCallback(async (): Promise<void> => {
    if (!game || !untrackedConfirmKind) return;
    setIsDeletingUntracked(true);
    try {
      const op =
        untrackedConfirmKind === "pull"
          ? await pull(game.id, true)
          : await resolveConflict(game.id, false, true);
      if (op.ok && op.applied) {
        showToast("クラウドからダウンロードしました", "success");
        setUntrackedDeletes(null);
        setUntrackedConfirmKind(null);
        setRefreshKey((k) => k + 1);
      } else {
        showToast((!op.ok && op.message) || "ダウンロードに失敗しました", "error");
      }
    } catch {
      showToast("ダウンロードに失敗しました", "error");
    } finally {
      setIsDeletingUntracked(false);
    }
  }, [game, untrackedConfirmKind, showToast, pull, resolveConflict]);

  const handleCancelUntrackedDelete = useCallback((): void => {
    setUntrackedDeletes(null);
    setUntrackedConfirmKind(null);
  }, []);

  // 「同期確認」は状態を取得して表示するだけにとどめ、
  // 実際のアップロード/ダウンロードはモーダル内でユーザーが選んで実行する。
  const handleSyncCheck = useCallback(async (): Promise<void> => {
    if (!game) return;
    if (!checkNetworkFeature("同期確認")) return;
    setIsSyncing(true);
    try {
      const statusResult = await getStatus(game.id);
      if (!statusResult.success || !statusResult.data) {
        showToast(
          (!statusResult.success && statusResult.message) || "同期状態の取得に失敗しました",
          "error",
        );
        return;
      }
      const detail = statusResult.data;
      if (detail.status === "conflict") {
        // 競合を Status モーダルに載せると上書き操作と混ざるので専用へ。
        setConflictMeta({ localMeta: detail.localMeta, remoteMeta: detail.remoteMeta });
        setIsConflictModalOpen(true);
        return;
      }
      setSyncStatusDetail(detail);
    } catch (error) {
      logger.error("同期状態の取得エラー:", {
        component: "GameDetail",
        function: "handleSyncCheck",
        data: error,
      });
      showToast("同期状態の取得に失敗しました", "error");
    } finally {
      setIsSyncing(false);
    }
  }, [game, checkNetworkFeature, getStatus, showToast]);

  const handleSyncUpload = useCallback(async (): Promise<void> => {
    if (!game) return;
    setIsSyncActionRunning(true);
    try {
      const op = await push(game.id);
      if (op.ok) {
        showToast("クラウドにアップロードしました", "success");
        setSyncStatusDetail(null);
        await refreshGameData();
      } else {
        showToast(op.message || "アップロードに失敗しました", "error");
      }
    } finally {
      setIsSyncActionRunning(false);
    }
  }, [game, push, showToast, refreshGameData]);

  const handleSyncDownload = useCallback(async (): Promise<void> => {
    if (!game) return;
    setIsSyncActionRunning(true);
    try {
      const op = await pull(game.id);
      if (op.ok && op.applied === false) {
        // untracked 削除確認が必要（ここまでローカル無変更）。
        setSyncStatusDetail(null);
        setUntrackedDeletes(op.untrackedDeletes ?? []);
        setUntrackedConfirmKind("pull");
        return;
      }
      if (op.ok) {
        showToast("クラウドからダウンロードしました", "success");
        setSyncStatusDetail(null);
        await refreshGameData();
      } else {
        showToast(op.message || "ダウンロードに失敗しました", "error");
      }
    } finally {
      setIsSyncActionRunning(false);
    }
  }, [game, pull, showToast, refreshGameData]);

  const handleStatusChange = useCallback(
    async (newStatus: "unplayed" | "playing" | "played"): Promise<void> => {
      if (!game) return;

      setIsUpdatingStatus(true);
      try {
        const result = await window.api.database.updatePlayStatus(game.id, newStatus);

        if (result.success) {
          showToast("プレイステータスを更新しました", "success");
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

  if (isLoadingGame) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="loading loading-spinner loading-lg"></div>
      </div>
    );
  }

  // 取得完了後に無いときだけ戻す（ロード中の誤 Navigate を防ぐ）。
  if (!game) {
    return <Navigate to="/" replace />;
  }

  return (
    <div className="px-6 py-6">
      <div className="mx-auto max-w-6xl space-y-5">
        <div>
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

        <div>
          <PlayStatistics
            game={game}
            refreshKey={refreshKey}
            onAddPlaySession={handleOpenPlaySessionModal}
            onOpenProcessManagement={() => setIsProcessModalOpen(true)}
          />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
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

          <MemoCard gameId={game.id} />
        </div>
      </div>

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

      <ConfirmModal
        id="delete-game-modal"
        isOpen={isDeleteModalOpen}
        message={`${game.title} を削除しますか？\nこの操作は取り消せません`}
        cancelText="キャンセル"
        confirmText="削除する"
        onConfirm={handleDeleteGame}
        onCancel={closeDelete}
      />

      <GameFormModal
        mode="edit"
        initialData={editData}
        isOpen={isEditModalOpen}
        onSubmit={handleUpdateGame}
        onClose={closeEdit}
        onClosed={onEditClosed}
      />

      <PlaySessionModal
        isOpen={isPlaySessionModalOpen}
        onClose={handleClosePlaySessionModal}
        onSubmit={handleAddPlaySession}
        gameTitle={game.title}
      />

      {/* 同期状態の確認（アップロード/ダウンロードはここで選択して実行） */}
      {syncStatusDetail && (
        <SyncStatusModal
          isOpen
          onClose={() => setSyncStatusDetail(null)}
          gameTitle={game.title}
          status={syncStatusDetail.status}
          localMeta={syncStatusDetail.localMeta}
          remoteMeta={syncStatusDetail.remoteMeta}
          hasSaveFolder={!!game.saveFolderPath}
          isProcessing={isSyncActionRunning}
          onUpload={handleSyncUpload}
          onDownload={handleSyncDownload}
        />
      )}

      {/* コンフリクト解決 */}
      <SyncConflictModal
        isOpen={isConflictModalOpen}
        onClose={() => {
          setIsConflictModalOpen(false);
          setConflictMeta(null);
          setPendingLaunchAfterConflict(false);
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
