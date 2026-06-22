/**
 * @fileoverview WailsバックエンドAPIをフロントエンドに公開するブリッジ。
 *
 * Wailsが自動生成した `wailsjs/go/app/App` および `wailsjs/runtime/runtime` のバインディングを
 * ラップし、`WindowApi` 型として統一したインターフェースを提供する。
 * フロントエンドは必ず `window.api`（`src/types/window.d.ts` で宣言）経由でアクセスし、
 * 生成バインディングを直接 import しない。
 */

import type { ApiResult } from "src/types/result";
import type {
  InputGameData,
  GameType,
  PlaySessionType,
  PlayStatus,
  MonitoringGameStatus,
  GameImport,
} from "src/types/game";
import type { ErogameScapeSearchResult } from "src/types/erogamescape";
import type { SortOption, FilterOption, SortDirection } from "src/types/menu";
import type {
  MemoType,
  CreateMemoData,
  UpdateMemoData,
  MemoSyncResult,
  CloudMemoInfo,
} from "src/types/memo";
import type { Creds } from "src/types/creds";
import type { CloudDataItem, CloudDirectoryNode, CloudFileDetail } from "src/types/cloud";
import type {
  app as modelsApp,
  domain as modelsDomain,
  services as modelsServices,
  time as modelsTime,
} from "../wailsjs/go/models";

export type SyncStatus = "never_synced" | "in_sync" | "push_needed" | "pull_needed" | "conflict";

export type SyncStatusDetail = {
  status: SyncStatus;
  localMeta?: SyncMetaSnapshot;
  remoteMeta?: SyncMetaSnapshot;
};

export type SyncMetaSnapshot = {
  "game.json": string;
  "sessions.json": string;
  saves: string;
  deviceName: string;
  createdAt: Date;
};

export type SyncProgressEvent = {
  operation: "push" | "pull";
  current: number;
  total: number;
};

/**
 * Pull / ResolveConflict(リモート採用) の結果。
 * applied=false かつ untrackedDeletes が非空のときは「未追跡ファイルの削除確認待ち」で、
 * この時点ではローカルに変更が加わっていない。確認後 deleteUntracked=true で再実行する。
 */
export type PullResult = {
  applied: boolean;
  untrackedDeletes?: string[];
};

import {
  CreateGame,
  CreateMemo,
  CreateSession,
  CheckDirectoryExists,
  CheckFileExists,
  DeleteCloudData,
  DeleteFile,
  DeleteGame,
  DeleteMemo,
  DeleteSession,
  GetMemoByID,
  GetMemoRootDir,
  GetMemoFilePath,
  GetGameMemoDir,
  GetCloudMemos,
  DownloadMemoFromCloud,
  UploadMemoToCloud,
  SyncMemosFromCloud,
  SyncStatus,
  PushSync,
  PullSync,
  ResolveConflict,
  GetCloudFileDetails,
  GetCloudFileDetailsByGame,
  GetDirectoryTree,
  GetGameByID,
  GetMonitoringStatus,
  GetProcessSnapshot,
  LaunchGame,
  ListCloudData,
  ListAllMemos,
  ListGames,
  ListMemosByGame,
  ListSessionsByGame,
  LoadImageFromLocal,
  LoadCredential,
  OpenFolder,
  OpenLogsDirectory,
  DeleteGameFromCloud,
  LoadCloudMetadata,
  SaveCredential,
  SelectFile,
  SelectFolder,
  ExportGameData,
  CreateFullBackup,
  RestoreFullBackup,
  CaptureGameScreenshot,
  SearchErogameScape,
  ReportError,
  ReportLog,
  UpdateAutoTracking,
  UpdateScreenshotClientOnly,
  UpdateScreenshotHotkey,
  UpdateScreenshotHotkeyNotify,
  UpdateScreenshotJpegQuality,
  UpdateScreenshotLocalJpeg,
  UpdateScreenshotSyncEnabled,
  UpdateScreenshotUploadJpeg,
  UpdateUploadConcurrency,
  UpdateGame,
  UpdateMemo,
  UpdateSessionName,
  PauseMonitoringSession,
  ResumeMonitoringSession,
  EndMonitoringSession,
  ValidateCredential,
  ValidateSavedCredential,
  FetchFromErogameScape,
} from "../wailsjs/go/app/App";
import {
  WindowMinimise,
  WindowToggleMaximise,
  Quit,
  EventsOn,
  EventsOff,
} from "../wailsjs/runtime/runtime";

