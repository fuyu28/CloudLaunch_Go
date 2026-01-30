/**
 * @fileoverview プレイ状況バーコンポーネント
 *
 * このコンポーネントは、アプリケーション画面下部に現在のプレイ状況を表示します。
 *
 * 主な機能：
 * - 現在プレイ中のゲームの表示
 * - プレイ経過時間の表示
 * - プロセス監視の状態表示
 *
 * 使用例：
 * ```tsx
 * <PlayStatusBar />
 * ```
 */

import { autoTrackingAtom } from "@renderer/state/settings";
import { isValidCredsAtom } from "@renderer/state/credentials";
import { useAtom, useAtomValue } from "jotai";
import React, { useEffect, useMemo, useState } from "react";
import { FaClock, FaGamepad } from "react-icons/fa";

import ConfirmModal from "@renderer/components/ConfirmModal";
import BaseModal from "@renderer/components/BaseModal";

import { useOfflineMode } from "@renderer/hooks/useOfflineMode";
import { useTimeFormat } from "@renderer/hooks/useTimeFormat";
import { useToastHandler } from "@renderer/hooks/useToastHandler";

import { logger } from "@renderer/utils/logger";
import { createRemotePath } from "@renderer/utils";

import type { MonitoringGameStatus } from "src/types/game";

/**
 * プレイ状況バーコンポーネント
 *
 * アプリケーション画面下部に表示され、
 * 現在のプレイ状況を表示します。
 *
 * @returns プレイ状況バー要素
 */
