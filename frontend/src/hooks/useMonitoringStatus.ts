/**
 * @fileoverview プレイ状況監視のカスタムフック
 *
 * プレイ状況の取得・更新と確認モーダル用の状態管理を提供します。
 */

import { useCallback, useEffect, useMemo, useState } from "react";

import { logger } from "@renderer/utils/logger";

import type { MonitoringGameStatus } from "src/types/game";

type UseMonitoringStatusResult = {
  monitoringGames: MonitoringGameStatus[];
  activeGames: MonitoringGameStatus[];
  pendingConfirmationGame: MonitoringGameStatus | null;
  pendingResumeGame: MonitoringGameStatus | null;
  setPendingConfirmationGame: (game: MonitoringGameStatus | null) => void;
  setPendingResumeGame: (game: MonitoringGameStatus | null) => void;
  updateMonitoringStatus: () => Promise<void>;
};

export function useMonitoringStatus(autoTracking: boolean): UseMonitoringStatusResult {
  const [monitoringGames, setMonitoringGames] = useState<MonitoringGameStatus[]>([]);
  const [pendingConfirmationGame, setPendingConfirmationGame] =
    useState<MonitoringGameStatus | null>(null);
  const [pendingResumeGame, setPendingResumeGame] = useState<MonitoringGameStatus | null>(null);

  const updateMonitoringStatus = useCallback(async (): Promise<void> => {
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
      const resumePending = status.find((game) => game.needsResume && game.isPaused);
      if (resumePending && !pendingResumeGame) {
        setPendingResumeGame(resumePending);
      }
    } catch (error) {
      logger.error("監視状況の取得に失敗しました:", {
        component: "useMonitoringStatus",
        function: "updateMonitoringStatus",
        data: error,
      });
    }
  }, [autoTracking, pendingConfirmationGame, pendingResumeGame]);

  useEffect(() => {
    const interval = setInterval(() => {
      void updateMonitoringStatus();
    }, 1000);

    return () => clearInterval(interval);
  }, [updateMonitoringStatus]);

  useEffect(() => {
    const timer = setTimeout(() => {
      void updateMonitoringStatus();
    }, 1000);

    return () => clearTimeout(timer);
  }, [updateMonitoringStatus]);

  const activeGames = useMemo(
    () =>
      monitoringGames.filter((game) => game.isPlaying || game.isPaused || game.needsConfirmation),
    [monitoringGames],
  );

  return {
    monitoringGames,
    activeGames,
    pendingConfirmationGame,
    pendingResumeGame,
    setPendingConfirmationGame,
    setPendingResumeGame,
    updateMonitoringStatus,
  };
}
