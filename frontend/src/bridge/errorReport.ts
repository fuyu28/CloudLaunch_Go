/**
 * @fileoverview エラー / ログ報告ブリッジ。
 *
 * UI を止めないよう fire-and-forget。失敗は console にだけ残す。
 */

import { ReportError, ReportLog } from "../../wailsjs/go/app/App";
import type { WindowApi } from "./types";

export function createErrorReportBridge(): WindowApi["errorReport"] {
  return {
    reportError: (payload) => {
      void ReportError({
        level: payload.level ?? "error",
        message: payload.message,
        stack: payload.stack ?? "",
        context: payload.context ?? "",
        component: payload.component ?? "",
        function: payload.function ?? "",
        data: payload.data ?? null,
        timestamp: payload.timestamp ?? new Date().toISOString(),
      }).catch((error: unknown) => {
        console.error("ReportError failed", error, payload);
      });
    },
    reportLog: (payload) => {
      void ReportLog({
        level: payload.level ?? "info",
        message: payload.message,
        component: payload.component ?? "",
        function: payload.function ?? "",
        context: payload.context ?? "",
        data: payload.data ?? null,
        timestamp: payload.timestamp ?? new Date().toISOString(),
      }).catch((error: unknown) => {
        console.error("ReportLog failed", error, payload);
      });
    },
  };
}
