/**
 * @fileoverview フロントエンドエントリポイント
 *
 * Wails ブリッジ初期化、ルーティング、グローバルエラーハンドラを起動する。
 */

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { HashRouter } from "react-router-dom";

import App from "./App";
import "./assets/tailwind.css";
import { createWailsBridge } from "./wailsBridge";
import { installGlobalErrorHandlers } from "./utils/globalErrorHandlers";

// テーマを初期化（初期描画のちらつき防止）
const initializeTheme = (): void => {
  // jotai の atomWithStorage は JSON 文字列（例: "light"）で保存するため、
  // クォートを除去して data-theme に渡す。未設定時は既定テーマを使う。
  const raw = localStorage.getItem("theme");
  let savedTheme = "cloudlaunch";
  if (raw) {
    try {
      savedTheme = JSON.parse(raw) as string;
    } catch {
      savedTheme = raw;
    }
  }
  document.documentElement.setAttribute("data-theme", savedTheme);
};

// アプリケーション起動時にテーマを復元
initializeTheme();

window.api = createWailsBridge();

// window.api 初期化後に未捕捉エラーのグローバル捕捉を有効化する
installGlobalErrorHandlers();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <HashRouter>
      <App />
    </HashRouter>
  </StrictMode>,
);
