/**
 * @fileoverview データベース操作ブリッジ (ゲーム・セッション)。
 */

import {
  ListGames,
  GetGameByID,
  CreateGame,
  UpdateGame,
  DeleteGame,
  CreateSession,
  ListSessionsByGame,
  UpdateSessionName,
  DeleteSession,
} from "../../wailsjs/go/app/App";
import { toGameType, toPlaySessionType, toApiResultVoid } from "./helpers";
import type { modelsServices, modelsTime } from "./helpers";
import type { GameType, PlayStatus } from "src/types/game";
import type { WindowApi } from "./types";

export function createDatabaseBridge(): WindowApi["database"] {
  return {
    listGames: async (searchWord, filter, sort, sortDirection) => {
      const result = await ListGames(searchWord, filter, sort, sortDirection ?? "asc");
      return result.success && result.data ? result.data.map(toGameType) : [];
    },
    getGameById: async (id) => {
      const result = await GetGameByID(id);
      if (!result.success) {
        return undefined;
      }
      return result.data ? toGameType(result.data) : undefined;
    },
    createGame: async (game) => {
      const payload = {
        Title: game.title,
        Publisher: game.publisher,
        ImagePath: game.imagePath ?? undefined,
        ExePath: game.exePath,
        SaveFolderPath: game.saveFolderPath ?? undefined,
      };
      const result = await CreateGame(payload);
      return toApiResultVoid(result, "エラー");
    },
    updateGame: async (id, game) => {
      const payload = {
        Title: game.title,
        Publisher: game.publisher,
        ImagePath: game.imagePath ?? undefined,
        ExePath: game.exePath,
        SaveFolderPath: game.saveFolderPath ?? undefined,
        // 空文字を渡すとバックエンド側 (UpdateGame) は playStatus を上書きしない。
        // 一般的なゲーム編集では playStatus を変更しないため空文字を維持する。
        PlayStatus: "" as string,
        ClearedAt: undefined,
        CurrentRouteID: undefined,
      };
      const result = await UpdateGame(id, payload as unknown as modelsServices.GameUpdateInput);
      return toApiResultVoid(result, "エラー");
    },
    deleteGame: async (id) => toApiResultVoid(await DeleteGame(id), "エラー"),
    updatePlayStatus: async (gameId, playStatus: PlayStatus) => {
      const current = await GetGameByID(gameId);
      if (!current.success || !current.data) {
        return { success: false, message: current.error?.message ?? "ゲーム取得に失敗しました" };
      }
      const game = toGameType(current.data);
      const clearedAt = playStatus === "played" ? new Date() : null;
      const updatePayload = {
        Title: game.title,
        Publisher: game.publisher,
        ImagePath: game.imagePath ?? undefined,
        ExePath: game.exePath,
        SaveFolderPath: game.saveFolderPath ?? undefined,
        PlayStatus: playStatus,
        ClearedAt: clearedAt !== null ? (clearedAt as unknown as modelsTime.Time) : undefined,
        CurrentRouteID: game.currentRouteId ?? undefined,
      };
      const result = await UpdateGame(
        gameId,
        updatePayload as unknown as modelsServices.GameUpdateInput,
      );
      if (!result.success) {
        return { success: false, message: result.error?.message ?? "エラー" };
      }
      const updated = await GetGameByID(gameId);
      if (!updated.success) {
        return { success: false, message: updated.error?.message ?? "エラー" };
      }
      return {
        success: true,
        data: updated.data ? toGameType(updated.data) : (undefined as unknown as GameType),
      };
    },
    createSession: async (duration, gameId, sessionName) => {
      const payload = {
        GameID: gameId,
        PlayedAt: new Date() as unknown as modelsTime.Time,
        Duration: duration,
        SessionName: sessionName ?? undefined,
        RouteID: undefined,
      };
      const result = await CreateSession(payload as unknown as modelsServices.SessionInput);
      return toApiResultVoid(result, "エラー");
    },
    getPlaySessions: async (gameId) => {
      const result = await ListSessionsByGame(gameId);
      return result.success
        ? { success: true, data: (result.data ?? []).map(toPlaySessionType) }
        : { success: false, message: result.error?.message ?? "エラー" };
    },
    updateSessionName: async (sessionId, sessionName) =>
      toApiResultVoid(await UpdateSessionName(sessionId, sessionName), "エラー"),
    deletePlaySession: async (sessionId) =>
      toApiResultVoid(await DeleteSession(sessionId), "エラー"),
  };
}
