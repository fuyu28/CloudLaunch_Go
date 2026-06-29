/**
 * @fileoverview クラウド同期ブリッジ。
 */

import {
  SyncStatus,
  PushSync,
  PullSync,
  ResolveConflict,
  DeleteGameFromCloud,
} from "../../wailsjs/go/app/App";
import { EventsOn, EventsOff } from "../../wailsjs/runtime/runtime";
import { toApiResultVoid } from "./helpers";
import type {
  SyncStatus as SyncStatusType,
  SyncMetaSnapshot,
  PullResult,
  SyncProgressEvent,
  WindowApi,
} from "./types";

export function createCloudSyncBridge(): WindowApi["cloudSync"] {
  return {
    status: async (gameId) => {
      const result = await SyncStatus(gameId);
      if (!result.success) {
        return { success: false, message: result.error?.message ?? "エラー" };
      }
      const raw = result.data as {
        status: SyncStatusType;
        localMeta?: {
          "game.json": string;
          "sessions.json": string;
          saves: string;
          deviceName: string;
          createdAt: string;
        };
        remoteMeta?: {
          "game.json": string;
          "sessions.json": string;
          saves: string;
          deviceName: string;
          createdAt: string;
        };
      };
      const normalizeMeta = (m?: typeof raw.localMeta): SyncMetaSnapshot | undefined =>
        m ? { ...m, createdAt: new Date(m.createdAt) } : undefined;
      return {
        success: true,
        data: {
          status: raw.status,
          localMeta: normalizeMeta(raw.localMeta),
          remoteMeta: normalizeMeta(raw.remoteMeta),
        },
      };
    },
    push: async (gameId) => toApiResultVoid(await PushSync(gameId)),
    pull: async (gameId, deleteUntracked = false) => {
      const result = await PullSync(gameId, deleteUntracked);
      return result.success
        ? { success: true, data: result.data as PullResult }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
    resolveConflict: async (gameId, useLocal, deleteUntracked = false) => {
      const result = await ResolveConflict(gameId, useLocal, deleteUntracked);
      return result.success
        ? { success: true, data: result.data as PullResult }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
    deleteFromCloud: async (gameId) => toApiResultVoid(await DeleteGameFromCloud(gameId)),
    onProgress: (callback: (event: SyncProgressEvent) => void) => {
      EventsOn("sync:progress", callback);
      return () => EventsOff("sync:progress");
    },
  };
}
