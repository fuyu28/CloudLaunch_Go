import { themeAtom } from "@renderer/state/settings";
import { useAtom } from "jotai";
import { useRef, useEffect, useState } from "react";
import { Toaster } from "react-hot-toast";
import { FaEdit } from "react-icons/fa";
import { FiMenu, FiCloud, FiArrowLeft } from "react-icons/fi";
import { IoIosHome, IoIosSettings } from "react-icons/io";
import { VscChromeClose, VscChromeMaximize, VscChromeMinimize } from "react-icons/vsc";
import { Outlet, NavLink, useLocation, useNavigate } from "react-router-dom";

import PlayStatusBar from "@renderer/components/game/PlayStatusBar";

export default function MainLayout(): React.JSX.Element {
  const location = useLocation();
  const navigate = useNavigate();
  const drawerRef = useRef<HTMLInputElement>(null);
  const [currentTheme] = useAtom(themeAtom);
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

  return (
    <div className="drawer drawer-mobile min-h-screen bg-base-200 wails-no-drag">
      <input id="main-drawer" type="checkbox" className="drawer-toggle" ref={drawerRef} />

      {/* サイドバー */}
      <div className="drawer-side">
        <label htmlFor="main-drawer" className="drawer-overlay bg-black/15 z-40" />

        <aside
          className="
          fixed left-0 z-50
          h-full w-56
          bg-base-100
          border-r border-base-300
          pt-4 pb-3 px-3
          rounded-tr-2xl rounded-br-2xl
          shadow-lg
          transform transition-transform duration-200 ease-out
        "
        >
          <div className="flex flex-col h-full">
            {/* ブランド */}
            <div className="flex items-center gap-2 px-2 pb-4 mb-2 border-b border-base-200">
              <FiCloud className="text-xl text-primary" />
              <span className="font-semibold tracking-tight">CloudLaunch</span>
            </div>

            {/* 上部メニュー */}
            <ul className="space-y-1">
              <li>
                <NavLink
                  to="/"
                  className={({ isActive }) =>
                    `flex items-center w-full p-3 rounded-md ${
                      isActive ? "bg-primary text-primary-content font-medium" : "hover:bg-base-300"
                    }`
                  }
                  onClick={closeDrawer}
                >
                  <IoIosHome className="mr-2 text-lg" />
                  <span className="flex-1">ホーム</span>
                </NavLink>
              </li>
              <li>
                <NavLink
                  to="/memo"
                  className={({ isActive }) =>
                    `flex items-center w-full p-3 rounded-md ${
                      isActive ? "bg-primary text-primary-content font-medium" : "hover:bg-base-300"
                    }`
                  }
                  onClick={closeDrawer}
                >
                  <FaEdit className="mr-2 text-lg" />
                  <span className="flex-1">メモ</span>
                </NavLink>
              </li>
              <li>
                <NavLink
                  to="/cloud"
                  className={({ isActive }) =>
                    `flex items-center w-full p-3 rounded-md ${
                      isActive ? "bg-primary text-primary-content font-medium" : "hover:bg-base-300"
                    }`
                  }
                  onClick={closeDrawer}
                >
                  <FiCloud className="mr-2 text-lg" />
                  <span className="flex-1">クラウド</span>
                </NavLink>
              </li>
            </ul>

            {/* 下部メニューは mt-auto で下端へ */}
            <ul className="space-y-1 mt-auto">
              <li>
                <NavLink
                  to="/settings"
                  className={({ isActive }) =>
                    `flex items-center w-full p-3 rounded-md ${
                      isActive ? "bg-primary text-primary-content font-medium" : "hover:bg-base-300"
                    }`
                  }
                  onClick={closeDrawer}
                >
                  <IoIosSettings className="mr-2 text-lg" />
                  <span className="flex-1">設定</span>
                </NavLink>
              </li>
            </ul>
          </div>
        </aside>
      </div>

      {/* メイン */}
      <div className="drawer-content flex flex-col h-screen">
        {/* ↓ ここをカスタムタイトルバーに */}
        <header
          className="
          relative
          flex items-center
          h-12 bg-base-100 border-b border-base-300
          select-none wails-drag
        "
        >
          {/* ドロワー開閉ボタンは no-drag */}
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
              "
              aria-label="メニューを開く"
            >
              <FiMenu size={22} />
            </label>
          )}

          {/* 中央タイトルもドラッグ可能 */}
          <h1 className="flex-1 text-center text-lg font-semibold tracking-tight leading-none">
            {pageLabel}
          </h1>

          {/* ウィンドウ操作ボタン群（Windows のフレームレス時のみ）。
              macOS / Linux はネイティブ装飾を使うため表示しない。 */}
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

        {/* ページ固有部分 */}
        <main className="flex-1 pt-6 overflow-y-auto scrollbar-thin scrollbar-thumb-base-content/30 scrollbar-track-transparent min-h-0">
          <Outlet />
        </main>

        {/* プレイ状況バー */}
        <PlayStatusBar />
      </div>

      <Toaster position="bottom-center" />
    </div>
  );
}
