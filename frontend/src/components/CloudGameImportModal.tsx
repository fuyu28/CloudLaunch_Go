/**
 * @fileoverview クラウドから既存ゲームを追加するモーダル
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "react-hot-toast";

import { useOfflineMode } from "@renderer/hooks/useOfflineMode";
import { useTimeFormat } from "@renderer/hooks/useTimeFormat";
import { logger } from "@renderer/utils/logger";

import { BaseModal } from "./BaseModal";
import CloudGameImportConflictModal from "./CloudGameImportConflictModal";

import type { CloudGameMetadata } from "src/types/cloud";
import type { GameType } from "src/types/game";

type CloudGameImportModalProps = {
  isOpen: boolean;
  onClose: () => void;
  localGames: GameType[];
  onImported?: () => Promise<void> | void;
};

type ConflictState = {
  cloudGame: CloudGameMetadata;
  localMatches: GameType[];
  remainingQueue: CloudGameMetadata[];
};

const normalizeTitle = (title: string): string => title.trim().toLowerCase();

export default function CloudGameImportModal({
  isOpen,
  onClose,
  localGames,
  onImported,
}: CloudGameImportModalProps): React.JSX.Element {
  const { checkNetworkFeature, isOfflineMode } = useOfflineMode();
  const { formatDateWithTime, formatSmart } = useTimeFormat();
  const [cloudGames, setCloudGames] = useState<CloudGameMetadata[]>([]);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [importing, setImporting] = useState(false);
  const [conflict, setConflict] = useState<ConflictState | null>(null);
  const [localCache, setLocalCache] = useState<GameType[]>(localGames);

  const localIds = useMemo(() => new Set(localCache.map((game) => game.id)), [localCache]);
  const localTitleMap = useMemo(() => {
    const map = new Map<string, GameType[]>();
    localCache.forEach((game) => {
      const key = normalizeTitle(game.title);
      const list = map.get(key) ?? [];
      list.push(game);
      map.set(key, list);
    });
    return map;
  }, [localCache]);

  const availableCloudGames = useMemo(
    () => cloudGames.filter((game) => !localIds.has(game.id)),
    [cloudGames, localIds],
  );

  const selectedGames = useMemo(
    () => availableCloudGames.filter((game) => selectedIds.has(game.id)),
    [availableCloudGames, selectedIds],
  );

  const fetchCloudGames = useCallback(async (): Promise<void> => {
    if (!checkNetworkFeature("クラウド同期")) {
      setCloudGames([]);
      setSelectedIds(new Set());
      return;
    }

    setLoading(true);
    try {
      const result = await window.api.cloudMetadata.loadCloudMetadata();
      if (!result.success || !result.data) {
        toast.error(result.message ?? "クラウドデータの取得に失敗しました");
        setCloudGames([]);
        return;
      }

      const sorted = [...(result.data.games ?? [])].sort((a, b) => {
        const aTime = new Date(a.updatedAt).getTime();
        const bTime = new Date(b.updatedAt).getTime();
        return bTime - aTime;
      });
      setCloudGames(sorted);
    } catch (error) {
      logger.error("クラウドメタ情報取得に失敗", {
        component: "CloudGameImportModal",
        function: "fetchCloudGames",
        data: error,
      });
      toast.error("クラウドデータの取得に失敗しました");
      setCloudGames([]);
    } finally {
      setLoading(false);
    }
  }, [checkNetworkFeature]);

  const fetchLocalGames = useCallback(async (): Promise<void> => {
    try {
      const games = await window.api.database.listGames("", "all", "title", "asc");
      setLocalCache(games as GameType[]);
    } catch (error) {
      logger.warn("ローカルゲーム一覧の取得に失敗", {
        component: "CloudGameImportModal",
        function: "fetchLocalGames",
        data: error,
      });
      setLocalCache(localGames);
    }
  }, [localGames]);

  useEffect(() => {
    if (!isOpen) {
      setSelectedIds(new Set());
      setCloudGames([]);
      setConflict(null);
      return;
    }

    setLocalCache(localGames);
    fetchLocalGames();
    fetchCloudGames();
  }, [fetchCloudGames, fetchLocalGames, isOpen, localGames]);

  const toggleSelection = (gameId: string): void => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(gameId)) {
        next.delete(gameId);
      } else {
        next.add(gameId);
      }
      return next;
    });
  };

  const toggleSelectAll = (): void => {
    setSelectedIds((prev) => {
      if (prev.size === availableCloudGames.length) {
        return new Set();
      }
      return new Set(availableCloudGames.map((game) => game.id));
    });
  };

  const importGame = useCallback(
    async (game: CloudGameMetadata): Promise<boolean> => {
      const result = await window.api.cloudSync.syncGame(game.id);
      if (!result.success) {
        toast.error(result.message ?? "クラウドゲームの追加に失敗しました");
        return false;
      }

      toast.success(`「${game.title}」を追加しました`);
      setCloudGames((prev) => prev.filter((item) => item.id !== game.id));
      setSelectedIds((prev) => {
        const next = new Set(prev);
        next.delete(game.id);
        return next;
      });

      if (onImported) {
        await onImported();
      }
      return true;
    },
    [onImported],
  );

  const processQueue = useCallback(
    async (queue: CloudGameMetadata[]): Promise<void> => {
      if (queue.length === 0) {
        setImporting(false);
        return;
      }

      const [current, ...rest] = queue;
      const conflicts = localTitleMap.get(normalizeTitle(current.title)) ?? [];
      if (conflicts.length > 0) {
        setConflict({ cloudGame: current, localMatches: conflicts, remainingQueue: rest });
        return;
      }

      const imported = await importGame(current);
      if (!imported) {
        setImporting(false);
        return;
      }

      await processQueue(rest);
    },
    [importGame, localTitleMap],
  );

  const handleStartImport = async (): Promise<void> => {
    if (selectedGames.length === 0) {
      toast.error("追加するゲームを選択してください");
      return;
    }

    setImporting(true);
    await processQueue(selectedGames);
  };

  const resolveConflict = async (mode: "duplicate" | "replace"): Promise<void> => {
    if (!conflict) return;

    setConflict(null);
    if (mode === "replace") {
      for (const localGame of conflict.localMatches) {
        const result = await window.api.database.deleteGame(localGame.id);
        if (!result.success) {
          toast.error(result.message ?? "ローカルゲームの削除に失敗しました");
          setImporting(false);
          return;
        }
      }
      if (onImported) {
        await onImported();
      }
    }

    const imported = await importGame(conflict.cloudGame);
    if (!imported) {
      setImporting(false);
      return;
    }

    await processQueue(conflict.remainingQueue);
  };

  const handleClose = (): void => {
    if (importing) {
      return;
    }
    onClose();
  };

  const footer = (
    <div className="flex flex-wrap justify-between gap-2 w-full">
      <button type="button" className="btn" onClick={handleClose} disabled={importing}>
        閉じる
      </button>
      <div className="flex gap-2">
        <button
          type="button"
          className="btn btn-outline"
          onClick={fetchCloudGames}
          disabled={loading || importing || isOfflineMode}
        >
          再読み込み
        </button>
        <button
          type="button"
          className="btn btn-primary"
          onClick={handleStartImport}
          disabled={loading || importing || isOfflineMode || selectedGames.length === 0}
        >
          {importing ? "追加中..." : `選択した${selectedGames.length}件を追加`}
        </button>
      </div>
    </div>
  );

  return (
    <>
      <BaseModal
        id="cloud-game-import-modal"
        isOpen={isOpen}
        onClose={handleClose}
        title="クラウドから既存ゲームを追加"
        size="xl"
        footer={footer}
      >
        <div className="space-y-4">
          {isOfflineMode && (
            <div className="alert alert-warning text-sm">
              オフラインモードではクラウドから追加できません。
            </div>
          )}
          <div className="flex items-center justify-between">
            <div className="text-sm text-base-content/70">
              {availableCloudGames.length}件のゲームがクラウドにあります
            </div>
            <button
              type="button"
              className="btn btn-ghost btn-sm"
              onClick={toggleSelectAll}
              disabled={availableCloudGames.length === 0}
            >
              {selectedIds.size === availableCloudGames.length ? "全解除" : "全選択"}
            </button>
          </div>

          {loading && <div className="text-sm text-base-content/60">読み込み中...</div>}

          {!loading && availableCloudGames.length === 0 && (
            <div className="text-sm text-base-content/60">
              追加できるクラウドゲームがありません。
            </div>
          )}

          {!loading && availableCloudGames.length > 0 && (
            <div className="space-y-2 max-h-[50vh] overflow-auto">
              {availableCloudGames.map((game) => {
                const conflictBadge = localTitleMap.has(normalizeTitle(game.title));
                return (
                  <label
                    key={game.id}
                    className="flex items-center gap-3 rounded-lg border border-base-300 bg-base-100 p-3 hover:bg-base-200"
                  >
                    <input
                      type="checkbox"
                      className="checkbox checkbox-primary"
                      checked={selectedIds.has(game.id)}
                      onChange={() => toggleSelection(game.id)}
                      disabled={importing}
                    />
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <div className="font-medium">{game.title}</div>
                        {conflictBadge && (
                          <span className="badge badge-warning badge-sm">競合</span>
                        )}
                      </div>
                      <div className="text-xs text-base-content/60">{game.publisher}</div>
                    </div>
                    <div className="text-right text-xs text-base-content/60">
                      <div>最終更新: {formatDateWithTime(game.updatedAt)}</div>
                      <div>総プレイ時間: {formatSmart(game.totalPlayTime)}</div>
                    </div>
                  </label>
                );
              })}
            </div>
          )}
        </div>
      </BaseModal>

      <CloudGameImportConflictModal
        isOpen={!!conflict}
        onClose={() => {
          setConflict(null);
          setImporting(false);
        }}
        cloudGame={conflict?.cloudGame ?? null}
        localMatches={conflict?.localMatches ?? []}
        onImportDuplicate={() => resolveConflict("duplicate")}
        onReplaceLocal={() => resolveConflict("replace")}
      />
    </>
  );
}
