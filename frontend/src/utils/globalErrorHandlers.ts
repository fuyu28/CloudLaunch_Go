/**
 * @fileoverview 未捕捉エラーのグローバル捕捉
 *
 * try/catch で囲まれていない同期例外（window の error イベント）と、
 * catch されない Promise の拒否（unhandledrejection）をログに記録する。
 */

import { logger } from "./logger";

/**
 * unknown な値を Error に正規化する。
 */
const toError = (value: unknown, fallbackMessage: string): Error => {
  if (value instanceof Error) {
    return value;
  }
  if (typeof value === "string") {
    return new Error(value);
  }
  try {
    return new Error(`${fallbackMessage}: ${JSON.stringify(value)}`);
  } catch {
    return new Error(fallbackMessage);
  }
};

let installed = false;

/**
 * グローバルエラーハンドラを登録する。多重登録は無視する。
 * window.api 初期化後（main.tsx）に1回だけ呼ぶこと。
 */
export const installGlobalErrorHandlers = (): void => {
  if (installed) {
    return;
  }
  installed = true;

  window.addEventListener("error", (event: ErrorEvent) => {
    const error = event.error ?? toError(event.message, "Uncaught error");
    logger.error("未捕捉エラー", {
      component: "window",
      function: "onerror",
      error: error instanceof Error ? error : toError(error, "Uncaught error"),
      data: {
        filename: event.filename,
        lineno: event.lineno,
        colno: event.colno,
      },
    });
  });

  window.addEventListener("unhandledrejection", (event: PromiseRejectionEvent) => {
    logger.error("未処理の Promise 拒否", {
      component: "window",
      function: "onunhandledrejection",
      error: toError(event.reason, "Unhandled promise rejection"),
    });
  });
};
