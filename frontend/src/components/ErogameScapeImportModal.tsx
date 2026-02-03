/**
 * @fileoverview 批評空間からゲーム情報を取得して登録するモーダル
 */

import { useCallback, useEffect, useRef, useState } from "react";

import { BaseModal } from "./BaseModal";
import { GameFormFields } from "./GameFormFields";
import { useFileSelection } from "@renderer/hooks/useFileSelection";
import { useGameFormValidationZod } from "@renderer/hooks/useGameFormValidationZod";
import {
  handleApiError,
  handleUnexpectedError,
  showSuccessToast,
} from "@renderer/utils/errorHandler";
import type { GameImport, InputGameData } from "src/types/game";
import type { ApiResult } from "src/types/result";
import type { ErogameScapeSearchItem, ErogameScapeSearchResult } from "src/types/erogamescape";

type ErogameScapeImportModalProps = {
  isOpen: boolean;
  onClose: () => void;
  onClosed?: () => void;
  onSubmit: (gameData: InputGameData) => Promise<ApiResult>;
};

const initialValues: InputGameData = {
  title: "",
  publisher: "",
  saveFolderPath: "",
  exePath: "",
  imagePath: "",
  playStatus: "unplayed",
};

const erogameScapeIdRegex = /^\d+$/;

export default function ErogameScapeImportModal({
  isOpen,
  onClose,
  onClosed,
  onSubmit,
}: ErogameScapeImportModalProps): React.JSX.Element {
  const [gameData, setGameData] = useState<InputGameData>(initialValues);
  const [erogameId, setErogameId] = useState("");
  const [importedInfo, setImportedInfo] = useState<GameImport | null>(null);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [fetching, setFetching] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<ErogameScapeSearchItem[]>([]);
  const [searchError, setSearchError] = useState<string | null>(null);
  const [searching, setSearching] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [nextPageUrl, setNextPageUrl] = useState<string | null>(null);
  const [activeQuery, setActiveQuery] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const lastFetchedIdRef = useRef<string | null>(null);
  const searchRequestIdRef = useRef(0);
  const { isBrowsing, selectFile, selectFolder } = useFileSelection();
  const validation = useGameFormValidationZod(gameData);
  const prevIsOpenRef = useRef(isOpen);
  useEffect(() => {
    if (isOpen && !prevIsOpenRef.current) {
      validation.resetTouchedFields();
    }
    prevIsOpenRef.current = isOpen;
  }, [isOpen, validation]);

  useEffect(() => {
    if (!isOpen) {
      setGameData(initialValues);
      setErogameId("");
      setImportedInfo(null);
      setFetchError(null);
      setFetching(false);
      setSearchQuery("");
      setSearchResults([]);
      setSearchError(null);
      setSearching(false);
      setLoadingMore(false);
      setNextPageUrl(null);
      setActiveQuery("");
      setSubmitting(false);
      lastFetchedIdRef.current = null;
      searchRequestIdRef.current = 0;
    }
  }, [isOpen]);

  const applyImport = useCallback(
    (info: GameImport) => {
      setGameData((prev) => ({
        ...prev,
        title: info.title ?? "",
        publisher: info.brand ?? "",
        imagePath: info.imagePath ?? "",
      }));
    },
    [setGameData],
  );

  const fetchFromErogameScape = useCallback(
    async (id: string) => {
      if (!id.trim()) {
        setFetchError("批評空間IDを入力してください");
        return;
      }
      if (!erogameScapeIdRegex.test(id)) {
        setFetchError("批評空間IDは数字のみで入力してください");
        return;
      }
      setFetching(true);
      setFetchError(null);
      try {
        const result = await window.api.erogameScape.fetchById(id);
        if (!result.success || !result.data) {
          handleApiError(result, "批評空間からの取得に失敗しました");
          setFetchError(
            (result as { success: false; message: string }).message ||
              "批評空間からの取得に失敗しました",
          );
          return;
        }
        setImportedInfo(result.data);
        lastFetchedIdRef.current = id;
        applyImport(result.data);
        showSuccessToast("批評空間から情報を取得しました");
      } catch (error) {
        handleUnexpectedError(error, "批評空間情報の取得");
        setFetchError("批評空間からの取得に失敗しました");
      } finally {
        setFetching(false);
      }
    },
    [applyImport],
  );

  const searchErogameScape = useCallback(async (query: string) => {
    const requestId = searchRequestIdRef.current + 1;
    searchRequestIdRef.current = requestId;
    setSearching(true);
    setSearchError(null);
    try {
      const result = await window.api.erogameScape.searchByTitle(query);
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
      setActiveQuery(query);
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

  const handleSelectSearchItem = useCallback(
    async (item: ErogameScapeSearchItem) => {
      setErogameId(item.erogameScapeId);
      await fetchFromErogameScape(item.erogameScapeId);
    },
    [fetchFromErogameScape],
  );

  const browseImage = useCallback(async () => {
    await selectFile([{ name: "Image", extensions: ["png", "jpg", "jpeg", "gif"] }], (filePath) => {
      setGameData((prev) => ({ ...prev, imagePath: filePath }));
      validation.markFieldAsTouched("imagePath");
      validation.validateFileField("imagePath");
    });
  }, [selectFile, validation]);

  const browseExe = useCallback(async () => {
    await selectFile([{ name: "Executable", extensions: ["exe", "app"] }], (filePath) => {
      setGameData((prev) => ({ ...prev, exePath: filePath }));
      validation.markFieldAsTouched("exePath");
      validation.validateFileField("exePath");
    });
  }, [selectFile, validation]);

  const browseSaveFolder = useCallback(async () => {
    await selectFolder((folderPath) => {
      setGameData((prev) => ({ ...prev, saveFolderPath: folderPath }));
      validation.markFieldAsTouched("saveFolderPath");
      validation.validateFileField("saveFolderPath");
    });
  }, [selectFolder, validation]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>): void => {
    const { name, value } = e.target;
    setGameData((prev) => ({
      ...prev,
      [name]: value,
    }));
    validation.markFieldAsTouched(name as keyof InputGameData);
  };

  const handleSubmit = async (e: React.FormEvent): Promise<void> => {
    e.preventDefault();

    validation.markAllFieldsAsTouched();
    setSubmitting(true);
    try {
      const validationResult = await validation.validateAllFieldsWithFileCheck();
      if (!validationResult.isValid) {
        return;
      }

      const result = await onSubmit(gameData);
      if (result.success) {
        setGameData(initialValues);
        setErogameId("");
        setImportedInfo(null);
        lastFetchedIdRef.current = null;
        onClose();
      } else {
        handleApiError(result, "エラーが発生しました");
      }
    } catch (error) {
      handleUnexpectedError(error, "ゲーム情報の送信");
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancel = (): void => {
    setGameData(initialValues);
    setErogameId("");
    setImportedInfo(null);
    setFetchError(null);
    lastFetchedIdRef.current = null;
    validation.resetTouchedFields();
    onClose();
  };

  const footer = (
    <div className="flex justify-end space-x-2">
      <button
        type="button"
        className="btn"
        onClick={handleCancel}
        disabled={submitting || fetching}
      >
        キャンセル
      </button>
      <button
        type="submit"
        className="btn btn-primary"
        onClick={handleSubmit}
        disabled={submitting || fetching || !validation.canSubmit}
      >
        {`追加${submitting ? "中…" : ""}`}
      </button>
    </div>
  );

  return (
    <BaseModal
      id="erogamescape-import-modal"
      isOpen={isOpen}
      onClose={onClose}
      onClosed={onClosed}
      title="批評空間から登録"
      size="lg"
      footer={footer}
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="label" htmlFor="erogamescape-id">
            <span className="label-text">批評空間ID</span>
          </label>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <input
              type="text"
              id="erogamescape-id"
              value={erogameId}
              onChange={(e) => setErogameId(e.target.value)}
              placeholder="例: 13050"
              className={`input input-bordered w-full ${fetchError ? "input-error" : ""}`}
              disabled={fetching || submitting}
            />
            <button
              type="button"
              className="btn btn-outline"
              onClick={() => fetchFromErogameScape(erogameId.trim())}
              disabled={fetching || submitting}
            >
              {fetching ? "取得中..." : "取得"}
            </button>
          </div>
          {fetchError && (
            <div className="label">
              <span className="label-text-alt text-error">{fetchError}</span>
            </div>
          )}
          {!fetchError && importedInfo && (
            <div className="label">
              <span className="label-text-alt opacity-70">取得済み: {importedInfo.title}</span>
            </div>
          )}
        </div>

        <div className="rounded-lg border border-base-300 bg-base-100 p-4">
          <div className="text-sm font-semibold mb-3">タイトル検索</div>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="タイトル名で検索"
              className={`input input-bordered w-full ${searchError ? "input-error" : ""}`}
              disabled={searching || fetching || submitting}
            />
            <button
              type="button"
              className="btn btn-outline"
              onClick={() => searchErogameScape(searchQuery.trim())}
              disabled={searching || fetching || submitting}
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
          {searchResults.length > 0 && (
            <div
              className="mt-3 max-h-64 overflow-y-auto rounded border border-base-200"
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
                        onClick={() => handleSelectSearchItem(item)}
                        disabled={fetching || submitting}
                      >
                        選択
                      </button>
                    </div>
                  </li>
                ))}
              </ul>
              {loadingMore && (
                <div className="p-3 text-center text-sm opacity-70">読み込み中...</div>
              )}
              {!loadingMore && nextPageUrl && (
                <div className="p-2 text-center text-xs opacity-70">
                  下までスクロールで続きを取得
                </div>
              )}
            </div>
          )}
        </div>

        <GameFormFields
          gameData={gameData}
          onChange={handleChange}
          onBrowseImage={browseImage}
          onBrowseExe={browseExe}
          onBrowseSaveFolder={browseSaveFolder}
          disabled={submitting || isBrowsing || fetching}
          validation={validation}
        />
      </form>
    </BaseModal>
  );
}
