import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

import SyncStatusModal from "./SyncStatusModal";
import type { SyncStatusDetail } from "src/wailsBridge";

afterEach(cleanup);

const baseDetail: SyncStatusDetail = {
  status: "push_needed",
  savesDiffer: true,
  hasLocalBaseline: true,
};

function renderModal(
  overrides: Record<string, unknown> = {},
  detail: Partial<SyncStatusDetail> = {},
) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    gameTitle: "Test game",
    detail: { ...baseDetail, ...detail },
    hasSaveFolder: true,
    isProcessing: false,
    onUpload: vi.fn(),
    onDownload: vi.fn(),
    onForceUpload: vi.fn(),
    onForceDownload: vi.fn(),
    ...overrides,
  };
  render(<SyncStatusModal {...props} />);
  return props;
}

function setup(overrides: Record<string, unknown> = {}, detail: Partial<SyncStatusDetail> = {}) {
  const props = renderModal(overrides, detail);
  fireEvent.click(screen.getByText("強制同期"));
  return props;
}

describe("状態の表示", () => {
  it.each([
    [{ status: "in_sync", savesDiffer: false }, "クラウドと一致しています"],
    [{ status: "push_needed", savesDiffer: true }, "このPCに未アップロードの変更があります"],
    [{ status: "push_needed", savesDiffer: false }, "プレイ記録だけが未アップロードです"],
    [{ status: "pull_needed", savesDiffer: true }, "クラウドに未取得の変更があります"],
    [
      { status: "pull_needed", savesDiffer: true, hasLocalBaseline: false },
      "このPCではまだ同期していません",
    ],
    [{ status: "never_synced", savesDiffer: false }, "クラウドにデータがありません"],
  ] as const)("%j → %s", (detail, title) => {
    renderModal({}, detail);
    expect(screen.getByText(title)).toBeInTheDocument();
  });

  it("新旧の比較に読める文言を出さない", () => {
    for (const status of ["push_needed", "pull_needed"] as const) {
      renderModal({}, { status });
      expect(document.body.textContent).not.toMatch(/より新しい/);
      cleanup();
    }
  });

  it("判定に使ったセーブフォルダとクラウドのアップロード情報を表示する", () => {
    renderModal(
      {},
      {
        status: "in_sync",
        local: {
          folderPath: "D:\\games\\Novel\\savedata",
          fileCount: 3,
          totalSize: 2048,
          lastModified: new Date("2026-11-01T12:00:00Z"),
        },
        remoteMeta: {
          "game.json": "g",
          "sessions.json": "s",
          saves: "v",
          deviceName: "OTHER-PC",
          createdAt: new Date("2026-10-31T12:00:00Z"),
          fileCount: 3,
          totalSize: 2048,
        },
      },
    );
    expect(screen.getByText("D:\\games\\Novel\\savedata")).toBeInTheDocument();
    expect(screen.getAllByText(/^3件/)).toHaveLength(2);
    expect(screen.getByText("OTHER-PC")).toBeInTheDocument();
    expect(screen.queryByText("情報なし")).toBeNull();
  });

  it("フォルダにファイルが無いときは最終更新を出さない", () => {
    renderModal(
      {},
      { local: { folderPath: "C:\\save", fileCount: 0, totalSize: 0, lastModified: null } },
    );
    expect(screen.getByText("なし")).toBeInTheDocument();
    expect(screen.getByText("—")).toBeInTheDocument();
  });
});

describe("強制同期", () => {
  it.each(["push", "pull"] as const)("Force %s は確認後に選んだ方向だけ実行する", (direction) => {
    const props = setup();
    fireEvent.click(
      screen.getByRole("button", { name: direction === "push" ? /Force Push/ : /Force Pull/ }),
    );
    expect(props.onForceUpload).not.toHaveBeenCalled();
    expect(props.onForceDownload).not.toHaveBeenCalled();
    expect(screen.getByRole("alert").textContent).toContain(
      direction === "push" ? "クラウド側の変更" : "このPC側の変更",
    );
    fireEvent.click(screen.getByRole("button", { name: "上書きする" }));
    expect(props.onForceUpload).toHaveBeenCalledTimes(direction === "push" ? 1 : 0);
    expect(props.onForceDownload).toHaveBeenCalledTimes(direction === "pull" ? 1 : 0);
    expect(props.onUpload).not.toHaveBeenCalled();
    expect(props.onDownload).not.toHaveBeenCalled();
  });

  it("確認から戻ると転送しない", () => {
    const props = setup();
    fireEvent.click(screen.getByRole("button", { name: /Force Push/ }));
    fireEvent.click(screen.getByRole("button", { name: "戻る" }));
    expect(props.onForceUpload).not.toHaveBeenCalled();
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it.each([{ hasSaveFolder: false }, { isProcessing: true }])(
    "実行できない状態では強制操作を無効化する: %j",
    (overrides) => {
      setup(overrides);
      expect(screen.getByRole("button", { name: /Force Push/ })).toBeDisabled();
      expect(screen.getByRole("button", { name: /Force Pull/ })).toBeDisabled();
    },
  );

  it("クラウドにデータがない場合は Force Pull を無効化する", () => {
    setup({}, { status: "never_synced" });
    expect(screen.getByRole("button", { name: /Force Pull/ })).toBeDisabled();
    expect(screen.getByRole("button", { name: /Force Push/ })).not.toBeDisabled();
  });
});
