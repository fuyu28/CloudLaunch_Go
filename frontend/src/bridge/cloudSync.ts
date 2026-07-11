/**
 * @fileoverview セーブ同期（status/push/pull/conflict）ブリッジ。
 *
 * savesDiffer の互換フォールバックと EventsOn の解除関数返却が非自明な箇所。
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
        savesDiffer?: boolean;
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
          // 旧クライアント / never_synced 早期 return では savesDiffer が欠ける。
          // 未定義を「差分あり」扱いすると誤ったアップロード確認が出るため false に倒す。
          savesDiffer: raw.savesDiffer ?? false,
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
      // EventsOff("sync:progress") は同名リスナーを全削除する。
      // EventsOn の戻り値で当該登録だけ解除する。
      return EventsOn("sync:progress", callback);
    },
  };
}
