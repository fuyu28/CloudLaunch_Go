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
import { EventsOn } from "../../wailsjs/runtime/runtime";
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
      // Wails v2 の EventsOn は登録解除用の関数を返すので、そのまま返して
      // このリスナーだけを解除する（EventsOff は同名の全リスナーを消してしまう）
      return EventsOn("sync:progress", callback);
    },
  };
}
