import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

import SyncStatusModal from "./SyncStatusModal";

afterEach(cleanup);

function setup(overrides = {}) {
  const props = {
    isOpen: true,
    onClose: vi.fn(),
    gameTitle: "Test game",
    status: "push_needed" as const,
    hasSaveFolder: true,
    isProcessing: false,
    onUpload: vi.fn(),
    onDownload: vi.fn(),
    onForceUpload: vi.fn(),
    onForceDownload: vi.fn(),
    ...overrides,
  };
  render(<SyncStatusModal {...props} />);
  fireEvent.click(screen.getByText("強制同期"));
  return props;
}

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
    setup({ status: "never_synced" });
    expect(screen.getByRole("button", { name: /Force Pull/ })).toBeDisabled();
    expect(screen.getByRole("button", { name: /Force Push/ })).not.toBeDisabled();
  });
});
