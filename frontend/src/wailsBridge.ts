/**
 * @fileoverview WailsバックエンドAPIをフロントエンドに公開するブリッジ。
 *
 * Wailsが自動生成した `wailsjs/go/app/App` および `wailsjs/runtime/runtime` のバインディングを
 * ラップし、`WindowApi` 型として統一したインターフェースを提供する。
 */

// ---- 後方互換 re-export ------------------------------------------------
// src/wailsBridge から import している既存コードへの互換性を維持する
export type {
  WindowApi,
  SyncStatus,
  SyncStatusDetail,
  SyncMetaSnapshot,
  SyncProgressEvent,
  PullResult,
} from "./bridge/types";

// ---- ドメインブリッジ合成 -----------------------------------------------
import { createWindowBridge } from "./bridge/window";
import { createBrowserBridge } from "./bridge/browser";
import { createSettingsBridge } from "./bridge/settings";
import { createMaintenanceBridge } from "./bridge/maintenance";
import { createFileBridge } from "./bridge/file";
import { createDatabaseBridge } from "./bridge/database";
import { createMemoBridge } from "./bridge/memo";
import { createCredentialBridge } from "./bridge/credential";
import { createCloudDataBridge } from "./bridge/cloudData";
import { createSaveDataBridge } from "./bridge/saveData";
import { createLoadImageBridge } from "./bridge/loadImage";
import { createProcessMonitorBridge } from "./bridge/processMonitor";
import { createCloudMetadataBridge } from "./bridge/cloudMetadata";
import { createCloudSyncBridge } from "./bridge/cloudSync";
import { createGameBridge } from "./bridge/game";
import { createErogameScapeBridge } from "./bridge/erogameScape";
import { createErrorReportBridge } from "./bridge/errorReport";
import type { WindowApi } from "./bridge/types";

export const createWailsBridge = (): WindowApi => ({
  window: createWindowBridge(),
  browser: createBrowserBridge(),
  settings: createSettingsBridge(),
  maintenance: createMaintenanceBridge(),
  file: createFileBridge(),
  database: createDatabaseBridge(),
  memo: createMemoBridge(),
  credential: createCredentialBridge(),
  cloudData: createCloudDataBridge(),
  saveData: createSaveDataBridge(),
  loadImage: createLoadImageBridge(),
  processMonitor: createProcessMonitorBridge(),
  cloudMetadata: createCloudMetadataBridge(),
  cloudSync: createCloudSyncBridge(),
  game: createGameBridge(),
  erogameScape: createErogameScapeBridge(),
  errorReport: createErrorReportBridge(),
});
