import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { HashRouter } from "react-router-dom";

import App from "./App";
import "./assets/tailwind.css";
import { createWailsBridge } from "./wailsBridge";

// テーマを初期化
const initializeTheme = (): void => {
  const savedTheme = localStorage.getItem("theme") || "light";
  document.documentElement.setAttribute("data-theme", savedTheme);
};

// アプリケーション起動時にテーマを復元
initializeTheme();

window.api = createWailsBridge();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <HashRouter>
      <App />
    </HashRouter>
  </StrictMode>,
);
