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

// 初期描画のテーマちらつきを防ぐため、描画前に data-theme を入れる。
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

initializeTheme();

window.api = createWailsBridge();

// bridge 未初期化のうちにハンドラを付けると報告先が無い。
installGlobalErrorHandlers();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <HashRouter>
      <App />
    </HashRouter>
  </StrictMode>,
);
