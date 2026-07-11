/**
 * @fileoverview 接続状態管理フック
 *
 * このフックは、クラウドストレージ（R2/S3）への接続状態を管理します。
 * 主な機能：
 * - 認証情報の検証
 * - 接続状態の監視
 * - エラーメッセージの管理
 * - リアルタイムな接続確認
 *
 * 使用例：
 * ```tsx
 * const { status, message, check } = useConnectionStatus()
 * ```
 */

import { useCallback, useEffect, useRef, useState } from "react";

import { useValidateCreds } from "./useValidCreds";
import type { AsyncStatus } from "src/types/common";

/**
 * 接続状態フックの戻り値
 */
export type ConnectionStatusResult = {
  /** 接続状態 */
  status: AsyncStatus;
  /** エラーメッセージ */
  message: string | undefined;
  /** 接続確認関数 */
  check: () => Promise<void>;
};

/**
 * 接続状態管理フック
 *
 * クラウドストレージへの接続状態を管理し、認証情報の有効性を確認します。
 *
 * @returns 接続状態と確認機能
 */
export function useConnectionStatus(): ConnectionStatusResult {
  const validateCreds = useValidateCreds();
  const [status, setStatus] = useState<AsyncStatus>("loading");
  const [message, setMessage] = useState<string | undefined>(undefined);
  // アンマウント後の setState を防ぐガード。外部から呼ばれる check() でも共通に参照する。
  const mountedRef = useRef(true);

  /**
   * 接続状態を確認する関数
   *
   * 認証情報の有効性を検証し、接続状態を更新します。
   * アンマウント後の呼び出しでは setState を行わない。
   */
  const check: () => Promise<void> = useCallback(async () => {
    if (!mountedRef.current) return;
    setStatus("loading");
    const ok = await validateCreds();
    if (!mountedRef.current) return;
    if (ok) {
      setStatus("success");
      setMessage(undefined);
    } else {
      setStatus("error");
      setMessage("クレデンシャルが有効ではありません");
    }
  }, [validateCreds]);

  useEffect(() => {
    mountedRef.current = true;
    void check();
    return () => {
      mountedRef.current = false;
    };
  }, [check]);

  return { status, message, check };
}