export type WindowApi = {
  window: {
    minimize: () => Promise<void>;
    toggleMaximize: () => Promise<void>;
    close: () => Promise<void>;
    openFolder: (path: string) => Promise<void>;
  };
  settings: {
    updateAutoTracking: (enabled: boolean) => Promise<ApiResult<void>>;
    updateUploadConcurrency: (value: number) => Promise<ApiResult<void>>;
    updateScreenshotSyncEnabled: (enabled: boolean) => Promise<ApiResult<void>>;
    updateScreenshotUploadJpeg: (enabled: boolean) => Promise<ApiResult<void>>;
    updateScreenshotJpegQuality: (value: number) => Promise<ApiResult<void>>;
    updateScreenshotClientOnly: (enabled: boolean) => Promise<ApiResult<void>>;
    updateScreenshotLocalJpeg: (enabled: boolean) => Promise<ApiResult<void>>;
    updateScreenshotHotkey: (combo: string) => Promise<ApiResult<void>>;
    updateScreenshotHotkeyNotify: (enabled: boolean) => Promise<ApiResult<void>>;
  };
  maintenance: {
    exportGameData: (
      outputDir: string,
    ) => Promise<ApiResult<{ jsonPath: string; csvPath: string }>>;
    createFullBackup: (outputDir: string) => Promise<ApiResult<string>>;
    restoreFullBackup: (backupPath: string) => Promise<ApiResult<void>>;
  };
  file: {
    selectFile: (filters?: { name: string; extensions: string[] }[]) => Promise<ApiResult<string>>;
    selectFolder: () => Promise<ApiResult<string>>;
    checkFileExists: (filePath: string) => Promise<boolean>;
    checkDirectoryExists: (dirPath: string) => Promise<boolean>;
    openLogsDirectory: () => Promise<ApiResult<string>>;
  };
  database: {
    listGames: (
      searchWord: string,
      filter: FilterOption,
      sort: SortOption,
      sortDirection?: SortDirection,
    ) => Promise<GameType[]>;
    getGameById: (id: string) => Promise<GameType | undefined>;
    createGame: (game: InputGameData) => Promise<ApiResult<void>>;
    updateGame: (id: string, game: InputGameData) => Promise<ApiResult<void>>;
    deleteGame: (id: string) => Promise<ApiResult<void>>;
    updatePlayStatus: (gameId: string, playStatus: PlayStatus) => Promise<ApiResult<GameType>>;
    createSession: (
      duration: number,
      gameId: string,
      sessionName?: string,
    ) => Promise<ApiResult<void>>;
    getPlaySessions: (gameId: string) => Promise<ApiResult<PlaySessionType[]>>;
    updateSessionName: (sessionId: string, sessionName: string) => Promise<ApiResult<void>>;
    deletePlaySession: (sessionId: string) => Promise<ApiResult<void>>;
  };
  memo: {
    getAllMemos: () => Promise<ApiResult<MemoType[]>>;
    getMemoById: (memoId: string) => Promise<ApiResult<MemoType>>;
    getMemosByGameId: (gameId: string) => Promise<ApiResult<MemoType[]>>;
    createMemo: (data: CreateMemoData) => Promise<ApiResult<void>>;
    updateMemo: (memoId: string, data: UpdateMemoData) => Promise<ApiResult<void>>;
    deleteMemo: (memoId: string) => Promise<ApiResult<void>>;
    getMemoRootDir: () => Promise<ApiResult<string>>;
    getMemoFilePath: (memoId: string) => Promise<ApiResult<string>>;
    getGameMemoDir: (gameId: string) => Promise<ApiResult<string>>;
    uploadMemoToCloud: (memoId: string) => Promise<ApiResult<void>>;
    downloadMemoFromCloud: (gameId: string, memoFileName: string) => Promise<ApiResult<string>>;
    getCloudMemos: () => Promise<ApiResult<CloudMemoInfo[]>>;
    syncMemosFromCloud: (gameId?: string) => Promise<ApiResult<MemoSyncResult>>;
  };
  credential: {
    upsertCredential: (creds: Creds) => Promise<ApiResult<void>>;
    getCredential: () => Promise<ApiResult<Creds>>;
    validateCredential: (creds: Creds) => Promise<ApiResult<void>>;
    validateSavedCredential: () => Promise<ApiResult<void>>;
  };
  cloudData: {
    listCloudData: () => Promise<ApiResult<CloudDataItem[]>>;
    getDirectoryTree: () => Promise<ApiResult<CloudDirectoryNode[]>>;
    deleteCloudData: (path: string) => Promise<ApiResult<void>>;
    deleteFile: (path: string) => Promise<ApiResult<void>>;
    getCloudFileDetails: (path: string) => Promise<ApiResult<CloudFileDetail[]>>;
  };
  saveData: {
    download: {
      getCloudFileDetails: (
        gameId: string,
      ) => Promise<ApiResult<{ exists: boolean; totalSize: number; files: CloudFileDetail[] }>>;
    };
  };
  loadImage: {
    loadImageFromLocal: (path: string) => Promise<ApiResult<string>>;
    loadImageFromWeb: (src: string) => Promise<ApiResult<string>>;
  };
  processMonitor: {
    getMonitoringStatus: () => Promise<MonitoringGameStatus[]>;
    pauseSession: (gameId: string) => Promise<ApiResult<void>>;
    resumeSession: (gameId: string) => Promise<ApiResult<void>>;
    endSession: (gameId: string) => Promise<ApiResult<void>>;
    getProcessSnapshot: () => Promise<{
      source: string;
      items: Array<{
        name: string;
        pid: number;
        cmd: string;
        normalizedName: string;
        normalizedCmd: string;
      }>;
    }>;
  };
  cloudMetadata: {
    loadCloudMetadata: () => Promise<
      ApiResult<{
        version: number;
        updatedAt: Date;
        games: import("src/types/cloud").CloudGameMetadata[];
      }>
    >;
  };
  cloudSync: {
    status: (gameId: string) => Promise<ApiResult<SyncStatusDetail>>;
    push: (gameId: string) => Promise<ApiResult<void>>;
    pull: (gameId: string, deleteUntracked?: boolean) => Promise<ApiResult<PullResult>>;
    resolveConflict: (
      gameId: string,
      useLocal: boolean,
      deleteUntracked?: boolean,
    ) => Promise<ApiResult<PullResult>>;
    deleteFromCloud: (gameId: string) => Promise<ApiResult<void>>;
    onProgress: (callback: (event: SyncProgressEvent) => void) => () => void;
  };
  game: {
    launchGame: (exePath: string) => Promise<ApiResult<void>>;
    captureWindow: (gameId: string) => Promise<ApiResult<string>>;
  };
  erogameScape: {
    fetchById: (id: string) => Promise<ApiResult<GameImport>>;
    searchByTitle: (
      query: string,
      pageUrl?: string,
    ) => Promise<ApiResult<ErogameScapeSearchResult>>;
  };
  errorReport: {
    reportError: (payload: {
      message: string;
      stack?: string;
      level?: string;
      context?: string;
      component?: string;
      function?: string;
      data?: unknown;
      timestamp?: string;
    }) => void;
    reportLog: (payload: {
      message: string;
      level?: string;
      context?: string;
      component?: string;
      function?: string;
      data?: unknown;
      timestamp?: string;
    }) => void;
  };
};

