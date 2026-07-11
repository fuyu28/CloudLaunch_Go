/**
 * @fileoverview クラウドから既存ゲームを追加するモーダル
 *
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "react-hot-toast";

import { useCloudSync } from "@renderer/hooks/useCloudSync";
import { useOfflineMode } from "@renderer/hooks/useOfflineMode";
import { useTimeFormat } from "@renderer/hooks/useTimeFormat";
import { logger } from "@renderer/utils/logger";

import { BaseModal } from "../common/BaseModal";
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
  const { pull } = useCloudSync(isOfflineMode);
  const { formatDateWithTime, formatSmart } = useTimeFormat();
  const [cloudGames, setCloudGames] = useState<CloudGameMetadata[]>([]);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [importing, setImporting] = useState(false);
  const [conflict, setConflict] = useState<ConflictState | null>(null);
  const [localCache, setLocalCache] = useState<GameType[]>(localGames);
  // バッチ取り込み中に1件でも取り込めたか（バッチ終了時に1回だけ親へ通知するため）
  const importedAnyRef = useRef(false);

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
        toast.error(
          (!result.success ? result.message : undefined) ?? "クラウドデータの取得に失敗しました",
        );
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

    // 取り込み処理中は再フェッチしない。
    // 取り込みごとに親の localGames が更新されて useEffect が再実行されると、
    // モーダルの一覧が1件ごとに再読み込みされてしまうため。
    if (importing) {
      return;
    }

    setLocalCache(localGames);
    fetchLocalGames();
    fetchCloudGames();
  }, [fetchCloudGames, fetchLocalGames, isOpen, localGames, importing]);

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
      const op = await pull(game.id);
      if (!op.ok) {
        toast.error(op.message ?? "クラウドゲームの追加に失敗しました");
        return false;
      }
      if (op.ok && op.applied === false) {
        // 同期対象外のローカルファイルがある場合は破壊を避けて中断（詳細画面で確認）
        toast.error(
          "同期対象外のローカルファイルがあるため、ゲーム詳細の「同期」から確認してください。",
        );
        return false;
      }

      toast.success(`「${game.title}」を追加しました`);
      setCloudGames((prev) => prev.filter((item) => item.id !== game.id));
      setSelectedIds((prev) => {
        const next = new Set(prev);
        next.delete(game.id);
        return next;
      });

      // 1件ごとに親へ通知すると一覧再取得が連打されるのでバッチ終了時に1回。
      importedAnyRef.current = true;
      return true;
    },
    [pull],
  );

  // 取り込み中フラグを下ろし、成功が1件でもあれば親へ1回だけ通知する。
  const finishBatch = useCallback(async (): Promise<void> => {
    setImporting(false);
    if (importedAnyRef.current) {
      importedAnyRef.current = false;
      if (onImported) {
        await onImported();
      }
    }
  }, [onImported]);

  const processQueue = useCallback(
    async (queue: CloudGameMetadata[], excludeLocalIds?: Set<string>): Promise<void> => {
      if (queue.length === 0) {
        await finishBatch();
        return;
      }

      const [current, ...rest] = queue;
      // replace 直後は localCache の setState が反映される前に processQueue を呼び出すため、
      // localTitleMap の再計算は次のレンダリングまで走らない。呼び出し元が「今削除したローカルID」を
      // excludeLocalIds として渡すことで、古い closure でも削除済みゲームを競合判定から除外できる。
      const rawConflicts = localTitleMap.get(normalizeTitle(current.title)) ?? [];
      const conflicts = excludeLocalIds
        ? rawConflicts.filter((g) => !excludeLocalIds.has(g.id))
        : rawConflicts;
      if (conflicts.length > 0) {
        setConflict({ cloudGame: current, localMatches: conflicts, remainingQueue: rest });
        return;
      }

      const imported = await importGame(current);
      if (!imported) {
        await finishBatch();
        return;
      }

      await processQueue(rest, excludeLocalIds);
    },
    [finishBatch, importGame, localTitleMap],
  );

  const handleStartImport = async (): Promise<void> => {
    if (selectedGames.length === 0) {
      toast.error("追加するゲームを選択してください");
      return;
    }

    importedAnyRef.current = false;
    setImporting(true);
    await processQueue(selectedGames);
  };

  const resolveConflict = async (mode: "duplicate" | "replace"): Promise<void> => {
    if (!conflict) return;

    setConflict(null);
    if (mode === "replace") {
      const deletedIds: string[] = [];
      for (const localGame of conflict.localMatches) {
        const result = await window.api.database.deleteGame(localGame.id);
        if (!result.success) {
          toast.error(result.message ?? "ローカルゲームの削除に失敗しました");
          // 部分削除後に localCache を直さないと、続く同名衝突判定がズレる。
          if (deletedIds.length > 0) {
            setLocalCache((prev) => prev.filter((g) => !deletedIds.includes(g.id)));
          }
          await finishBatch();
          return;
        }
        // 削除が発生した時点で一覧更新が必要。バッチ終了時にまとめて通知する。
        importedAnyRef.current = true;
        deletedIds.push(localGame.id);
      }
      // importing 中は fetchLocalGames が走らないため、削除済みIDを localCache から除去し
      // 後続キューの localTitleMap 判定に「削除済み」を反映する。
      if (deletedIds.length > 0) {
        setLocalCache((prev) => prev.filter((g) => !deletedIds.includes(g.id)));
      }
    }

    const imported = await importGame(conflict.cloudGame);
    if (!imported) {
      await finishBatch();
      return;
    }

    // replace 時に削除したローカルIDを後続キューへ引き継ぐ。localCache の再計算前に
    // processQueue が走る場合でも「削除済み」を競合判定から除外できる。
    const excludeLocalIds =
      mode === "replace" ? new Set(conflict.localMatches.map((g) => g.id)) : undefined;
    await processQueue(conflict.remainingQueue, excludeLocalIds);
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
          void finishBatch();
        }}
        cloudGame={conflict?.cloudGame ?? null}
        localMatches={conflict?.localMatches ?? []}
        onImportDuplicate={() => resolveConflict("duplicate")}
        onReplaceLocal={() => resolveConflict("replace")}
      />
    </>
  );
}
