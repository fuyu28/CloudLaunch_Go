/**
 * @fileoverview 批評空間（ErogameScape）連携ブリッジ。
 *
 * ID / タイトル検索の入力検証と URL 組み立てをフロント側で行い、Go に委譲する。
 */

import { FetchFromErogameScape, SearchErogameScape } from "../../wailsjs/go/app/App";
import { getErrorMessage } from "./helpers";
import type { GameImport } from "src/types/game";
import type { ErogameScapeSearchResult } from "src/types/erogamescape";
import type { WindowApi } from "./types";

export function createErogameScapeBridge(): WindowApi["erogameScape"] {
  return {
    fetchById: async (id) => {
      const trimmed = id.trim();
      if (!trimmed) {
        return { success: false, message: "批評空間IDを入力してください" };
      }
      const url = `https://erogamescape.dyndns.org/~ap2/ero/toukei_kaiseki/game.php?game=${encodeURIComponent(
        trimmed,
      )}`;
      try {
        const result = await FetchFromErogameScape(url);
        return { success: true, data: result as GameImport };
      } catch (error) {
        return {
          success: false,
          message: getErrorMessage(error, "批評空間からの取得に失敗しました"),
        };
      }
    },
    searchByTitle: async (query, pageUrl) => {
      const trimmed = query.trim();
      if (!trimmed && !pageUrl) {
        return { success: false, message: "検索ワードを入力してください" };
      }
      try {
        const result = await SearchErogameScape(trimmed, pageUrl ?? "");
        return { success: true, data: result as ErogameScapeSearchResult };
      } catch (error) {
        return { success: false, message: getErrorMessage(error, "批評空間の検索に失敗しました") };
      }
    },
  };
}