function normalizeApiDate(
  value: Date | string | number | null | undefined | modelsTime.Time,
): Date {
  if (value instanceof Date) {
    return value;
  }
  if (typeof value === "string" && value.startsWith("0001-01-01T00:00:00")) {
    return new Date(Number.NaN);
  }
  // time.Time arrives as ISO string at runtime; coerce to string for Date constructor
  const coerced = value as Date | string | number | null | undefined;
  return new Date(coerced ?? Number.NaN);
}

function normalizeCloudDirectoryNode(node: modelsApp.CloudDirectoryNode): CloudDirectoryNode {
  return {
    name: node.name,
    path: node.path,
    isDirectory: node.isDirectory,
    size: node.size,
    lastModified: normalizeApiDate(node.lastModified),
    children: node.children?.map(normalizeCloudDirectoryNode),
    objectKey: node.objectKey,
  };
}

function normalizeCloudDataItem(item: modelsApp.CloudDataItem): CloudDataItem {
  return {
    name: item.name,
    totalSize: item.totalSize,
    fileCount: item.fileCount,
    lastModified: normalizeApiDate(item.lastModified),
    remotePath: item.remotePath,
  };
}

function normalizeCloudFileDetail(file: modelsApp.CloudFileDetail): CloudFileDetail {
  return {
    name: file.name,
    size: file.size,
    lastModified: normalizeApiDate(file.lastModified),
    key: file.key,
    relativePath: file.relativePath,
  };
}

