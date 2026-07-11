/**
 * @fileoverview レンダラープロセス用ログユーティリティ
 *
 * このファイルは、レンダラープロセスで使用するログ機能を提供します。
 */

import { logLevelManager, type LogLevel } from "./logLevel";

/**
 * ログメタデータ
 */
export interface LogMetadata {
  component?: string;
  function?: string;
  context?: string;
  error?: Error;
  data?: unknown;
}

/**
 * レンダラープロセス用ログクラス
 *
 * メインプロセスのログシステムと連携して
 * 統一的なログ出力を提供します。
 */
class RendererLogger {
  private isDevelopment: boolean;

  constructor() {
    this.isDevelopment = process.env.NODE_ENV === "development";
  }

  /**
   * デバッグログを出力
   * ログレベル設定と開発環境をチェック
   */
  debug(message: string, metadata?: LogMetadata): void {
    if (!logLevelManager.shouldLog("debug")) {
      return;
    }

    if (this.isDevelopment) {
      this.logToConsole("debug", message, metadata);
    }
    // 本番でも level 次第で出す（完全黙りにしない）。
    this.logToMain("debug", message, metadata);
  }

  /**
   * 情報ログを出力
   */
  info(message: string, metadata?: LogMetadata): void {
    if (!logLevelManager.shouldLog("info")) {
      return;
    }

    this.logToConsole("info", message, metadata);
    this.logToMain("info", message, metadata);
  }

  /**
   * 警告ログを出力
   */
  warn(message: string, metadata?: LogMetadata): void {
    if (!logLevelManager.shouldLog("warn")) {
      return;
    }

    this.logToConsole("warn", message, metadata);
    this.logToMain("warn", message, metadata);
  }

  /**
   * エラーログを出力
   */
  error(message: string, metadata?: LogMetadata): void {
    if (!logLevelManager.shouldLog("error")) {
      return;
    }

    this.logToConsole("error", message, metadata);
    this.logToMain("error", message, metadata);

    // コンソールだけでなくエラーバウンダリ経路にも載せる。
    if (metadata?.error && window.api?.errorReport?.reportError) {
      window.api.errorReport.reportError({
        message: metadata.error.message,
        stack: metadata.error.stack || "",
        level: "error",
        context: `${metadata.component || "unknown"}:${metadata.function || "unknown"} - ${message}`,
        component: metadata.component,
        function: metadata.function,
        data: metadata.data,
        timestamp: new Date().toISOString(),
      });
    }
  }

  /**
   * コンソールにログを出力
   */
  private logToConsole(level: LogLevel, message: string, metadata?: LogMetadata): void {
    const timestamp = new Date().toISOString();
    const componentInfo = metadata?.component
      ? `[${metadata.component}${metadata.function ? `:${metadata.function}` : ""}]`
      : "";

    const logMessage = `${timestamp} ${componentInfo} ${message}`;

    switch (level) {
      case "debug":
        console.log(`🐛 ${logMessage}`, metadata?.data);
        break;
      case "info":
        console.log(`ℹ️ ${logMessage}`, metadata?.data);
        break;
      case "warn":
        console.warn(`⚠️ ${logMessage}`, metadata?.data);
        break;
      case "error":
        console.error(`❌ ${logMessage}`, metadata?.error || metadata?.data);
        break;
    }
  }

  /**
   * メインプロセスにログを送信
   */
  private logToMain(
    level: "debug" | "info" | "warn" | "error",
    message: string,
    metadata?: LogMetadata,
  ): void {
    // bridge 未準備時は送らない（起動直後の例外を落とさないため握りつぶしではなくスキップ）。
    if (window.api?.errorReport?.reportLog) {
      window.api.errorReport.reportLog({
        level,
        message,
        component: metadata?.component,
        function: metadata?.function,
        context: metadata?.context,
        data: metadata?.data,
        timestamp: new Date().toISOString(),
      });
    }
  }

  /**
   * ユーザーアクションをログに記録
   */
  logUserAction(action: string, details?: Record<string, unknown>): void {
    this.info(`ユーザーアクション: ${action}`, {
      component: "UserAction",
      data: details,
    });
  }

  /**
   * パフォーマンス測定を開始
   */
  startPerformanceTimer(label: string): () => void {
    const startTime = performance.now();
    return () => {
      const duration = performance.now() - startTime;
      this.info(`パフォーマンス測定: ${label}`, {
        component: "Performance",
        data: { duration: `${duration.toFixed(2)}ms` },
      });
    };
  }
}

export const logger = new RendererLogger();

if (process.env.NODE_ENV === "development") {
  (window as unknown as Window & { logger: typeof logger }).logger = logger;
}
