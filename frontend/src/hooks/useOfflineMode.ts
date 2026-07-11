/**
 * @fileoverview オフラインモード判定用のカスタムフック
 *
 * オフラインモードの状態を管理し、ネットワーク機能の利用可否を判定します。
 */

import { useCallback } from "react";
import { useAtom } from "jotai";
import toast from "react-hot-toast";

import { offlineModeAtom } from "../state/settings";

type OfflineModeHook = {
  isOfflineMode: boolean;
  isNetworkAvailable: boolean;
  showOfflineError: (feature?: string) => void;
  checkNetworkFeature: (feature?: string) => boolean;
};

export function useOfflineMode(): OfflineModeHook {
  const [isOfflineMode] = useAtom(offlineModeAtom);

  const isNetworkAvailable = !isOfflineMode;

  const showOfflineError = useCallback((feature = "この機能"): void => {
    toast.error(`${feature}はオフラインモードでは利用できません`);
  }, []);

  const checkNetworkFeature = useCallback(
    (feature = "この機能"): boolean => {
      if (isOfflineMode) {
        showOfflineError(feature);
        return false;
      }
      return true;
    },
    [isOfflineMode, showOfflineError],
  );

  return {
    isOfflineMode,
    isNetworkAvailable,
    showOfflineError,
    checkNetworkFeature,
  };
}