function toGameType(g: modelsDomain.Game): GameType {
  return {
    id: g.id,
    title: g.title,
    publisher: g.publisher,
    imagePath: g.imagePath,
    exePath: g.exePath,
    saveFolderPath: g.saveFolderPath,
    createdAt: normalizeApiDate(g.createdAt),
    localSaveHash: g.localSaveHash,
    localSaveHashUpdatedAt: g.localSaveHashUpdatedAt
      ? normalizeApiDate(g.localSaveHashUpdatedAt)
      : undefined,
    playStatus: g.playStatus as GameType["playStatus"],
    totalPlayTime: g.totalPlayTime,
    lastPlayed: g.lastPlayed ? normalizeApiDate(g.lastPlayed) : null,
    clearedAt: g.clearedAt ? normalizeApiDate(g.clearedAt) : null,
    currentRouteId: g.currentRouteId ?? null,
  };
}

function toPlaySessionType(s: modelsDomain.PlaySession): PlaySessionType {
  return {
    id: s.id,
    gameId: s.gameId,
    playedAt: normalizeApiDate(s.playedAt),
    duration: s.duration,
    sessionName: s.sessionName,
  };
}

function toMemoType(m: modelsDomain.Memo): MemoType {
  return {
    id: m.id,
    title: m.title,
    content: m.content,
    gameId: m.gameId,
    createdAt: normalizeApiDate(m.createdAt),
    updatedAt: normalizeApiDate(m.updatedAt),
  };
}

function toCloudMemoInfo(c: modelsServices.CloudMemoInfo): CloudMemoInfo {
  return {
    key: c.key,
    fileName: c.fileName,
    gameId: c.gameId,
    memoTitle: c.memoTitle,
    memoId: c.memoId,
    lastModified: normalizeApiDate(c.lastModified),
    size: c.size,
  };
}