export function PlayStatusBar(): React.JSX.Element {
  const [autoTracking] = useAtom(autoTrackingAtom);
  const isValidCreds = useAtomValue(isValidCredsAtom);
  const [monitoringGames, setMonitoringGames] = useState<MonitoringGameStatus[]>([]);
  const [pendingConfirmationGame, setPendingConfirmationGame] =
    useState<MonitoringGameStatus | null>(null);
  const [pendingUpload, setPendingUpload] = useState<{
    gameId: string;
    gameTitle: string;
    saveFolderPath: string;
    localHash: string;
  } | null>(null);
  const [, setCurrentTime] = useState<Date>(new Date());
  const { formatShort } = useTimeFormat();
  const { isOfflineMode } = useOfflineMode();
  const { showToast } = useToastHandler();

  // 監視状況を更新
  const updateMonitoringStatus = React.useCallback(async (): Promise<void> => {
    // 自動ゲーム検出がOFFの場合は更新しない
    if (!autoTracking) {
      return;
    }

    try {
      const status = await window.api.processMonitor.getMonitoringStatus();
      setMonitoringGames(status);
      const pending = status.find((game) => game.needsConfirmation);
      if (pending && !pendingConfirmationGame) {
        setPendingConfirmationGame(pending);
      }
    } catch (error) {
      logger.error("監視状況の取得に失敗しました:", {
        component: "PlayStatusBar",
        function: "unknown",
        data: error,
      });
    }
  }, [autoTracking]);

  // 時間更新とステータス更新
  useEffect(() => {
    const interval = setInterval(() => {
      setCurrentTime(new Date());
      updateMonitoringStatus();
    }, 1000);

    return () => clearInterval(interval);
  }, [updateMonitoringStatus]);

  // 初期化
  useEffect(() => {
    // 少し遅延させて監視状態を取得（メインプロセスの初期化を待つ）
    const timer = setTimeout(() => {
      updateMonitoringStatus();
    }, 1000);

    return () => clearTimeout(timer);
  }, [updateMonitoringStatus]);

  // 自動ゲーム検出がOFFの場合は非表示
  if (!autoTracking) {
    return <></>;
  }

  const activeGames = useMemo(
    () =>
      monitoringGames.filter((game) => game.isPlaying || game.isPaused || game.needsConfirmation),
    [monitoringGames],
  );
  const hasActiveGames = activeGames.length > 0;

  const handlePause = async (gameId: string): Promise<void> => {
    const result = await window.api.processMonitor.pauseSession(gameId);
    if (!result.success) {
      logger.error("セッション中断に失敗しました:", {
        component: "PlayStatusBar",
        function: "handlePause",
        data: result.message,
      });
    }
    await updateMonitoringStatus();
  };

  const handleResume = async (gameId: string): Promise<void> => {
    const result = await window.api.processMonitor.resumeSession(gameId);
    if (!result.success) {
      logger.error("セッション再開に失敗しました:", {
        component: "PlayStatusBar",
        function: "handleResume",
        data: result.message,
      });
    }
    await updateMonitoringStatus();
  };

  const checkUploadPrompt = React.useCallback(
    async (gameId: string): Promise<void> => {
      if (isOfflineMode || !isValidCreds) {
        return;
      }
      const game = await window.api.database.getGameById(gameId);
      if (!game || !game.saveFolderPath) {
        return;
      }
      const localHashResult = await window.api.saveData.hash.computeLocalHash(game.saveFolderPath);
      if (!localHashResult.success || !localHashResult.data) {
        return;
      }
      const cloudHashResult = await window.api.saveData.hash.getCloudHash(gameId);
      const cloudHash = cloudHashResult.success ? cloudHashResult.data?.hash : null;
      if (!cloudHash || cloudHash !== localHashResult.data) {
        setPendingUpload({
          gameId,
          gameTitle: game.title,
          saveFolderPath: game.saveFolderPath,
          localHash: localHashResult.data,
        });
      }
    },
    [isOfflineMode, isValidCreds],
  );

  const handleEnd = async (gameId: string): Promise<void> => {
    const result = await window.api.processMonitor.endSession(gameId);
    if (!result.success) {
      logger.error("セッション終了に失敗しました:", {
        component: "PlayStatusBar",
        function: "handleEnd",
        data: result.message,
      });
    }
    setPendingConfirmationGame(null);
    await updateMonitoringStatus();
    await checkUploadPrompt(gameId);
  };

  const handleKeepPaused = async (gameId: string): Promise<void> => {
    const result = await window.api.processMonitor.pauseSession(gameId);
    if (!result.success) {
      logger.error("セッション中断に失敗しました:", {
        component: "PlayStatusBar",
        function: "handleKeepPaused",
        data: result.message,
      });
    }
    setPendingConfirmationGame(null);
    await updateMonitoringStatus();
  };

  const handleUploadAfterEnd = async (): Promise<void> => {
    if (!pendingUpload) return;
    const remotePath = createRemotePath(pendingUpload.gameId);
    const result = await window.api.saveData.upload.uploadSaveDataFolder(
      pendingUpload.saveFolderPath,
      remotePath,
    );
    if (result.success) {
      await window.api.saveData.hash.saveCloudHash(pendingUpload.gameId, pendingUpload.localHash);
      showToast("セーブデータをクラウドにアップロードしました", "success");
    } else {
      showToast(result.message || "セーブデータのアップロードに失敗しました", "error");
    }
    setPendingUpload(null);
  };

  const handleSkipUploadAfterEnd = (): void => {
    setPendingUpload(null);
  };

  return (
    <>
      <div className="bg-base-300 border-t border-base-content/10 px-4 py-1 h-12">
        <div className="flex items-center justify-between h-full">
          {/* 左側：プレイ状況 */}
          <div className="flex items-center gap-3">
            {hasActiveGames ? (
              <>
                <FaGamepad className="text-primary text-sm" />
                <div className="flex flex-col justify-center">
                  <div className="text-sm font-medium leading-tight">
                    プレイ中: {activeGames.map((game) => game.gameTitle).join(", ")}
                  </div>
                  <div className="text-xs text-base-content/70 leading-tight">
                    {activeGames.map((game) => (
                      <span key={game.gameId} className="mr-4 inline-flex items-center gap-2">
                        <span>
                          {game.exeName}: {formatShort(game.playTime)}
                          {game.needsConfirmation && "（確認待ち）"}
                          {game.isPaused && !game.needsConfirmation && "（中断中）"}
                        </span>
                        {!game.needsConfirmation && (
                          <button
                            className="btn btn-xs btn-ghost"
                            onClick={() =>
                              game.isPaused ? handleResume(game.gameId) : handlePause(game.gameId)
                            }
                          >
                            {game.isPaused ? "再開" : "中断"}
                          </button>
                        )}
                      </span>
                    ))}
                  </div>
                </div>
              </>
            ) : (
              <>
                <FaClock className="text-base-content/50 text-sm" />
                <div className="text-sm text-base-content/70">プレイ中のゲームはありません</div>
              </>
            )}
          </div>
        </div>
      </div>
      <BaseModal
        isOpen={!!pendingConfirmationGame}
        onClose={() => setPendingConfirmationGame(null)}
        title="プレイを終了しますか？"
        size="md"
        showCloseButton={false}
        closeOnClickOutside={false}
        closeOnEscape={false}
        footer={
          pendingConfirmationGame ? (
            <>
              <button
                className="btn btn-outline"
                onClick={() => handleKeepPaused(pendingConfirmationGame.gameId)}
              >
                No
              </button>
              <button
                className="btn btn-primary"
                onClick={() => handleEnd(pendingConfirmationGame.gameId)}
              >
                Yes
              </button>
            </>
          ) : undefined
        }
      >
        {pendingConfirmationGame && (
          <div className="text-sm">
            <div className="font-medium mb-2">{pendingConfirmationGame.gameTitle}</div>
            <p className="text-base-content/70">
              プロセスが見つかりません。セッションを終了しますか？
            </p>
          </div>
        )}
      </BaseModal>
      <ConfirmModal
        id="upload-save-after-session-modal"
        isOpen={!!pendingUpload}
        title="セーブデータの同期"
        message={
          pendingUpload
            ? `${pendingUpload.gameTitle} のセーブデータがクラウドと異なります。\nアップロードしますか？`
            : ""
        }
        cancelText="しない"
        confirmText="アップロードする"
        onConfirm={handleUploadAfterEnd}
        onCancel={handleSkipUploadAfterEnd}
      />
    </>
  );
}

export default PlayStatusBar;
