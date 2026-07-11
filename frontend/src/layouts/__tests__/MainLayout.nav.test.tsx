import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route, Link } from "react-router-dom";
import { Provider as JotaiProvider } from "jotai";

import MainLayout from "../MainLayout";

function mockApi(): void {
  const settings = {
    updateOfflineMode: vi.fn().mockResolvedValue({ success: true }),
    updateAutoTracking: vi.fn().mockResolvedValue({ success: true }),
    updateUploadConcurrency: vi.fn().mockResolvedValue({ success: true }),
    updateScreenshotSyncEnabled: vi.fn().mockResolvedValue({ success: true }),
    updateScreenshotUploadJpeg: vi.fn().mockResolvedValue({ success: true }),
    updateScreenshotJpegQuality: vi.fn().mockResolvedValue({ success: true }),
    updateScreenshotClientOnly: vi.fn().mockResolvedValue({ success: true }),
    updateScreenshotLocalJpeg: vi.fn().mockResolvedValue({ success: true }),
    updateScreenshotHotkeyNotify: vi.fn().mockResolvedValue({ success: true }),
    updateScreenshotHotkey: vi.fn().mockResolvedValue({ success: true }),
    updateS3ForcePathStyle: vi.fn().mockResolvedValue({ success: true }),
    updateS3UseTLS: vi.fn().mockResolvedValue({ success: true }),
    updateLogLevel: vi.fn().mockResolvedValue({ success: true }),
  };
  (window as unknown as { api: unknown }).api = {
    settings,
    window: {
      getPlatform: vi.fn().mockResolvedValue("darwin"),
      minimize: vi.fn(),
      toggleMaximize: vi.fn(),
      close: vi.fn(),
    },
    processMonitor: { getMonitoringStatus: vi.fn().mockResolvedValue([]) },
    errorReport: { reportError: vi.fn() },
  };
}

function renderApp(initialPath = "/"): ReturnType<typeof render> {
  return render(
    <JotaiProvider>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route path="/" element={<MainLayout />}>
            <Route
              index
              element={
                <div>
                  <div>ホーム画面</div>
                  <Link to="/game/g1">ゲーム詳細へ</Link>
                </div>
              }
            />
            <Route path="/memo" element={<div>メモ画面</div>} />
            <Route path="/game/:id" element={<div>詳細画面</div>} />
            <Route path="/cloud" element={<div>クラウド画面</div>} />
            <Route path="/settings" element={<div>設定画面</div>} />
          </Route>
        </Routes>
      </MemoryRouter>
    </JotaiProvider>,
  );
}

describe("MainLayout navigation", () => {
  beforeEach(() => {
    mockApi();
  });

  it("navigates to memo via NavLink", async () => {
    const user = userEvent.setup();
    renderApp();

    expect(screen.getByText("ホーム画面")).toBeInTheDocument();
    await user.click(screen.getByRole("link", { name: /メモ/ }));
    expect(screen.getByText("メモ画面")).toBeInTheDocument();
  });

  it("navigates to game detail via Link", async () => {
    const user = userEvent.setup();
    renderApp();

    await user.click(screen.getByRole("link", { name: "ゲーム詳細へ" }));
    expect(screen.getByText("詳細画面")).toBeInTheDocument();
  });

  it("navigates to settings via NavLink", async () => {
    const user = userEvent.setup();
    renderApp();

    await user.click(screen.getByRole("link", { name: /設定/ }));
    expect(screen.getByText("設定画面")).toBeInTheDocument();
  });
});
