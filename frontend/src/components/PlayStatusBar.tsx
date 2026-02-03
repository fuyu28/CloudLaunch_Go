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
import { visibleGamesAtom } from "@renderer/state/home";
import { useAtom, useAtomValue } from "jotai";
import { FaClock, FaGamepad } from "react-icons/fa";

import ConfirmModal from "@renderer/components/ConfirmModal";
import BaseModal from "@renderer/components/BaseModal";

import { useOfflineMode } from "@renderer/hooks/useOfflineMode";
import { useTimeFormat } from "@renderer/hooks/useTimeFormat";
import { useToastHandler } from "@renderer/hooks/useToastHandler";
import { useMonitoringStatus } from "@renderer/hooks/useMonitoringStatus";
import { useUploadAfterSession } from "@renderer/hooks/useUploadAfterSession";

import { logger } from "@renderer/utils/logger";

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
  const [, setVisibleGames] = useAtom(visibleGamesAtom);
  const isValidCreds = useAtomValue(isValidCredsAtom);
  const { formatShort } = useTimeFormat();
  const { isOfflineMode } = useOfflineMode();
  const toastHandler = useToastHandler();
  const {
    activeGames,
    pendingConfirmationGame,
    pendingResumeGame,
    setPendingConfirmationGame,
    setPendingResumeGame,
    updateMonitoringStatus,
  } = useMonitoringStatus(autoTracking);
  const { pendingUpload, checkUploadPrompt, handleUploadAfterEnd, handleSkipUploadAfterEnd } =
    useUploadAfterSession(isOfflineMode, isValidCreds, toastHandler);

  // 自動ゲーム検出がOFFの場合は非表示
  if (!autoTracking) {
    return <></>;
  }

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

  const handleResumeConfirm = async (gameId: string): Promise<void> => {
    await handleResume(gameId);
    setPendingResumeGame(null);
  };

  const handleKeepPausedConfirm = async (gameId: string): Promise<void> => {
    const result = await window.api.processMonitor.pauseSession(gameId);
    if (!result.success) {
      logger.error("セッション中断に失敗しました:", {
        component: "PlayStatusBar",
        function: "handleKeepPausedConfirm",
        data: result.message,
      });
    }
    setPendingResumeGame(null);
    await updateMonitoringStatus();
  };

  const refreshGame = async (gameId: string): Promise<void> => {
    try {
      const updated = await window.api.database.getGameById(gameId);
      if (!updated) return;
      setVisibleGames((prev) => prev.map((game) => (game.id === gameId ? updated : game)));
    } catch (error) {
      logger.warn("ゲーム再取得に失敗しました:", {
        component: "PlayStatusBar",
        function: "refreshGame",
        data: error,
      });
    }
  };

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
    await refreshGame(gameId);
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

  return (
    <>
      <div className="bg-base-300 border-t border-base-content/10 px-4 py-1 h-12">
        <div className="flex items-center justify-between h-full">
          {/* 左側：プレイ状況 */}
          <div className="flex items-center gap-3">
            {hasActiveGames ? (
              <>
                <FaGamepad className="text-primary text-sm" />
                <div className="flex flex-1 items-center justify-between gap-4">
                  <div className="min-w-0">
                    <div className="text-sm font-medium leading-tight">
                      プレイ中: {activeGames.map((game) => game.gameTitle).join(", ")}
                    </div>
                    <div className="text-xs text-base-content/70 leading-tight truncate">
                      {activeGames.map((game) => (
                        <span key={game.gameId} className="mr-4">
                          {game.exeName}: {formatShort(game.playTime)}
                          {game.needsConfirmation && "（確認待ち）"}
                          {game.isPaused && !game.needsConfirmation && "（中断中）"}
                        </span>
                      ))}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {activeGames.map((game) => (
                      <div key={game.gameId} className="flex items-center gap-2">
                        {!game.needsConfirmation && !game.isPaused && (
                          <button
                            className="btn btn-sm btn-warning"
                            onClick={() => handlePause(game.gameId)}
                          >
                            中断
                          </button>
                        )}
                        {!game.needsConfirmation && game.isPaused && (
                          <>
                            <button
                              className="btn btn-sm btn-primary"
                              onClick={() => handleResume(game.gameId)}
                            >
                              再開
                            </button>
                            <button
                              className="btn btn-sm btn-error"
                              onClick={() => handleEnd(game.gameId)}
                            >
                              終了
                            </button>
                          </>
                        )}
                      </div>
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
      <ConfirmModal
        id="resume-session-modal"
        isOpen={!!pendingResumeGame}
        title="セッションを再開しますか？"
        message={
          pendingResumeGame
            ? `${pendingResumeGame.gameTitle} が起動されました。\nセッションを再開しますか？`
            : ""
        }
        cancelText="中断を維持"
        confirmText="再開する"
        onConfirm={() => pendingResumeGame && handleResumeConfirm(pendingResumeGame.gameId)}
        onCancel={() => pendingResumeGame && handleKeepPausedConfirm(pendingResumeGame.gameId)}
      />
    </>
  );
}

export default PlayStatusBar;
