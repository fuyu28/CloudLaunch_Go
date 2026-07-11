/**
 * @fileoverview クラウド同期操作の共通フック
 *
 * window.api.cloudSync の各メソッドを useCallback でラップし、
 * 正規化された CloudSyncOp 型を返す。
 */

import { useCallback } from "react";
import type { ApiResult } from "src/types/result";
import type { SyncStatusDetail, SyncProgressEvent } from "src/wailsBridge";

export type CloudSyncOp = {
  ok: boolean;
  applied?: boolean;
  untrackedDeletes?: string[];
  message?: string;
};

// isOfflineMode は将来の拡張のために受け取るが、フック本体では使用しない。
// オフラインモードガードは呼び出し元で行う。
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export function useCloudSync(_isOfflineMode: boolean) {
  const getStatus = useCallback(
    (gameId: string): Promise<ApiResult<SyncStatusDetail>> => window.api.cloudSync.status(gameId),
    [],
  );

  const push = useCallback(async (gameId: string): Promise<CloudSyncOp> => {
    const result = await window.api.cloudSync.push(gameId);
    return {
      ok: result.success,
      message: result.success ? undefined : result.message,
    };
  }, []);

  const pull = useCallback(
    async (gameId: string, deleteUntracked?: boolean): Promise<CloudSyncOp> => {
      const result = await window.api.cloudSync.pull(gameId, deleteUntracked);
      return {
        ok: result.success,
        applied: result.success ? result.data?.applied : undefined,
        untrackedDeletes: result.success ? result.data?.untrackedDeletes : undefined,
        message: result.success ? undefined : result.message,
      };
    },
    [],
  );

  const resolveConflict = useCallback(
    async (gameId: string, useLocal: boolean, deleteUntracked?: boolean): Promise<CloudSyncOp> => {
      const result = await window.api.cloudSync.resolveConflict(gameId, useLocal, deleteUntracked);
      return {
        ok: result.success,
        applied: result.success ? result.data?.applied : undefined,
        untrackedDeletes: result.success ? result.data?.untrackedDeletes : undefined,
        message: result.success ? undefined : result.message,
      };
    },
    [],
  );

  const subscribeProgress = useCallback(
    (cb: (e: SyncProgressEvent) => void): (() => void) => window.api.cloudSync.onProgress(cb),
    [],
  );

  return { getStatus, push, pull, resolveConflict, subscribeProgress };
}
