/**
 * @fileoverview 一般設定コンポーネント
 *
 * アプリケーションの一般的な設定項目を管理するコンポーネントです。
 *
 * 主な機能：
 * - DaisyUIテーマの選択・変更
 * - デフォルトソート順の設定
 * - デフォルトフィルター状態の設定
 * - オフラインモードの設定
 * - 自動ゲーム検出の設定
 * - 設定の永続化
 * - リアルタイムでの変更反映
 *
 * 使用技術：
 * - Jotai atoms（状態管理）
 * - DaisyUI コンポーネント
 */

import { useAtom } from "jotai";
import { useEffect, useState } from "react";
import toast from "react-hot-toast";

import { logger } from "@renderer/utils/logger";

import { DAISYUI_THEMES } from "@renderer/constants/themes";
import {
  themeAtom,
  changeThemeAtom,
  isChangingThemeAtom,
  defaultSortOptionAtom,
  defaultFilterStateAtom,
  offlineModeAtom,
  autoTrackingAtom,
  transferConcurrencyAtom,
  transferRetryCountAtom,
  screenshotSyncEnabledAtom,
  screenshotUploadJpegAtom,
  screenshotJpegQualityAtom,
  screenshotClientOnlyAtom,
  screenshotLocalJpegAtom,
  sortOptionLabels,
  filterStateLabels,
} from "../state/settings";
import type { SortOption, FilterOption } from "src/types/menu";

/**
 * 一般設定コンポーネント
 *
 * テーマ選択、デフォルトソート順、デフォルトフィルター状態、
 * オフラインモード、自動ゲーム検出など、アプリケーションの一般的な設定を提供します。
 *
 * @returns 一般設定コンポーネント要素
 */
