import { describe, expect, it } from "vitest";

import { decideAfterSessionSyncAction, decideLaunchSyncAction } from "../cloudSyncDecision";
import type { SyncStatusDetail } from "src/wailsBridge";

const detail = (status: SyncStatusDetail["status"], savesDiffer: boolean): SyncStatusDetail => ({
  status,
  savesDiffer,
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
