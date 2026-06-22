/**
 * @fileoverview プロセス監視ブリッジ。
 */

import {
  GetMonitoringStatus,
  GetProcessSnapshot,
  PauseMonitoringSession,
  ResumeMonitoringSession,
  EndMonitoringSession,
} from "../../wailsjs/go/app/App";
import { toApiResultVoid } from "./helpers";
import type { MonitoringGameStatus } from "src/types/game";
import type { WindowApi } from "./types";

export function createProcessMonitorBridge(): WindowApi["processMonitor"] {
  return {
    getMonitoringStatus: async () => {
      const result = await GetMonitoringStatus();
      if (!result.success) {
        return [];
      }
      if (Array.isArray(result.data)) {
        return result.data as MonitoringGameStatus[];
      }
      return [];
    },
    getProcessSnapshot: async () => {
      const result = await GetProcessSnapshot();
      if (!result.success || !result.data) {
        return { source: "error", items: [] };
      }
      return result.data as {
        source: string;
        items: Array<{
          name: string;
          pid: number;
          cmd: string;
          normalizedName: string;
          normalizedCmd: string;
        }>;
      };
    },
    pauseSession: async (gameId) => toApiResultVoid(await PauseMonitoringSession(gameId), "エラー"),
    resumeSession: async (gameId) =>
      toApiResultVoid(await ResumeMonitoringSession(gameId), "エラー"),
    endSession: async (gameId) => toApiResultVoid(await EndMonitoringSession(gameId), "エラー"),
  };
}
