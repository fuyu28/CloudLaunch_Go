/**
 * @fileoverview 認証情報検証フック
 *
 * このフックは、クラウドストレージ認証情報の検証機能を提供します。
 */

import { isValidCredsAtom } from "@renderer/state/credentials";
import { useSetAtom } from "jotai";
import { useCallback } from "react";

import { logger } from "@renderer/utils/logger";

export function useValidateCreds(): () => Promise<boolean> {
  const setIsValidCreds = useSetAtom(isValidCredsAtom);

  const validate = useCallback(async () => {
    try {
      const result = await window.api.credential.validateSavedCredential();
      const { success } = result;
      setIsValidCreds(success);
      if (!success) {
        logger.error("Credential validation failed:", {
          component: "useValidCreds",
          function: "unknown",
          data: result.message ?? "不明なエラー",
        });
      }
      return success;
    } catch {
      setIsValidCreds(false);
      return false;
    }
  }, [setIsValidCreds]);

  return validate;
}