export const createWailsBridge = (): WindowApi => {
  return {
    window: {
      minimize: async () => {
        await WindowMinimise();
      },
      toggleMaximize: async () => {
        await WindowToggleMaximise();
      },
      close: async () => {
        await Quit();
      },
      openFolder: async (path) => {
        await OpenFolder(path);
      },
    },
    settings: {
      updateAutoTracking: async (enabled) => {
        const result = await UpdateAutoTracking(enabled);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      updateUploadConcurrency: async (value) => {
        const result = await UpdateUploadConcurrency(value);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      updateScreenshotSyncEnabled: async (enabled) => {
        const result = await UpdateScreenshotSyncEnabled(enabled);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      updateScreenshotUploadJpeg: async (enabled) => {
        const result = await UpdateScreenshotUploadJpeg(enabled);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      updateScreenshotJpegQuality: async (value) => {
        const result = await UpdateScreenshotJpegQuality(value);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      updateScreenshotClientOnly: async (enabled) => {
        const result = await UpdateScreenshotClientOnly(enabled);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      updateScreenshotLocalJpeg: async (enabled) => {
        const result = await UpdateScreenshotLocalJpeg(enabled);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      updateScreenshotHotkey: async (combo) => {
        const result = await UpdateScreenshotHotkey(combo);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      updateScreenshotHotkeyNotify: async (enabled) => {
        const result = await UpdateScreenshotHotkeyNotify(enabled);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
    },
    maintenance: {
      exportGameData: async (outputDir) => {
        const result = await ExportGameData(outputDir);
        return result.success
          ? { success: true, data: result.data as { jsonPath: string; csvPath: string } }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      createFullBackup: async (outputDir) => {
        const result = await CreateFullBackup(outputDir);
        return result.success
          ? { success: true, data: result.data as string }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      restoreFullBackup: async (backupPath) => {
        const result = await RestoreFullBackup(backupPath);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
    },
    file: {
      selectFile: async (filters) => {
        const result = await SelectFile(filters ?? []);
        if (!result.success) {
          return {
            success: false,
            message: result.error?.message ?? "ファイルが選択されませんでした",
          };
        }
        return { success: true, data: result.data as string };
      },
      selectFolder: async () => {
        const result = await SelectFolder();
        if (!result.success) {
          return {
            success: false,
            message: result.error?.message ?? "フォルダが選択されませんでした",
          };
        }
        return { success: true, data: result.data as string };
      },
      checkFileExists: async (filePath) => {
        const result = await CheckFileExists(filePath);
        return result.success ? Boolean(result.data) : false;
      },
      checkDirectoryExists: async (dirPath) => {
        const result = await CheckDirectoryExists(dirPath);
        return result.success ? Boolean(result.data) : false;
      },
      openLogsDirectory: async () => {
        const result = await OpenLogsDirectory();
        return result.success
          ? { success: true, data: result.data as string }
          : {
              success: false,
              message: result.error?.message ?? "ログフォルダの表示に失敗しました",
            };
      },
    },
    database: {
      listGames: async (searchWord, filter, sort, sortDirection) => {
        const result = await ListGames(searchWord, filter, sort, sortDirection ?? "asc");
        return result.success && result.data ? result.data.map(toGameType) : [];
      },
      getGameById: async (id) => {
        const result = await GetGameByID(id);
        if (!result.success) {
          return undefined;
        }
        return result.data ? toGameType(result.data) : undefined;
      },
      createGame: async (game) => {
        const payload = {
          Title: game.title,
          Publisher: game.publisher,
          ImagePath: game.imagePath ?? undefined,
          ExePath: game.exePath,
          SaveFolderPath: game.saveFolderPath ?? undefined,
        };
        const result = await CreateGame(payload);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      updateGame: async (id, game) => {
        const payload = {
          Title: game.title,
          Publisher: game.publisher,
          ImagePath: game.imagePath ?? undefined,
          ExePath: game.exePath,
          SaveFolderPath: game.saveFolderPath ?? undefined,
          // 空文字を渡すとバックエンド側 (UpdateGame) は playStatus を上書きしない。
          // 一般的なゲーム編集では playStatus を変更しないため空文字を維持する。
          PlayStatus: "" as string,
          ClearedAt: undefined,
          CurrentRouteID: undefined,
        };
        const result = await UpdateGame(id, payload as unknown as modelsServices.GameUpdateInput);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      deleteGame: async (id) => {
        const result = await DeleteGame(id);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      updatePlayStatus: async (gameId, playStatus) => {
        const current = await GetGameByID(gameId);
        if (!current.success || !current.data) {
          return { success: false, message: current.error?.message ?? "ゲーム取得に失敗しました" };
        }
        const game = toGameType(current.data);
        const clearedAt = playStatus === "played" ? new Date() : null;
        const updatePayload = {
          Title: game.title,
          Publisher: game.publisher,
          ImagePath: game.imagePath ?? undefined,
          ExePath: game.exePath,
          SaveFolderPath: game.saveFolderPath ?? undefined,
          PlayStatus: playStatus,
          ClearedAt: clearedAt !== null ? (clearedAt as unknown as modelsTime.Time) : undefined,
          CurrentRouteID: game.currentRouteId ?? undefined,
        };
        const result = await UpdateGame(
          gameId,
          updatePayload as unknown as modelsServices.GameUpdateInput,
        );
        if (!result.success) {
          return { success: false, message: result.error?.message ?? "エラー" };
        }
        const updated = await GetGameByID(gameId);
        if (!updated.success) {
          return { success: false, message: updated.error?.message ?? "エラー" };
        }
        return {
          success: true,
          data: updated.data ? toGameType(updated.data) : (undefined as unknown as GameType),
        };
      },
      createSession: async (duration, gameId, sessionName) => {
        const payload = {
          GameID: gameId,
          PlayedAt: new Date() as unknown as modelsTime.Time,
          Duration: duration,
          SessionName: sessionName ?? undefined,
          RouteID: undefined,
        };
        const result = await CreateSession(payload as unknown as modelsServices.SessionInput);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      getPlaySessions: async (gameId) => {
        const result = await ListSessionsByGame(gameId);
        return result.success
          ? { success: true, data: (result.data ?? []).map(toPlaySessionType) }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      updateSessionName: async (sessionId, sessionName) => {
        const result = await UpdateSessionName(sessionId, sessionName);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      deletePlaySession: async (sessionId) => {
        const result = await DeleteSession(sessionId);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
    },
    memo: {
      getAllMemos: async () => {
        const result = await ListAllMemos();
        return result.success
          ? { success: true, data: (result.data ?? []).map(toMemoType) }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      getMemoById: async (memoId) => {
        const result = await GetMemoByID(memoId);
        return result.success
          ? {
              success: true,
              data: result.data ? toMemoType(result.data) : (undefined as unknown as MemoType),
            }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      getMemosByGameId: async (gameId) => {
        const result = await ListMemosByGame(gameId);
        return result.success
          ? { success: true, data: (result.data ?? []).map(toMemoType) }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      createMemo: async (data) => {
        const result = await CreateMemo({
          Title: data.title,
          Content: data.content,
          GameID: data.gameId,
        });
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      updateMemo: async (memoId, data) => {
        const result = await UpdateMemo(memoId, { Title: data.title, Content: data.content });
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      deleteMemo: async (memoId) => {
        const result = await DeleteMemo(memoId);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      getMemoRootDir: async () => {
        const result = await GetMemoRootDir();
        return result.success
          ? { success: true, data: result.data as string }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      getMemoFilePath: async (memoId) => {
        const result = await GetMemoFilePath(memoId);
        return result.success
          ? { success: true, data: result.data as string }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      getGameMemoDir: async (gameId) => {
        const result = await GetGameMemoDir(gameId);
        return result.success
          ? { success: true, data: result.data as string }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      uploadMemoToCloud: async (memoId) => {
        const result = await UploadMemoToCloud(memoId);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      downloadMemoFromCloud: async (gameId, memoFileName) => {
        const result = await DownloadMemoFromCloud(gameId, memoFileName);
        return result.success
          ? { success: true, data: result.data as string }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      getCloudMemos: async () => {
        const result = await GetCloudMemos();
        return result.success
          ? { success: true, data: (result.data ?? []).map(toCloudMemoInfo) }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      syncMemosFromCloud: async (gameId) => {
        const result = await SyncMemosFromCloud(gameId ?? "");
        return result.success
          ? { success: true, data: result.data as MemoSyncResult }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
    },
    credential: {
      upsertCredential: async (creds) => {
        const result = await SaveCredential("default", {
          BucketName: creds.bucketName,
          Region: creds.region,
          Endpoint: creds.endpoint,
          AccessKeyID: creds.accessKeyId,
          SecretAccessKey: creds.secretAccessKey,
        });
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      getCredential: async () => {
        const result = await LoadCredential("default");
        if (!result.success || !result.data) {
          return { success: false, message: result.error?.message ?? "認証情報がありません" };
        }
        return {
          success: true,
          data: {
            accessKeyId: result.data.AccessKeyID,
            secretAccessKey: "",
            bucketName: result.data.BucketName ?? "",
            region: result.data.Region ?? "",
            endpoint: result.data.Endpoint ?? "",
          },
        };
      },
      validateCredential: async (creds) => {
        const result = await ValidateCredential({
          bucketName: creds.bucketName,
          region: creds.region,
          endpoint: creds.endpoint,
          accessKeyId: creds.accessKeyId,
          secretAccessKey: creds.secretAccessKey,
        });
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      validateSavedCredential: async () => {
        const result = await ValidateSavedCredential("default");
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
    },
    cloudData: {
      listCloudData: async () => {
        const result = await ListCloudData();
        return result.success
          ? {
              success: true,
              data: (result.data ?? []).map(normalizeCloudDataItem),
            }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      getDirectoryTree: async () => {
        const result = await GetDirectoryTree();
        return result.success
          ? {
              success: true,
              data: (result.data ?? []).map(normalizeCloudDirectoryNode),
            }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      deleteCloudData: async (path) => {
        const result = await DeleteCloudData(path);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      deleteFile: async (path) => {
        const result = await DeleteFile(path);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      getCloudFileDetails: async (path) => {
        const result = await GetCloudFileDetails(path);
        return result.success
          ? {
              success: true,
              data: (result.data ?? []).map(normalizeCloudFileDetail),
            }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
    },
    saveData: {
      download: {
        getCloudFileDetails: async (gameId) => {
          const result = await GetCloudFileDetailsByGame(gameId);
          return result.success
            ? {
                success: true,
                data: {
                  exists: Boolean(result.data?.exists),
                  totalSize: Number(result.data?.totalSize ?? 0),
                  files: (result.data?.files ?? []).map(normalizeCloudFileDetail),
                },
              }
            : { success: false, message: result.error?.message ?? "エラー" };
        },
      },
    },
    loadImage: {
      loadImageFromLocal: async (path) => {
        const result = await LoadImageFromLocal(path);
        return result.success
          ? { success: true, data: result.data as string }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      loadImageFromWeb: async (src) => ({ success: true, data: src }),
    },
    processMonitor: {
      getMonitoringStatus: async () => {
        const result = await GetMonitoringStatus();
        if (!result.success) {
          return [];
        }
        if (Array.isArray(result.data)) {
          return result.data as MonitoringGameStatus[];
        }
        return [];
      },
      getProcessSnapshot: async () => {
        const result = await GetProcessSnapshot();
        if (!result.success || !result.data) {
          return { source: "error", items: [] };
        }
        return result.data as {
          source: string;
          items: Array<{
            name: string;
            pid: number;
            cmd: string;
            normalizedName: string;
            normalizedCmd: string;
          }>;
        };
      },
      pauseSession: async (gameId) => {
        const result = await PauseMonitoringSession(gameId);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      resumeSession: async (gameId) => {
        const result = await ResumeMonitoringSession(gameId);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      endSession: async (gameId) => {
        const result = await EndMonitoringSession(gameId);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
    },
    cloudMetadata: {
      loadCloudMetadata: async () => {
        const result = await LoadCloudMetadata();
        if (!result.success || !result.data) {
          return { success: false, message: result.error?.message ?? "エラー" };
        }
        return {
          success: true,
          data: {
            version: result.data.version,
            updatedAt: normalizeApiDate(result.data.updatedAt),
            games: result.data.games as unknown as import("src/types/cloud").CloudGameMetadata[],
          },
        };
      },
    },
    cloudSync: {
      status: async (gameId) => {
        const result = await SyncStatus(gameId);
        if (!result.success) {
          return { success: false, message: result.error?.message ?? "エラー" };
        }
        const raw = result.data as {
          status: SyncStatus;
          localMeta?: {
            "game.json": string;
            "sessions.json": string;
            saves: string;
            deviceName: string;
            createdAt: string;
          };
          remoteMeta?: {
            "game.json": string;
            "sessions.json": string;
            saves: string;
            deviceName: string;
            createdAt: string;
          };
        };
        const normalizeMeta = (m?: typeof raw.localMeta): SyncMetaSnapshot | undefined =>
          m ? { ...m, createdAt: new Date(m.createdAt) } : undefined;
        return {
          success: true,
          data: {
            status: raw.status,
            localMeta: normalizeMeta(raw.localMeta),
            remoteMeta: normalizeMeta(raw.remoteMeta),
          },
        };
      },
      push: async (gameId) => {
        const result = await PushSync(gameId);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      pull: async (gameId, deleteUntracked = false) => {
        const result = await PullSync(gameId, deleteUntracked);
        return result.success
          ? { success: true, data: result.data as PullResult }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      resolveConflict: async (gameId, useLocal, deleteUntracked = false) => {
        const result = await ResolveConflict(gameId, useLocal, deleteUntracked);
        return result.success
          ? { success: true, data: result.data as PullResult }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      deleteFromCloud: async (gameId) => {
        const result = await DeleteGameFromCloud(gameId);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      onProgress: (callback) => {
        EventsOn("sync:progress", callback);
        return () => EventsOff("sync:progress");
      },
    },
    game: {
      launchGame: async (exePath) => {
        const result = await LaunchGame(exePath);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      captureWindow: async (gameId) => {
        try {
          const result = await CaptureGameScreenshot(gameId);
          return result.success
            ? { success: true, data: result.data as string }
            : { success: false, message: result.error?.message ?? "エラー" };
        } catch (error) {
          const message =
            error instanceof Error ? error.message : "スクリーンショットに失敗しました";
          return { success: false, message };
        }
      },
    },
    erogameScape: {
      fetchById: async (id) => {
        const trimmed = id.trim();
        if (!trimmed) {
          return { success: false, message: "批評空間IDを入力してください" };
        }
        const url = `https://erogamescape.dyndns.org/~ap2/ero/toukei_kaiseki/game.php?game=${encodeURIComponent(
          trimmed,
        )}`;
        try {
          const result = await FetchFromErogameScape(url);
          return { success: true, data: result as GameImport };
        } catch (error) {
          let message = "批評空間からの取得に失敗しました";
          if (error instanceof Error) {
            message = error.message;
          } else if (typeof error === "string") {
            message = error;
          } else if (error) {
            message = JSON.stringify(error);
          }
          return { success: false, message };
        }
      },
      searchByTitle: async (query, pageUrl) => {
        const trimmed = query.trim();
        if (!trimmed && !pageUrl) {
          return { success: false, message: "検索ワードを入力してください" };
        }
        try {
          const result = await SearchErogameScape(trimmed, pageUrl ?? "");
          return { success: true, data: result as ErogameScapeSearchResult };
        } catch (error) {
          let message = "批評空間の検索に失敗しました";
          if (error instanceof Error) {
            message = error.message;
          } else if (typeof error === "string") {
            message = error;
          } else if (error) {
            message = JSON.stringify(error);
          }
          return { success: false, message };
        }
      },
    },
    errorReport: {
      reportError: (payload) => {
        void ReportError({
          level: payload.level ?? "error",
          message: payload.message,
          stack: payload.stack ?? "",
          context: payload.context ?? "",
          component: payload.component ?? "",
          function: payload.function ?? "",
          data: payload.data ?? null,
          timestamp: payload.timestamp ?? new Date().toISOString(),
        }).catch((error: unknown) => {
          console.error("ReportError failed", error, payload);
        });
      },
      reportLog: (payload) => {
        void ReportLog({
          level: payload.level ?? "info",
          message: payload.message,
          component: payload.component ?? "",
          function: payload.function ?? "",
          context: payload.context ?? "",
          data: payload.data ?? null,
          timestamp: payload.timestamp ?? new Date().toISOString(),
        }).catch((error: unknown) => {
          console.error("ReportLog failed", error, payload);
        });
      },
    },
  };
};
