/**
 * @fileoverview useUploadAfterSession のテスト
 *
 * 「セーブが不変のセッション（sessions.json だけ変化した）でもアップロード確認
 * プロンプトが出てしまう」バグの再発防止テスト。
 * バックエンドの Status() が返す savesDiffer で pendingUpload を狭窄することを確認する。
 */

import { vi, type Mock } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";

import type { WindowApi, SyncStatusDetail } from "src/wailsBridge";
import type { ApiResult } from "src/types/result";
import type { GameType } from "src/types/game";
import { useUploadAfterSession } from "../useUploadAfterSession";

// Window型拡張
declare global {
  interface Window {
    api: WindowApi;
  }
}

const mockGetStatus: Mock = vi.fn();
const mockGetGameById: Mock = vi.fn();
const mockPush: Mock = vi.fn();
const mockResolveConflict: Mock = vi.fn();
const mockOnProgress: Mock = vi.fn().mockReturnValue(() => {});

Object.defineProperty(window, "api", {
  value: {
    database: {
      getGameById: mockGetGameById,
    },
    cloudSync: {
      status: mockGetStatus,
      push: mockPush,
      resolveConflict: mockResolveConflict,
      onProgress: mockOnProgress,
    },
  },
  writable: true,
});

const toastHandler = {
  showToast: vi.fn(),
  showLoading: vi.fn().mockReturnValue("toast-id"),
  showSuccess: vi.fn(),
  showError: vi.fn(),
};

const gameId = "game-1";

const gameFixture: GameType = {
  id: gameId,
  title: "Test Game",
  publisher: "Test Publisher",
  exePath: "/path/to/game.exe",
  saveFolderPath: "/path/to/saves",
  imagePath: "/path/to/image.jpg",
  totalPlayTime: 0,
  lastPlayed: null,
  playStatus: "playing",
  createdAt: new Date("2026-01-01"),
  currentRouteId: null,
  clearedAt: null,
};

function statusOk(detail: SyncStatusDetail): ApiResult<SyncStatusDetail> {
  return { success: true, data: detail };
}

describe("useUploadAfterSession", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockOnProgress.mockReturnValue(() => {});
  });

  it("Status returns push_needed with savesDiffer=false → pendingUpload は null のまま", async () => {
    mockGetGameById.mockResolvedValue(gameFixture);
    mockGetStatus.mockResolvedValue(
      statusOk({
        status: "push_needed",
        savesDiffer: false,
      }),
    );

    const { result } = renderHook(() => useUploadAfterSession(false, true, toastHandler));

    await act(async () => {
      await result.current.checkUploadPrompt(gameId);
    });

    expect(mockGetStatus).toHaveBeenCalledWith(gameId);
    expect(result.current.pendingUpload).toBeNull();
    expect(toastHandler.showToast).not.toHaveBeenCalled();
  });

  it("Status returns push_needed with savesDiffer=true → pendingUpload に該当ゲームが入る", async () => {
    mockGetGameById.mockResolvedValue(gameFixture);
    mockGetStatus.mockResolvedValue(
      statusOk({
        status: "push_needed",
        savesDiffer: true,
      }),
    );

    const { result } = renderHook(() => useUploadAfterSession(false, true, toastHandler));

    await act(async () => {
      await result.current.checkUploadPrompt(gameId);
    });

    await waitFor(() => {
      expect(result.current.pendingUpload).toEqual({
        gameId,
        gameTitle: gameFixture.title,
        saveFolderPath: gameFixture.saveFolderPath,
      });
    });
  });

  it("Status returns conflict with savesDiffer=false → pendingUpload は null のまま", async () => {
    mockGetGameById.mockResolvedValue(gameFixture);
    mockGetStatus.mockResolvedValue(
      statusOk({
        status: "conflict",
        savesDiffer: false,
      }),
    );

    const { result } = renderHook(() => useUploadAfterSession(false, true, toastHandler));

    await act(async () => {
      await result.current.checkUploadPrompt(gameId);
    });

    expect(result.current.pendingUpload).toBeNull();
    expect(result.current.pendingConflict).toBeNull();
  });

  it("never_synced → 初回アップロードも確認を出す", async () => {
    mockGetGameById.mockResolvedValue(gameFixture);
    mockGetStatus.mockResolvedValue(statusOk({ status: "never_synced", savesDiffer: false }));

    const { result } = renderHook(() => useUploadAfterSession(false, true, toastHandler));

    await act(async () => {
      await result.current.checkUploadPrompt(gameId);
    });

    expect(result.current.pendingUpload?.gameId).toBe(gameId);
  });

  it("「しない」を選ぶと push を呼ばずに確認を閉じる", async () => {
    mockGetGameById.mockResolvedValue(gameFixture);
    mockGetStatus.mockResolvedValue(statusOk({ status: "push_needed", savesDiffer: true }));

    const { result } = renderHook(() => useUploadAfterSession(false, true, toastHandler));

    await act(async () => {
      await result.current.checkUploadPrompt(gameId);
    });
    act(() => {
      result.current.handleSkipUploadAfterEnd();
    });

    expect(result.current.pendingUpload).toBeNull();
    expect(mockPush).not.toHaveBeenCalled();
  });

  it("conflict with savesDiffer=true → アップロード確認ではなく競合解決を出す", async () => {
    mockGetGameById.mockResolvedValue(gameFixture);
    mockGetStatus.mockResolvedValue(statusOk({ status: "conflict", savesDiffer: true }));
    mockResolveConflict.mockResolvedValue({ success: true, data: { applied: true } });

    const { result } = renderHook(() => useUploadAfterSession(false, true, toastHandler));

    await act(async () => {
      await result.current.checkUploadPrompt(gameId);
    });

    expect(result.current.pendingUpload).toBeNull();
    expect(result.current.pendingConflict?.gameId).toBe(gameId);

    await act(async () => {
      await result.current.handleResolveConflictAfterEnd(true);
    });

    expect(mockResolveConflict).toHaveBeenCalledWith(gameId, true, undefined);
    expect(mockPush).not.toHaveBeenCalled();
    expect(result.current.pendingConflict).toBeNull();
  });
});
