/**
 * @fileoverview 批評空間タイトル検索モーダル
 */

import { useCallback, useEffect, useRef, useState } from "react";

import { BaseModal } from "./BaseModal";
import type { ErogameScapeSearchItem, ErogameScapeSearchResult } from "src/types/erogamescape";

type ErogameScapeSearchModalProps = {
  isOpen: boolean;
  onClose: () => void;
  onSelect: (item: ErogameScapeSearchItem) => void;
};

export default function ErogameScapeSearchModal({
  isOpen,
  onClose,
  onSelect,
}: ErogameScapeSearchModalProps): React.JSX.Element {
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<ErogameScapeSearchItem[]>([]);
  const [searchError, setSearchError] = useState<string | null>(null);
  const [searching, setSearching] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [nextPageUrl, setNextPageUrl] = useState<string | null>(null);
  const [activeQuery, setActiveQuery] = useState("");
  const searchRequestIdRef = useRef(0);

  useEffect(() => {
    if (!isOpen) {
      setSearchQuery("");
      setSearchResults([]);
      setSearchError(null);
      setSearching(false);
      setLoadingMore(false);
      setNextPageUrl(null);
      setActiveQuery("");
      searchRequestIdRef.current = 0;
    }
  }, [isOpen]);

  const searchErogameScape = useCallback(async (query: string) => {
    const trimmed = query.trim();
    if (!trimmed) {
      setSearchError("検索ワードを入力してください");
      return;
    }
    const requestId = searchRequestIdRef.current + 1;
    searchRequestIdRef.current = requestId;
    setSearching(true);
    setSearchError(null);
    try {
      const result = await window.api.erogameScape.searchByTitle(trimmed);
      if (requestId !== searchRequestIdRef.current) {
        return;
      }
      if (!result.success || !result.data) {
        setSearchError(
          (result as { success: false; message: string }).message || "批評空間の検索に失敗しました",
        );
        setSearchResults([]);
        setNextPageUrl(null);
        return;
      }
      const data = result.data as ErogameScapeSearchResult;
      setSearchResults(data.items ?? []);
      setNextPageUrl(data.nextPageUrl ?? null);
      setActiveQuery(trimmed);
    } catch {
      setSearchError("批評空間の検索に失敗しました");
      setSearchResults([]);
      setNextPageUrl(null);
    } finally {
      setSearching(false);
    }
  }, []);

  const loadMoreResults = useCallback(async () => {
    if (!nextPageUrl || loadingMore) {
      return;
    }
    setLoadingMore(true);
    setSearchError(null);
    try {
      const result = await window.api.erogameScape.searchByTitle(activeQuery, nextPageUrl);
      if (!result.success || !result.data) {
        setSearchError(
          (result as { success: false; message: string }).message || "批評空間の検索に失敗しました",
        );
        return;
      }
      const data = result.data as ErogameScapeSearchResult;
      setSearchResults((prev) => [...prev, ...(data.items ?? [])]);
      setNextPageUrl(data.nextPageUrl ?? null);
    } catch {
      setSearchError("批評空間の検索に失敗しました");
    } finally {
      setLoadingMore(false);
    }
  }, [activeQuery, loadingMore, nextPageUrl]);

  const handleResultsScroll = useCallback(
    (event: React.UIEvent<HTMLDivElement>) => {
      const target = event.currentTarget;
      if (!nextPageUrl || loadingMore) {
        return;
      }
      const remaining = target.scrollHeight - target.scrollTop - target.clientHeight;
      if (remaining < 64) {
        void loadMoreResults();
      }
    },
    [loadMoreResults, loadingMore, nextPageUrl],
  );

  const footer = (
    <div className="flex justify-end">
      <button type="button" className="btn" onClick={onClose}>
        閉じる
      </button>
    </div>
  );

  return (
    <BaseModal
      id="erogamescape-search-modal"
      isOpen={isOpen}
      onClose={onClose}
      title="批評空間のタイトル検索"
      size="lg"
      footer={footer}
    >
      <div className="space-y-4">
        <div>
          <label className="label" htmlFor="erogamescape-search">
            <span className="label-text">タイトル</span>
          </label>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <input
              type="text"
              id="erogamescape-search"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="タイトル名で検索"
              className={`input input-bordered w-full ${searchError ? "input-error" : ""}`}
              disabled={searching || loadingMore}
            />
            <button
              type="button"
              className="btn btn-outline"
              onClick={() => searchErogameScape(searchQuery)}
              disabled={searching || loadingMore}
            >
              {searching ? "検索中..." : "検索"}
            </button>
          </div>
          {searchError && (
            <div className="label">
              <span className="label-text-alt text-error">{searchError}</span>
            </div>
          )}
          {!searchError &&
            searchResults.length === 0 &&
            searchQuery.trim() !== "" &&
            !searching && (
              <div className="label">
                <span className="label-text-alt opacity-70">検索結果がありません</span>
              </div>
            )}
        </div>

        {searchResults.length > 0 && (
          <div
            className="max-h-72 overflow-y-auto rounded border border-base-200"
            onScroll={handleResultsScroll}
          >
            <ul className="divide-y divide-base-200">
              {searchResults.map((item) => (
                <li key={`${item.erogameScapeId}-${item.gameUrl}`} className="p-3">
                  <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                    <div>
                      <div className="font-semibold">{item.title}</div>
                      <div className="text-xs opacity-70">
                        ID: {item.erogameScapeId}
                        {item.brand ? ` / ${item.brand}` : ""}
                      </div>
                    </div>
                    <button
                      type="button"
                      className="btn btn-sm btn-outline"
                      onClick={() => onSelect(item)}
                    >
                      選択
                    </button>
                  </div>
                </li>
              ))}
            </ul>
            {loadingMore && <div className="p-3 text-center text-sm opacity-70">読み込み中...</div>}
            {!loadingMore && nextPageUrl && (
              <div className="p-2 text-center text-xs opacity-70">下までスクロールで続きを取得</div>
            )}
          </div>
        )}
      </div>
    </BaseModal>
  );
}
