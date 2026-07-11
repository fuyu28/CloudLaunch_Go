import {
  themeAtom,
  offlineModeAtom,
  autoTrackingAtom,
  transferConcurrencyAtom,
} from "@renderer/state/settings";
import { useAtom, useAtomValue } from "jotai";
import { useRef, useEffect, useState } from "react";
import { Toaster } from "react-hot-toast";
import { FaEdit } from "react-icons/fa";
import { FiMenu, FiCloud, FiArrowLeft } from "react-icons/fi";
import { IoIosHome, IoIosSettings } from "react-icons/io";
import { VscChromeClose, VscChromeMaximize, VscChromeMinimize } from "react-icons/vsc";
import { Outlet, NavLink, useLocation, useNavigate } from "react-router-dom";

import PlayStatusBar from "@renderer/components/game/PlayStatusBar";

function navClassName({ isActive }: { isActive: boolean }): string {
  return `flex items-center w-full p-3 rounded-md ${
    isActive ? "bg-primary text-primary-content font-medium" : "hover:bg-base-300"
  }`;
}

export default function MainLayout(): React.JSX.Element {
  const location = useLocation();
  const navigate = useNavigate();
  const drawerRef = useRef<HTMLInputElement>(null);
  const [currentTheme] = useAtom(themeAtom);
  const offlineMode = useAtomValue(offlineModeAtom);
  const autoTracking = useAtomValue(autoTrackingAtom);
  const transferConcurrency = useAtomValue(transferConcurrencyAtom);
  // Windows のみフレームレス＝独自のウィンドウ操作ボタンを表示する。
  // macOS / Linux はネイティブ装飾を使うため非表示にする。
  const [isWindows, setIsWindows] = useState(false);
  const isHome = location.pathname === "/";
  const isSettings = location.pathname === "/settings";
  const isMemo = location.pathname === "/memo" || location.pathname.startsWith("/memo/");
  const isCloud = location.pathname === "/cloud";
  const isGameDetail = location.pathname.startsWith("/game/");

  const pageMap: [boolean, string][] = [
    [isHome, "ホーム"],
    [isSettings, "設定"],
    [isCloud, "クラウド"],
    [isMemo, "メモ"],
  ];

  const pageLabel = pageMap.find(([cond]) => cond)?.[1] ?? "";

  const closeDrawer = (): void => {
    if (drawerRef.current) drawerRef.current.checked = false;
  };

  // テーマ初期化：アプリケーション起動時にHTMLのdata-theme属性を設定
  useEffect(() => {
    document.documentElement.setAttribute("data-theme", currentTheme);
  }, [currentTheme]);

  // 実行プラットフォームを判定（独自ウィンドウ操作ボタンの出し分け用）
  useEffect(() => {
    let active = true;
    void window.api.window.getPlatform().then((platform) => {
      if (active) setIsWindows(platform === "windows");
    });
    return () => {
      active = false;
    };
  }, []);

  // 起動時にバックエンドへ localStorage 永続設定を再同期する。
  // バックエンドはプロセス起動毎に既定値に戻るため、ここで再宣言しないと
  // autoTracking OFF なのに監視が動き続けたり、offline / 並列度が無視されたりする。
  // 初回マウント時のみ実行する（atom 更新の度に呼ぶのはハンドラ側の責務）。
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    void window.api.settings.updateOfflineMode(offlineMode);
    void window.api.settings.updateAutoTracking(autoTracking);
    void window.api.settings.updateUploadConcurrency(transferConcurrency);
  }, []);

  return (
    // daisyUI 5 では drawer-mobile が廃止済み。lg:drawer-open で大画面は常時サイドバー表示。
    <div className="drawer lg:drawer-open min-h-screen bg-base-200 wails-no-drag">
      <input id="main-drawer" type="checkbox" className="drawer-toggle" ref={drawerRef} />

      {/* メイン（DaisyUI 推奨順: toggle → content → side） */}
      <div className="drawer-content flex flex-col h-screen">
        <header
          className="
          relative
          flex items-center
          h-12 bg-base-100 border-b border-base-300
          select-none wails-drag
        "
        >
          {isGameDetail ? (
            <button
              type="button"
              onClick={() => navigate("/")}
              className="
                absolute inset-y-0 left-0
                flex items-center justify-center
                h-full w-10
                btn btn-ghost p-0 focus:outline-none
              hover:bg-base-300
                wails-no-drag
              "
              aria-label="ホームに戻る"
            >
              <FiArrowLeft size={22} />
            </button>
          ) : (
            <label
              htmlFor="main-drawer"
              className="
                absolute inset-y-0 left-0
                flex items-center justify-center
                h-full w-10
                btn btn-ghost p-0 focus:outline-none
              hover:bg-base-300
                wails-no-drag
                lg:hidden
              "
              aria-label="メニューを開く"
            >
              <FiMenu size={22} />
            </label>
          )}

          <h1 className="flex-1 text-center text-lg font-semibold tracking-tight leading-none">
            {pageLabel}
          </h1>

          {isWindows && (
            <div className="absolute inset-y-0 right-0 flex wails-no-drag">
              <button
                onClick={() => window.api.window.minimize()}
                aria-label="最小化"
                className="h-full window-control flex items-center justify-center text-base-content/70 hover:bg-base-200 hover:text-base-content transition-colors"
              >
                <VscChromeMinimize className="text-[15px]" />
              </button>
              <button
                onClick={() => window.api.window.toggleMaximize()}
                aria-label="最大化"
                className="h-full window-control flex items-center justify-center text-base-content/70 hover:bg-base-200 hover:text-base-content transition-colors"
              >
                <VscChromeMaximize className="text-[15px]" />
              </button>
              <button
                onClick={() => window.api.window.close()}
                aria-label="閉じる"
                className="h-full window-control flex items-center justify-center text-base-content/70 hover:bg-error hover:text-error-content transition-colors"
              >
                <VscChromeClose className="text-[15px]" />
              </button>
            </div>
          )}
        </header>

        <main className="flex-1 pt-6 overflow-y-auto scrollbar-thin scrollbar-thumb-base-content/30 scrollbar-track-transparent min-h-0">
          <Outlet />
        </main>

        <PlayStatusBar />
      </div>

      {/* サイドバー */}
      <div className="drawer-side z-40">
        <label htmlFor="main-drawer" aria-label="メニューを閉じる" className="drawer-overlay" />

        <aside className="flex flex-col min-h-full w-56 bg-base-100 border-r border-base-300 pt-4 pb-3 px-3">
          <div className="flex items-center gap-2 px-2 pb-4 mb-2 border-b border-base-200">
            <FiCloud className="text-xl text-primary" />
            <span className="font-semibold tracking-tight">CloudLaunch</span>
          </div>

          <ul className="space-y-1">
            <li>
              <NavLink to="/" className={navClassName} onClick={closeDrawer}>
                <IoIosHome className="mr-2 text-lg" />
                <span className="flex-1">ホーム</span>
              </NavLink>
            </li>
            <li>
              <NavLink to="/memo" className={navClassName} onClick={closeDrawer}>
                <FaEdit className="mr-2 text-lg" />
                <span className="flex-1">メモ</span>
              </NavLink>
            </li>
            <li>
              <NavLink to="/cloud" className={navClassName} onClick={closeDrawer}>
                <FiCloud className="mr-2 text-lg" />
                <span className="flex-1">クラウド</span>
              </NavLink>
            </li>
          </ul>

          <ul className="space-y-1 mt-auto">
            <li>
              <NavLink to="/settings" className={navClassName} onClick={closeDrawer}>
                <IoIosSettings className="mr-2 text-lg" />
                <span className="flex-1">設定</span>
              </NavLink>
            </li>
          </ul>
        </aside>
      </div>

      <Toaster position="bottom-center" />
    </div>
  );
}
