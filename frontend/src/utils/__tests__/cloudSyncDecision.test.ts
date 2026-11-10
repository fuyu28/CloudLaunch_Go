import { describe, expect, it } from "vitest";

import {
  decideAfterSessionSyncAction,
  decideLaunchSyncAction,
  decideManualTransfer,
} from "../cloudSyncDecision";
import type { SyncStatusDetail } from "src/wailsBridge";

const detail = (
  status: SyncStatusDetail["status"],
  savesDiffer: boolean,
  hasLocalBaseline = true,
): SyncStatusDetail => ({
  status,
  savesDiffer,
  hasLocalBaseline,
});

describe("decideLaunchSyncAction", () => {
  it.each([
    ["pull_needed", true, "confirmDownload"],
    ["pull_needed", false, "syncMetadata"],
    ["push_needed", true, "launch"],
    ["push_needed", false, "syncMetadata"],
    ["conflict", true, "resolveConflict"],
    ["conflict", false, "resolveConflict"],
    ["in_sync", false, "launch"],
    ["never_synced", false, "launch"],
  ] as const)("status=%s savesDiffer=%s → %s", (status, savesDiffer, expected) => {
    expect(decideLaunchSyncAction(detail(status, savesDiffer))).toBe(expected);
  });
});

describe("decideAfterSessionSyncAction", () => {
  it.each([
    ["push_needed", true, "confirmUpload"],
    ["push_needed", false, "none"],
    ["never_synced", false, "confirmUpload"],
    ["conflict", true, "resolveConflict"],
    ["conflict", false, "none"],
    ["pull_needed", true, "none"],
    ["in_sync", false, "none"],
  ] as const)("status=%s savesDiffer=%s → %s", (status, savesDiffer, expected) => {
    expect(decideAfterSessionSyncAction(detail(status, savesDiffer))).toBe(expected);
  });
});

describe("decideManualTransfer", () => {
  it.each([
    ["upload", "push_needed", true, "run"],
    ["upload", "never_synced", false, "run"],
    ["upload", "in_sync", true, "showStatus"],
    ["upload", "pull_needed", true, "showStatus"],
    ["upload", "conflict", true, "resolveConflict"],
    ["download", "pull_needed", true, "run"],
    ["download", "pull_needed", false, "showStatus"],
    ["download", "push_needed", true, "showStatus"],
    ["download", "in_sync", true, "showStatus"],
    ["download", "conflict", true, "resolveConflict"],
  ] as const)(
    "%s で status=%s hasLocalBaseline=%s → %s",
    (direction, status, hasLocalBaseline, expected) => {
      expect(decideManualTransfer(detail(status, true, hasLocalBaseline), direction)).toBe(
        expected,
      );
    },
  );
});
