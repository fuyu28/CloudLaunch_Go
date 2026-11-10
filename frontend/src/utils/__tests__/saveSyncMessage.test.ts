import { describe, expect, it } from "vitest";

import { buildSaveSyncMessage } from "../saveSyncMessage";

const format = (date: Date | string | number | null | undefined): string =>
  date instanceof Date ? date.toISOString() : "不明";

const remoteMeta = {
  "game.json": "g",
  "sessions.json": "s",
  saves: "v",
  deviceName: "OTHER-PC",
  createdAt: new Date("2026-10-31T12:00:00Z"),
  fileCount: 1,
  totalSize: 10,
};

describe("buildSaveSyncMessage", () => {
  it("フォルダの最終更新とクラウドのアップロード日時・端末を載せる", () => {
    const message = buildSaveSyncMessage(format, "Novel", {
      hasLocalBaseline: true,
      local: {
        folderPath: "D:\\games",
        fileCount: 1,
        totalSize: 10,
        lastModified: new Date("2026-11-01T12:00:00Z"),
      },
      remoteMeta,
    });
    expect(message).toContain("別のPCからセーブデータがアップロードされています");
    expect(message).toContain("このPCのセーブ最終更新: 2026-11-01T12:00:00.000Z");
    expect(message).toContain("クラウドへのアップロード: 2026-10-31T12:00:00.000Z（OTHER-PC）");
  });

  it("このPCで未同期のときはクラウドで変更があったとは言わない", () => {
    const message = buildSaveSyncMessage(format, "Novel", {
      hasLocalBaseline: false,
      local: { folderPath: "D:\\games", fileCount: 0, totalSize: 0, lastModified: null },
      remoteMeta,
    });
    expect(message).toContain("このPCでまだ同期しておらず");
    expect(message).not.toContain("別のPCから");
    expect(message).toContain("このPCのセーブ最終更新: ファイルなし");
  });
});
