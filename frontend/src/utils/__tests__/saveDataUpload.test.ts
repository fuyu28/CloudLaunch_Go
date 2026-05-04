import { beforeEach, describe, expect, it, vi } from "vitest";

import { downloadSaveDataAndSyncMetadata, uploadSaveDataAndSyncHash } from "../saveDataUpload";

describe("saveDataUpload", () => {
  beforeEach(() => {
    globalThis.window = {
      api: {
        saveData: {
          upload: {
            uploadSaveDataFolder: vi.fn(),
          },
          download: {
            downloadSaveData: vi.fn(),
          },
          hash: {
            computeLocalHash: vi.fn(),
            saveCloudHash: vi.fn(),
          },
        },
        cloudSync: {
          syncGame: vi.fn(),
        },
      },
    } as typeof window;
  });

  it("アップロード後にゲーム同期を行う", async () => {
    vi.mocked(window.api.saveData.upload.uploadSaveDataFolder).mockResolvedValue({ success: true });
    vi.mocked(window.api.saveData.hash.computeLocalHash).mockResolvedValue({
      success: true,
      data: "local-hash",
    });
    vi.mocked(window.api.saveData.hash.saveCloudHash).mockResolvedValue({ success: true });
    vi.mocked(window.api.cloudSync.syncGame).mockResolvedValue({ success: true });

    const result = await uploadSaveDataAndSyncHash({
      gameId: "game-1",
      saveFolderPath: "/tmp/save",
      localUpdatedAt: "2026-05-04T12:34:56Z",
    });

    expect(result).toEqual({ success: true });
    expect(window.api.saveData.upload.uploadSaveDataFolder).toHaveBeenCalledWith(
      "/tmp/save",
      "games/game-1/save_data",
    );
    expect(window.api.saveData.hash.saveCloudHash).toHaveBeenCalledWith(
      "game-1",
      "local-hash",
      "2026-05-04T12:34:56Z",
    );
    expect(window.api.cloudSync.syncGame).toHaveBeenCalledWith("game-1");
  });

  it("ダウンロード後にゲーム同期を行う", async () => {
    vi.mocked(window.api.saveData.download.downloadSaveData).mockResolvedValue({ success: true });
    vi.mocked(window.api.cloudSync.syncGame).mockResolvedValue({ success: true });

    const result = await downloadSaveDataAndSyncMetadata({
      gameId: "game-2",
      saveFolderPath: "/tmp/save",
    });

    expect(result).toEqual({ success: true });
    expect(window.api.saveData.download.downloadSaveData).toHaveBeenCalledWith(
      "/tmp/save",
      "games/game-2/save_data",
    );
    expect(window.api.cloudSync.syncGame).toHaveBeenCalledWith("game-2");
  });
});
