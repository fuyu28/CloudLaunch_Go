import { useCallback, useEffect, useMemo, useState } from "react";
import { FaPlus, FaRoute, FaTrash } from "react-icons/fa";

import { handleApiError, showSuccessToast } from "@renderer/utils/errorHandler";
import { logger } from "@renderer/utils/logger";

import type { PlayRouteType } from "src/types/game";

type PlayRouteCardProps = {
  gameId: string;
};

export default function PlayRouteCard({ gameId }: PlayRouteCardProps): React.JSX.Element {
  const [routes, setRoutes] = useState<PlayRouteType[]>([]);
  const [routeName, setRouteName] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const fetchRoutes = useCallback(async () => {
    setIsLoading(true);
    try {
      const result = await window.api.playRoute.listByGame(gameId);
      if (result.success && result.data) {
        setRoutes(result.data);
      } else {
        setRoutes([]);
      }
    } catch (error) {
      logger.error("プレイルート取得に失敗", {
        component: "PlayRouteCard",
        function: "fetchRoutes",
        data: error,
      });
      setRoutes([]);
    } finally {
      setIsLoading(false);
    }
  }, [gameId]);

  useEffect(() => {
    void fetchRoutes();
  }, [fetchRoutes]);

  const nextSortOrder = useMemo(() => {
    if (routes.length === 0) {
      return 0;
    }
    return Math.max(...routes.map((route) => route.sortOrder)) + 1;
  }, [routes]);

  const handleCreate = useCallback(async () => {
    const trimmed = routeName.trim();
    if (!trimmed) {
      return;
    }

    setIsSubmitting(true);
    try {
      const result = await window.api.playRoute.create({
        gameId,
        name: trimmed,
        sortOrder: nextSortOrder,
      });
      if (!result.success) {
        handleApiError(result, "プレイルートの作成に失敗しました");
        return;
      }
      setRouteName("");
      showSuccessToast("プレイルートを追加しました");
      await fetchRoutes();
    } catch (error) {
      logger.error("プレイルート作成に失敗", {
        component: "PlayRouteCard",
        function: "handleCreate",
        data: error,
      });
    } finally {
      setIsSubmitting(false);
    }
  }, [fetchRoutes, gameId, nextSortOrder, routeName]);

  const handleDelete = useCallback(
    async (routeId: string) => {
      try {
        const result = await window.api.playRoute.delete(routeId);
        if (!result.success) {
          handleApiError(result, "プレイルートの削除に失敗しました");
          return;
        }
        showSuccessToast("プレイルートを削除しました");
        await fetchRoutes();
      } catch (error) {
        logger.error("プレイルート削除に失敗", {
          component: "PlayRouteCard",
          function: "handleDelete",
          data: error,
        });
      }
    },
    [fetchRoutes],
  );

  return (
    <div className="card bg-base-100 shadow-xl h-full">
      <div className="card-body">
        <div className="flex items-center justify-between mb-4">
          <h2 className="card-title text-lg">
            <FaRoute className="text-primary" />
            プレイルート
          </h2>
          <div className="badge badge-outline">{routes.length}件</div>
        </div>

        <div className="flex gap-2 mb-4">
          <input
            type="text"
            className="input input-bordered flex-1"
            placeholder="ルート名を入力"
            value={routeName}
            onChange={(event) => setRouteName(event.target.value)}
            disabled={isSubmitting}
          />
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => void handleCreate()}
            disabled={isSubmitting || routeName.trim() === ""}
          >
            <FaPlus />
            追加
          </button>
        </div>

        {isLoading ? (
          <div className="flex justify-center py-8">
            <div className="loading loading-spinner loading-md"></div>
          </div>
        ) : routes.length > 0 ? (
          <div className="space-y-2">
            {routes.map((route) => (
              <div
                key={route.id}
                className="flex items-center justify-between rounded-lg border border-base-300 bg-base-200 px-3 py-2"
              >
                <div className="min-w-0">
                  <div className="font-medium truncate">{route.name}</div>
                  <div className="text-xs text-base-content/70">順序: {route.sortOrder}</div>
                </div>
                <button
                  type="button"
                  className="btn btn-ghost btn-sm text-error"
                  onClick={() => void handleDelete(route.id)}
                >
                  <FaTrash />
                </button>
              </div>
            ))}
          </div>
        ) : (
          <div className="text-sm text-base-content/70 py-6 text-center">
            プレイルートはまだありません
          </div>
        )}
      </div>
    </div>
  );
}
