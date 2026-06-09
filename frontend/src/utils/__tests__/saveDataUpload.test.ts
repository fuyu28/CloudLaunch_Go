import { beforeEach, describe, expect, it, vi } from "vitest";

import { downloadSaveDataAndSyncMetadata, uploadSaveDataAndSyncHash } from "../saveDataUpload";

describe("saveDataUpload", () => {
  beforeEach(() => {
    globalThis.window = {
      api: {
        cloudSync: {
          push: vi.fn(),
          pull: vi.fn(),
        },
      },
    } as typeof window;
  });

  it("アップロードで cloudSync.push を呼ぶ", async () => {
    vi.mocked(window.api.cloudSync.push).mockResolvedValue({ success: true });

    const result = await uploadSaveDataAndSyncHash({
      gameId: "game-1",
      saveFolderPath: "/tmp/save",
    });

    expect(result).toEqual({ success: true });
    expect(window.api.cloudSync.push).toHaveBeenCalledWith("game-1");
  });

  it("ダウンロードで cloudSync.pull を呼ぶ", async () => {
    vi.mocked(window.api.cloudSync.pull).mockResolvedValue({ success: true });

    const result = await downloadSaveDataAndSyncMetadata({
      gameId: "game-2",
      saveFolderPath: "/tmp/save",
    });

    expect(result).toEqual({ success: true });
    expect(window.api.cloudSync.pull).toHaveBeenCalledWith("game-2");
  });
});
