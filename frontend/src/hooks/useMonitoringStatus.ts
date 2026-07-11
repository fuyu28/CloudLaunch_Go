/**
 * @fileoverview プレイ状況監視のカスタムフック
 *
 * プレイ状況の取得・更新と確認モーダル用の状態管理を提供します。
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

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
  const [pendingConfirmationGameState, setPendingConfirmationGameState] =
    useState<MonitoringGameStatus | null>(null);
  const [pendingResumeGameState, setPendingResumeGameState] = useState<MonitoringGameStatus | null>(
    null,
  );
  const [isFocused, setIsFocused] = useState<boolean>(
    typeof document !== "undefined" ? document.visibilityState === "visible" : true,
  );
  const hasActiveGamesRef = useRef<boolean>(false);
  const backoffIndexRef = useRef<number>(0);
  const pollTimeoutRef = useRef<number | null>(null);
  // 再帰 setTimeout の内側から参照される updateMonitoringStatus が
  // 古い pending* を閉じ込んだままにならないよう、ref で最新値を参照する。
  // setTimeout 起動時点の state を見てしまうと、モーダル表示中でも別ゲームで
  // 再度モーダルを開いてしまう silent swap が発生する。
  const pendingConfirmationGameRef = useRef<MonitoringGameStatus | null>(null);
  const pendingResumeGameRef = useRef<MonitoringGameStatus | null>(null);

  const setPendingConfirmationGame = useCallback((game: MonitoringGameStatus | null): void => {
    pendingConfirmationGameRef.current = game;
    setPendingConfirmationGameState(game);
  }, []);
  const setPendingResumeGame = useCallback((game: MonitoringGameStatus | null): void => {
    pendingResumeGameRef.current = game;
    setPendingResumeGameState(game);
  }, []);

  const clearPollTimeout = useCallback((): void => {
    if (pollTimeoutRef.current !== null) {
      window.clearTimeout(pollTimeoutRef.current);
      pollTimeoutRef.current = null;
    }
  }, []);

  const scheduleNextPoll = useCallback(
    (delayMs: number, handler: () => void): void => {
      clearPollTimeout();
      pollTimeoutRef.current = window.setTimeout(() => {
        handler();
      }, delayMs);
    },
    [clearPollTimeout],
  );

  const resetBackoff = useCallback((): void => {
    backoffIndexRef.current = 0;
  }, []);

  const updateMonitoringStatus = useCallback(async (): Promise<void> => {
    if (!autoTracking) {
      return;
    }

    try {
      const status = await window.api.processMonitor.getMonitoringStatus();
      setMonitoringGames(status);
      hasActiveGamesRef.current = status.some(
        (game) => game.isPlaying || game.isPaused || game.needsConfirmation,
      );
      const pending = status.find((game) => game.needsConfirmation);
      if (pending && !pendingConfirmationGameRef.current) {
        setPendingConfirmationGame(pending);
      }
      const resumePending = status.find((game) => game.needsResume && game.isPaused);
      if (resumePending && !pendingResumeGameRef.current) {
        setPendingResumeGame(resumePending);
      }
    } catch (error) {
      logger.error("監視状況の取得に失敗しました:", {
        component: "useMonitoringStatus",
        function: "updateMonitoringStatus",
        data: error,
      });
    }

    if (!autoTracking) {
      return;
    }
    const shouldFastPoll = isFocused || hasActiveGamesRef.current;
    if (shouldFastPoll) {
      resetBackoff();
      scheduleNextPoll(1000, () => void updateMonitoringStatus());
      return;
    }

    const backoffDelays = [3000, 5000, 8000, 10000];
    const delay = backoffDelays[Math.min(backoffIndexRef.current, backoffDelays.length - 1)];
    backoffIndexRef.current += 1;
    scheduleNextPoll(delay, () => void updateMonitoringStatus());
  }, [
    autoTracking,
    isFocused,
    resetBackoff,
    scheduleNextPoll,
    setPendingConfirmationGame,
    setPendingResumeGame,
  ]);

  useEffect(() => {
    if (!autoTracking) {
      clearPollTimeout();
      return;
    }
    resetBackoff();
    scheduleNextPoll(0, () => void updateMonitoringStatus());
    return () => {
      clearPollTimeout();
    };
  }, [autoTracking, clearPollTimeout, resetBackoff, scheduleNextPoll]);

  useEffect(() => {
    const handleFocus = (): void => {
      setIsFocused(true);
      resetBackoff();
      scheduleNextPoll(0, () => void updateMonitoringStatus());
    };
    const handleBlur = (): void => {
      setIsFocused(false);
    };
    const handleVisibility = (): void => {
      const visible = document.visibilityState === "visible";
      setIsFocused(visible);
      if (visible) {
        resetBackoff();
        scheduleNextPoll(0, () => void updateMonitoringStatus());
      }
    };

    window.addEventListener("focus", handleFocus);
    window.addEventListener("blur", handleBlur);
    document.addEventListener("visibilitychange", handleVisibility);

    return () => {
      window.removeEventListener("focus", handleFocus);
      window.removeEventListener("blur", handleBlur);
      document.removeEventListener("visibilitychange", handleVisibility);
    };
  }, [resetBackoff, scheduleNextPoll]);

  const activeGames = useMemo(
    () =>
      monitoringGames.filter((game) => game.isPlaying || game.isPaused || game.needsConfirmation),
    [monitoringGames],
  );

  return {
    monitoringGames,
    activeGames,
    pendingConfirmationGame: pendingConfirmationGameState,
    pendingResumeGame: pendingResumeGameState,
    setPendingConfirmationGame,
    setPendingResumeGame,
    updateMonitoringStatus,
  };
}
