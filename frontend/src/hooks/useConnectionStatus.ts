/**
 * @fileoverview 接続状態管理フック
 *
 * このフックは、クラウドストレージ（R2/S3）への接続状態を管理します。
 */

import { useCallback, useEffect, useRef, useState } from "react";

import { useValidateCreds } from "./useValidCreds";
import type { AsyncStatus } from "src/types/common";

export type ConnectionStatusResult = {
  status: AsyncStatus;
  message: string | undefined;
  check: () => Promise<void>;
};

export function useConnectionStatus(): ConnectionStatusResult {
  const validateCreds = useValidateCreds();
  const [status, setStatus] = useState<AsyncStatus>("loading");
  const [message, setMessage] = useState<string | undefined>(undefined);
  // アンマウント後の setState を防ぐガード。外部から呼ばれる check() でも共通に参照する。
  const mountedRef = useRef(true);

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