export default function GeneralSettings(): React.JSX.Element {
  const [currentTheme] = useAtom(themeAtom);
  const [isChangingTheme] = useAtom(isChangingThemeAtom);
  const [, changeTheme] = useAtom(changeThemeAtom);
  const [defaultSortOption, setDefaultSortOption] = useAtom(defaultSortOptionAtom);
  const [defaultFilterState, setDefaultFilterState] = useAtom(defaultFilterStateAtom);
  const [offlineMode, setOfflineMode] = useAtom(offlineModeAtom);
  const [autoTracking, setAutoTracking] = useAtom(autoTrackingAtom);
  const [transferConcurrency, setTransferConcurrency] = useAtom(transferConcurrencyAtom);
  const [transferRetryCount, setTransferRetryCount] = useAtom(transferRetryCountAtom);
  const [screenshotSyncEnabled, setScreenshotSyncEnabled] = useAtom(screenshotSyncEnabledAtom);
  const [screenshotUploadJpeg, setScreenshotUploadJpeg] = useAtom(screenshotUploadJpegAtom);
  const [screenshotJpegQuality, setScreenshotJpegQuality] = useAtom(screenshotJpegQualityAtom);
  const [screenshotClientOnly, setScreenshotClientOnly] = useAtom(screenshotClientOnlyAtom);
  const [screenshotLocalJpeg, setScreenshotLocalJpeg] = useAtom(screenshotLocalJpegAtom);
  const [isSyncingAll, setIsSyncingAll] = useState(false);

  // ソート変更ハンドラー
  const handleSortChange = (newSortOption: SortOption): void => {
    setDefaultSortOption(newSortOption);
    toast.success(`デフォルトソート順を「${sortOptionLabels[newSortOption]}」に変更しました`);
  };

  // フィルター変更ハンドラー
  const handleFilterChange = (newFilterState: FilterOption): void => {
    setDefaultFilterState(newFilterState);
    toast.success(`デフォルトフィルターを「${filterStateLabels[newFilterState]}」に変更しました`);
  };

  // オフラインモード変更ハンドラー
  const handleOfflineModeChange = async (enabled: boolean): Promise<void> => {
    setOfflineMode(enabled);
    try {
      const result = await window.api.settings.updateOfflineMode(enabled);
      if (!result.success) {
        toast.error("オフラインモードの更新に失敗しました");
        return;
      }
      if (enabled) {
        toast.success("オフラインモードを有効にしました");
      } else {
        toast.success("オフラインモードを無効にしました");
      }
    } catch (error) {
      logger.error("オフラインモード更新エラー:", {
        component: "GeneralSettings",
        function: "handleOfflineModeChange",
        data: error,
      });
      toast.error("オフラインモードの更新に失敗しました");
    }
  };

  // 自動ゲーム検出変更ハンドラー
  const handleAutoTrackingChange = async (enabled: boolean): Promise<void> => {
    setAutoTracking(enabled);

    // メインプロセスに設定変更を通知
    try {
      const result = await window.api.settings.updateAutoTracking(enabled);
      if (result.success) {
        if (enabled) {
          toast.success("自動ゲーム検出を有効にしました");
        } else {
          toast.success("自動ゲーム検出を無効にしました");
        }
      } else {
        toast.error("設定の更新に失敗しました");
      }
    } catch (error) {
      logger.error("自動ゲーム検出設定の更新エラー:", {
        component: "GeneralSettings",
        function: "unknown",
        data: error,
      });
      toast.error("設定の更新に失敗しました");
    }
  };

  const applyTransferConcurrency = async (value: number, showToast: boolean): Promise<void> => {
    try {
      const result = await window.api.settings.updateUploadConcurrency(value);
      if (!result.success) {
        if (showToast) {
          toast.error("同時転送数の更新に失敗しました");
        }
      } else if (showToast) {
        toast.success(`同時転送数を ${value} に設定しました`);
      }
    } catch (error) {
      logger.error("同時転送数設定の更新エラー:", {
        component: "GeneralSettings",
        function: "unknown",
        data: error,
      });
      if (showToast) {
        toast.error("同時転送数の更新に失敗しました");
      }
    }
  };

  const handleTransferConcurrencyChange = async (value: number): Promise<void> => {
    const nextValue = Math.min(32, Math.max(1, value));
    setTransferConcurrency(nextValue);
    await applyTransferConcurrency(nextValue, true);
  };

  const applyTransferRetryCount = async (value: number, showToast: boolean): Promise<void> => {
    try {
      const result = await window.api.settings.updateTransferRetryCount(value);
      if (!result.success) {
        if (showToast) {
          toast.error("リトライ回数の更新に失敗しました");
        }
      } else if (showToast) {
        toast.success(`リトライ回数を ${value} に設定しました`);
      }
    } catch (error) {
      logger.error("リトライ回数設定の更新エラー:", {
        component: "GeneralSettings",
        function: "unknown",
        data: error,
      });
      if (showToast) {
        toast.error("リトライ回数の更新に失敗しました");
      }
    }
  };

  const handleTransferRetryCountChange = async (value: number): Promise<void> => {
    const nextValue = Math.min(10, Math.max(0, value));
    setTransferRetryCount(nextValue);
    await applyTransferRetryCount(nextValue, true);
  };

  const handleScreenshotSyncEnabledChange = async (enabled: boolean): Promise<void> => {
    setScreenshotSyncEnabled(enabled);
    const result = await window.api.settings.updateScreenshotSyncEnabled(enabled);
    if (!result.success) {
      toast.error("スクリーンショット同期の更新に失敗しました");
      return;
    }
    toast.success(`スクリーンショット同期を${enabled ? "有効" : "無効"}にしました`);
  };

  const handleScreenshotUploadJpegChange = async (enabled: boolean): Promise<void> => {
    setScreenshotUploadJpeg(enabled);
    const result = await window.api.settings.updateScreenshotUploadJpeg(enabled);
    if (!result.success) {
      toast.error("スクリーンショット形式の更新に失敗しました");
      return;
    }
    toast.success(`スクリーンショットを${enabled ? "JPEG" : "PNG"}でアップロードします`);
  };

  const handleScreenshotJpegQualityChange = async (value: number): Promise<void> => {
    const nextValue = Math.min(100, Math.max(1, value));
    setScreenshotJpegQuality(nextValue);
    const result = await window.api.settings.updateScreenshotJpegQuality(nextValue);
    if (!result.success) {
      toast.error("スクリーンショット品質の更新に失敗しました");
      return;
    }
  };

  const handleScreenshotClientOnlyChange = async (enabled: boolean): Promise<void> => {
    setScreenshotClientOnly(enabled);
    const result = await window.api.settings.updateScreenshotClientOnly(enabled);
    if (!result.success) {
      toast.error("スクリーンショット設定の更新に失敗しました");
      return;
    }
    toast.success(enabled ? "タイトルバーを除外して撮影します" : "タイトルバーを含めて撮影します");
  };

  const handleScreenshotLocalJpegChange = async (enabled: boolean): Promise<void> => {
    setScreenshotLocalJpeg(enabled);
    const result = await window.api.settings.updateScreenshotLocalJpeg(enabled);
    if (!result.success) {
      toast.error("スクリーンショット設定の更新に失敗しました");
      return;
    }
    toast.success(enabled ? "ローカル保存をJPEGにします" : "ローカル保存をPNGにします");
  };

  // ログフォルダを開くハンドラー
  const handleOpenLogsDirectory = async (): Promise<void> => {
    try {
      const result = await window.api.file.openLogsDirectory();
      if (result.success) {
        toast.success("ログフォルダを開きました");
      } else {
        toast.error(result.message || "ログフォルダを開くことができませんでした");
      }
    } catch (error) {
      logger.error("ログフォルダを開くエラー:", {
        component: "GeneralSettings",
        function: "handleOpenLogsDirectory",
        data: error,
      });
      toast.error("ログフォルダを開くことができませんでした");
    }
  };

  const handleSyncAllGames = async (): Promise<void> => {
    if (offlineMode) {
      toast.error("オフラインモードでは同期できません");
      return;
    }
    setIsSyncingAll(true);
    try {
      const result = await window.api.cloudSync.syncAllGames();
      if (!result.success || !result.data) {
        toast.error(result.message || "クラウド同期に失敗しました");
        return;
      }
      const summary = result.data;
      toast.success(
        `同期完了: アップロード${summary.uploadedGames}件 / ダウンロード${summary.downloadedGames}件`,
      );
    } catch (error) {
      logger.error("全ゲーム同期エラー:", {
        component: "GeneralSettings",
        function: "handleSyncAllGames",
        data: error,
      });
      toast.error("クラウド同期に失敗しました");
    } finally {
      setIsSyncingAll(false);
    }
  };

  useEffect(() => {
    void applyTransferConcurrency(transferConcurrency, false);
  }, []);

  useEffect(() => {
    void applyTransferRetryCount(transferRetryCount, false);
  }, []);

  useEffect(() => {
    void window.api.settings.updateOfflineMode(offlineMode);
  }, []);

  useEffect(() => {
    void window.api.settings.updateScreenshotSyncEnabled(screenshotSyncEnabled);
  }, []);

  useEffect(() => {
    void window.api.settings.updateScreenshotUploadJpeg(screenshotUploadJpeg);
  }, []);

  useEffect(() => {
    void window.api.settings.updateScreenshotJpegQuality(screenshotJpegQuality);
  }, []);

  useEffect(() => {
    void window.api.settings.updateScreenshotClientOnly(screenshotClientOnly);
  }, []);

  useEffect(() => {
    void window.api.settings.updateScreenshotLocalJpeg(screenshotLocalJpeg);
  }, []);

  return (
    <div className="w-full">
      <h2 className="text-xl font-semibold mb-6">一般設定</h2>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* 外観設定グループ */}
        <div className="space-y-6">
          <div className="border-l-4 border-primary pl-4">
            <h3 className="text-lg font-semibold text-primary mb-1">外観設定</h3>
            <p className="text-sm text-base-content/60">アプリケーションの見た目を設定</p>
          </div>

          {/* テーマ選択 */}
          <div className="bg-base-200 p-4 rounded-lg">
            <div className="mb-3">
              <h4 className="font-medium">テーマ</h4>
              <p className="text-sm text-base-content/70">外観テーマを選択</p>
            </div>
            <div className="form-control">
              <label className="label pb-1">
                <span className="label-text text-sm">現在: {currentTheme}</span>
              </label>
              <div className="flex items-center gap-2">
                <select
                  className="select select-bordered select-sm"
                  value={currentTheme}
                  onChange={(e) => changeTheme(e.target.value as typeof currentTheme)}
                  disabled={isChangingTheme}
                >
                  {DAISYUI_THEMES.map((theme) => (
                    <option key={theme} value={theme}>
                      {theme}
                    </option>
                  ))}
                </select>
                {isChangingTheme && <span className="loading loading-spinner loading-sm"></span>}
              </div>
            </div>
          </div>
        </div>

        {/* スクリーンショット設定グループ */}
        <div className="space-y-6">
          <div className="border-l-4 border-primary pl-4">
            <h3 className="text-lg font-semibold text-primary mb-1">スクリーンショット</h3>
            <p className="text-sm text-base-content/60">撮影データの同期と形式</p>
          </div>

          <div className="bg-base-200 p-4 rounded-lg space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h4 className="font-medium">クラウド同期</h4>
                <p className="text-sm text-base-content/70">
                  スクリーンショットをクラウドにアップロードします
                </p>
              </div>
              <input
                type="checkbox"
                className="toggle toggle-primary"
                checked={screenshotSyncEnabled}
                onChange={(event) => void handleScreenshotSyncEnabledChange(event.target.checked)}
              />
            </div>

            <div className="flex items-center justify-between">
              <div>
                <h4 className="font-medium">タイトルバーを除外</h4>
                <p className="text-sm text-base-content/70">
                  オンでクライアント領域のみを撮影します
                </p>
              </div>
              <input
                type="checkbox"
                className="toggle toggle-primary"
                checked={screenshotClientOnly}
                onChange={(event) => void handleScreenshotClientOnlyChange(event.target.checked)}
              />
            </div>

            <div className="flex items-center justify-between">
              <div>
                <h4 className="font-medium">ローカル保存をJPEGにする</h4>
                <p className="text-sm text-base-content/70">オンでPNGより容量を抑えて保存します</p>
              </div>
              <input
                type="checkbox"
                className="toggle toggle-primary"
                checked={screenshotLocalJpeg}
                onChange={(event) => void handleScreenshotLocalJpegChange(event.target.checked)}
              />
            </div>

            <div className="flex items-center justify-between">
              <div>
                <h4 className="font-medium">JPEGでアップロード</h4>
                <p className="text-sm text-base-content/70">PNGより容量を抑えられます</p>
              </div>
              <input
                type="checkbox"
                className="toggle toggle-primary"
                checked={screenshotUploadJpeg}
                onChange={(event) => void handleScreenshotUploadJpegChange(event.target.checked)}
                disabled={!screenshotSyncEnabled}
              />
            </div>

            <div>
              <div className="flex items-center justify-between mb-2">
                <div>
                  <h4 className="font-medium">JPEG品質</h4>
                  <p className="text-sm text-base-content/70">
                    数値が高いほど画質は向上します（1-100）
                  </p>
                </div>
                <span className="text-sm font-mono">{screenshotJpegQuality}</span>
              </div>
              <input
                type="range"
                min={1}
                max={100}
                value={screenshotJpegQuality}
                onChange={(event) =>
                  void handleScreenshotJpegQualityChange(Number(event.target.value))
                }
                className="range range-primary"
                disabled={!screenshotSyncEnabled || !screenshotUploadJpeg}
              />
            </div>
          </div>
        </div>

        {/* 動作設定グループ */}
        <div className="space-y-6">
          <div className="border-l-4 border-secondary pl-4">
            <h3 className="text-lg font-semibold text-secondary mb-1">動作設定</h3>
            <p className="text-sm text-base-content/60">アプリケーションの動作を設定</p>
          </div>

          {/* オフラインモード & 自動計測 */}
          <div className="bg-base-200 p-4 rounded-lg space-y-4">
            <div>
              <h4 className="font-medium mb-3">機能設定</h4>

              {/* オフラインモード */}
              <div className="form-control mb-4">
                <label className="label cursor-pointer justify-start p-0">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary mr-3"
                    checked={offlineMode}
                    onChange={(e) => handleOfflineModeChange(e.target.checked)}
                  />
                  <div>
                    <span className="label-text font-medium">オフラインモード</span>
                    <p className="text-xs text-base-content/50 mt-1">
                      {offlineMode ? "クラウド機能が無効" : "すべての機能が利用可能"}
                    </p>
                  </div>
                </label>
              </div>

              {/* 自動ゲーム検出 */}
              <div className="form-control">
                <label className="label cursor-pointer justify-start p-0">
                  <input
                    type="checkbox"
                    className="toggle toggle-primary mr-3"
                    checked={autoTracking}
                    onChange={(e) => handleAutoTrackingChange(e.target.checked)}
                  />
                  <div>
                    <span className="label-text font-medium">自動ゲーム検出</span>
                    <p className="text-xs text-base-content/50 mt-1">
                      {autoTracking
                        ? "実行中ゲームを自動検出して監視開始"
                        : "手動でのゲーム登録のみ"}
                    </p>
                  </div>
                </label>
              </div>

              {/* リトライ回数 */}
              <div className="form-control mt-4">
                <label className="label p-0 mb-2">
                  <span className="label-text font-medium">リトライ回数</span>
                </label>
                <div className="flex items-center gap-3">
                  <input
                    type="number"
                    min={0}
                    max={10}
                    step={1}
                    className="input input-bordered input-sm w-24"
                    value={transferRetryCount}
                    onChange={(e) => setTransferRetryCount(Number(e.target.value))}
                    onBlur={(e) => handleTransferRetryCountChange(Number(e.target.value))}
                  />
                  <span className="text-xs text-base-content/50">0〜10</span>
                </div>
                <p className="text-xs text-base-content/50 mt-2">
                  アップロード/ダウンロード共通のリトライ回数です
                </p>
              </div>

              {/* 同時アップロード数 */}
              <div className="form-control mt-4">
                <label className="label p-0 mb-2">
                  <span className="label-text font-medium">同時転送数</span>
                </label>
                <div className="flex items-center gap-3">
                  <input
                    type="number"
                    min={1}
                    max={32}
                    step={1}
                    className="input input-bordered input-sm w-24"
                    value={transferConcurrency}
                    onChange={(e) => setTransferConcurrency(Number(e.target.value))}
                    onBlur={(e) => handleTransferConcurrencyChange(Number(e.target.value))}
                  />
                  <span className="text-xs text-base-content/50">1〜32</span>
                </div>
                <p className="text-xs text-base-content/50 mt-2">
                  アップロード/ダウンロード共通の同時転送数です
                </p>
              </div>
            </div>
          </div>

          {/* ログ・デバッグ */}
          <div className="bg-base-200 p-4 rounded-lg">
            <div className="mb-3">
              <h4 className="font-medium">ログ・デバッグ</h4>
              <p className="text-sm text-base-content/70">トラブルシューティング用</p>
            </div>
            <div className="form-control">
              <button className="btn btn-outline btn-sm w-fit" onClick={handleOpenLogsDirectory}>
                <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2H5a2 2 0 00-2-2z"
                  />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 1v6" />
                </svg>
                ログフォルダを開く
              </button>
              <p className="text-xs text-base-content/50 mt-2">
                アプリケーションのログファイルが保存されているフォルダを開きます
              </p>
            </div>
          </div>
        </div>

        {/* クラウド同期 */}
        <div className="bg-base-200 p-4 rounded-lg lg:col-span-2">
          <div className="mb-3">
            <h4 className="font-medium">クラウド同期</h4>
            <p className="text-sm text-base-content/70">ゲーム情報とセッションを同期します</p>
          </div>
          <div className="form-control">
            <button
              className="btn btn-outline btn-sm w-fit"
              onClick={handleSyncAllGames}
              disabled={isSyncingAll || offlineMode}
            >
              {isSyncingAll ? "同期中..." : "全ゲームを同期"}
            </button>
            <p className="text-xs text-base-content/50 mt-2">
              変更があったゲームのみクラウドと同期します
            </p>
          </div>
        </div>

        {/* デフォルト設定グループ */}
        <div className="lg:col-span-2 space-y-6">
          <div className="border-l-4 border-accent pl-4">
            <h3 className="text-lg font-semibold text-accent mb-1">デフォルト設定</h3>
            <p className="text-sm text-base-content/60">ホーム画面の初期表示設定</p>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            {/* デフォルトソート */}
            <div className="bg-base-200 p-4 rounded-lg">
              <div className="mb-3">
                <h4 className="font-medium">ソート順</h4>
                <p className="text-sm text-base-content/70">初期表示時のソート方法</p>
              </div>
              <div className="form-control">
                <div className="mb-2">
                  <p className="text-xs text-base-content/60 mt-1">
                    {`現在: ${sortOptionLabels[defaultSortOption]}`}
                  </p>
                </div>
                <select
                  className="select select-bordered select-sm"
                  value={defaultSortOption}
                  onChange={(e) => handleSortChange(e.target.value as SortOption)}
                >
                  {Object.entries(sortOptionLabels).map(([value, label]) => (
                    <option key={value} value={value}>
                      {label}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            {/* デフォルトフィルター */}
            <div className="bg-base-200 p-4 rounded-lg">
              <div className="mb-3">
                <h4 className="font-medium">フィルター</h4>
                <p className="text-sm text-base-content/70">初期表示時のフィルター状態</p>
              </div>
              <div className="form-control">
                <div className="mb-2">
                  <p className="text-xs text-base-content/60 mt-1">
                    {`現在: ${filterStateLabels[defaultFilterState]}`}
                  </p>
                </div>
                <select
                  className="select select-bordered select-sm"
                  value={defaultFilterState}
                  onChange={(e) => handleFilterChange(e.target.value as FilterOption)}
                >
                  {Object.entries(filterStateLabels).map(([value, label]) => (
                    <option key={value} value={value}>
                      {label}
                    </option>
                  ))}
                </select>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
