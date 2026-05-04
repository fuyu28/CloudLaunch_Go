/**
 * @fileoverview Electron IPC互換のWailsブリッジを提供する。
 */

import type { ApiResult } from "src/types/result";
import type {
  InputGameData,
  GameType,
  PlayRouteType,
  PlaySessionType,
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
import type { CloudDataItem, CloudDirectoryNode, CloudFileDetail } from "./hooks/useCloudData";
import type { CloudMetadata } from "src/types/cloud";

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
  SyncAllGames,
  SyncGame,
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
  DownloadSaveData,
  OpenFolder,
  OpenLogsDirectory,
  LoadCloudMetadata,
  SaveCredential,
  DeleteCloudGame,
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
  UpdateOfflineMode,
  UpdateScreenshotClientOnly,
  UpdateScreenshotHotkey,
  UpdateScreenshotHotkeyNotify,
  UpdateScreenshotJpegQuality,
  UpdateScreenshotLocalJpeg,
  UpdateScreenshotSyncEnabled,
  UpdateScreenshotUploadJpeg,
  UpdateUploadConcurrency,
  UpdateTransferRetryCount,
  UpdateGame,
  UpdateMemo,
  UploadFolder,
  PauseMonitoringSession,
  ResumeMonitoringSession,
  EndMonitoringSession,
  ComputeLocalSaveHash,
  GetCloudSaveHash,
  SaveCloudSaveHash,
  ValidateCredential,
  ValidateSavedCredential,
  FetchFromErogameScape,
} from "../wailsjs/go/app/App";
import { WindowMinimise, WindowToggleMaximise, Quit } from "../wailsjs/runtime/runtime";

function parseClearedAtInput(value?: string): Date | null {
  if (!value || !value.trim()) {
    return null;
  }
  const match = value.trim().match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (!match) {
    return null;
  }
  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  const parsed = new Date(year, month - 1, day);
  return Number.isNaN(parsed.getTime()) ? null : parsed;
}

function playRouteApp(): {
  CreatePlayRoute: (input: {
    GameID: string;
    Name: string;
    SortOrder: number;
  }) => Promise<{ success: boolean; data?: unknown; error?: { message?: string } }>;
  ListPlayRoutesByGame: (
    gameId: string,
  ) => Promise<{ success: boolean; data?: unknown; error?: { message?: string } }>;
  DeletePlayRoute: (
    routeId: string,
  ) => Promise<{ success: boolean; data?: unknown; error?: { message?: string } }>;
} {
  return (window as typeof window & { go: Record<string, Record<string, unknown>> }).go["app"][
    "App"
  ] as unknown as {
    CreatePlayRoute: (input: {
      GameID: string;
      Name: string;
      SortOrder: number;
    }) => Promise<{ success: boolean; data?: unknown; error?: { message?: string } }>;
    ListPlayRoutesByGame: (
      gameId: string,
    ) => Promise<{ success: boolean; data?: unknown; error?: { message?: string } }>;
    DeletePlayRoute: (
      routeId: string,
    ) => Promise<{ success: boolean; data?: unknown; error?: { message?: string } }>;
  };
}

export type WindowApi = {
  window: {
    minimize: () => Promise<void>;
    toggleMaximize: () => Promise<void>;
    close: () => Promise<void>;
    openFolder: (path: string) => Promise<void>;
  };
  settings: {
    updateAutoTracking: (enabled: boolean) => Promise<ApiResult<void>>;
    updateOfflineMode: (enabled: boolean) => Promise<ApiResult<void>>;
    updateUploadConcurrency: (value: number) => Promise<ApiResult<void>>;
    updateTransferRetryCount: (value: number) => Promise<ApiResult<void>>;
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
    createSession: (duration: number, gameId: string) => Promise<ApiResult<void>>;
    getPlaySessions: (gameId: string) => Promise<ApiResult<PlaySessionType[]>>;
    deletePlaySession: (sessionId: string) => Promise<ApiResult<void>>;
  };
  playRoute: {
    listByGame: (gameId: string) => Promise<ApiResult<PlayRouteType[]>>;
    create: (input: {
      gameId: string;
      name: string;
      sortOrder: number;
    }) => Promise<ApiResult<PlayRouteType>>;
    delete: (routeId: string) => Promise<ApiResult<void>>;
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
  cloudMetadata: {
    loadCloudMetadata: () => Promise<ApiResult<CloudMetadata>>;
  };
  saveData: {
    upload: {
      uploadSaveDataFolder: (localPath: string, remotePath: string) => Promise<ApiResult<void>>;
    };
    download: {
      downloadSaveData: (localPath: string, remotePath: string) => Promise<ApiResult<void>>;
      getCloudFileDetails: (
        gameId: string,
      ) => Promise<ApiResult<{ exists: boolean; totalSize: number; files: CloudFileDetail[] }>>;
    };
    hash: {
      computeLocalHash: (localPath: string) => Promise<ApiResult<string>>;
      getCloudHash: (
        gameId: string,
      ) => Promise<ApiResult<{ hash: string; updatedAt: Date } | null>>;
      saveCloudHash: (
        gameId: string,
        hash: string,
        updatedAt?: Date | string | null,
      ) => Promise<ApiResult<void>>;
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
  cloudSync: {
    syncAllGames: () => Promise<
      ApiResult<{
        uploadedGames: number;
        downloadedGames: number;
        uploadedSessions: number;
        downloadedSessions: number;
        uploadedImages: number;
        downloadedImages: number;
        skippedGames: number;
      }>
    >;
    syncGame: (gameId: string) => Promise<
      ApiResult<{
        uploadedGames: number;
        downloadedGames: number;
        uploadedSessions: number;
        downloadedSessions: number;
        uploadedImages: number;
        downloadedImages: number;
        skippedGames: number;
      }>
    >;
    deleteGame: (gameId: string) => Promise<ApiResult<void>>;
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

function normalizeApiDate(value: Date | string | number | null | undefined): Date {
  if (value instanceof Date) {
    return value;
  }
  if (typeof value === "string" && value.startsWith("0001-01-01T00:00:00")) {
    return new Date(Number.NaN);
  }
  return new Date(value ?? Number.NaN);
}

function normalizeCloudDirectoryNode(node: CloudDirectoryNode): CloudDirectoryNode {
  return {
    ...node,
    lastModified: normalizeApiDate(node.lastModified),
    children: node.children?.map(normalizeCloudDirectoryNode),
  };
}

function normalizeCloudDataItem(item: CloudDataItem): CloudDataItem {
  return {
    ...item,
    lastModified: normalizeApiDate(item.lastModified),
  };
}

function normalizeCloudFileDetail(file: CloudFileDetail): CloudFileDetail {
  return {
    ...file,
    lastModified: normalizeApiDate(file.lastModified),
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
      updateOfflineMode: async (enabled) => {
        const result = await UpdateOfflineMode(enabled);
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
      updateTransferRetryCount: async (value) => {
        const result = await UpdateTransferRetryCount(value);
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
        return result.success && result.data ? result.data : [];
      },
      getGameById: async (id) => {
        const result = await GetGameByID(id);
        if (!result.success) {
          return undefined;
        }
        return result.data ?? undefined;
      },
      createGame: async (game) => {
        const payload = {
          Title: game.title,
          Publisher: game.publisher,
          ImagePath: game.imagePath ?? null,
          ExePath: game.exePath,
          SaveFolderPath: game.saveFolderPath ?? null,
          ClearedAt: parseClearedAtInput(game.clearedAt),
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
          ImagePath: game.imagePath ?? null,
          ExePath: game.exePath,
          SaveFolderPath: game.saveFolderPath ?? null,
          ClearedAt: parseClearedAtInput(game.clearedAt),
        };
        const result = await UpdateGame(id, payload);
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
      createSession: async (duration, gameId) => {
        const payload = {
          GameID: gameId,
          PlayedAt: new Date(),
          Duration: duration,
        };
        const result = await CreateSession(payload);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      getPlaySessions: async (gameId) => {
        const result = await ListSessionsByGame(gameId);
        return result.success
          ? { success: true, data: (result.data ?? []) as PlaySessionType[] }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      deletePlaySession: async (sessionId) => {
        const result = await DeleteSession(sessionId);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
    },
    playRoute: {
      listByGame: async (gameId) => {
        const result = await playRouteApp().ListPlayRoutesByGame(gameId);
        return result.success
          ? { success: true, data: (result.data ?? []) as PlayRouteType[] }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      create: async (input) => {
        const result = await playRouteApp().CreatePlayRoute({
          GameID: input.gameId,
          Name: input.name,
          SortOrder: input.sortOrder,
        });
        return result.success
          ? { success: true, data: result.data as PlayRouteType }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      delete: async (routeId) => {
        const result = await playRouteApp().DeletePlayRoute(routeId);
        return result.success
          ? { success: true }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
    },
    memo: {
      getAllMemos: async () => {
        const result = await ListAllMemos();
        return result.success
          ? { success: true, data: (result.data ?? []) as MemoType[] }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      getMemoById: async (memoId) => {
        const result = await GetMemoByID(memoId);
        return result.success
          ? { success: true, data: result.data as MemoType }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      getMemosByGameId: async (gameId) => {
        const result = await ListMemosByGame(gameId);
        return result.success
          ? { success: true, data: (result.data ?? []) as MemoType[] }
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
          ? { success: true, data: (result.data ?? []) as CloudMemoInfo[] }
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
              data: ((result.data ?? []) as CloudDataItem[]).map(normalizeCloudDataItem),
            }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      getDirectoryTree: async () => {
        const result = await GetDirectoryTree();
        return result.success
          ? {
              success: true,
              data: ((result.data ?? []) as CloudDirectoryNode[]).map(normalizeCloudDirectoryNode),
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
              data: ((result.data ?? []) as CloudFileDetail[]).map(normalizeCloudFileDetail),
            }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
    },
    cloudMetadata: {
      loadCloudMetadata: async () => {
        const result = await LoadCloudMetadata("default");
        return result.success && result.data
          ? { success: true, data: result.data as CloudMetadata }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
    },
    saveData: {
      upload: {
        uploadSaveDataFolder: async (localPath, remotePath) => {
          const result = await UploadFolder("default", localPath, remotePath);
          return result.success
            ? { success: true }
            : { success: false, message: result.error?.message ?? "エラー" };
        },
      },
      download: {
        downloadSaveData: async (localPath, remotePath) => {
          const result = await DownloadSaveData(localPath, remotePath);
          return result.success
            ? { success: true }
            : { success: false, message: result.error?.message ?? "エラー" };
        },
        getCloudFileDetails: async (gameId) => {
          const result = await GetCloudFileDetailsByGame(gameId);
          return result.success
            ? {
                success: true,
                data: {
                  exists: Boolean(result.data?.exists),
                  totalSize: Number(result.data?.totalSize ?? 0),
                  files: ((result.data?.files ?? []) as CloudFileDetail[]).map(
                    normalizeCloudFileDetail,
                  ),
                },
              }
            : { success: false, message: result.error?.message ?? "エラー" };
        },
      },
      hash: {
        computeLocalHash: async (localPath) => {
          const result = await ComputeLocalSaveHash(localPath);
          return result.success
            ? { success: true, data: result.data as string }
            : { success: false, message: result.error?.message ?? "エラー" };
        },
        getCloudHash: async (gameId) => {
          const result = await GetCloudSaveHash(gameId);
          return result.success
            ? { success: true, data: result.data as { hash: string; updatedAt: Date } | null }
            : { success: false, message: result.error?.message ?? "エラー" };
        },
        saveCloudHash: async (gameId, hash, updatedAt) => {
          const normalizedUpdatedAt =
            updatedAt instanceof Date
              ? updatedAt.toISOString()
              : typeof updatedAt === "string"
                ? updatedAt
                : "";
          const result = await SaveCloudSaveHash(gameId, hash, normalizedUpdatedAt);
          return result.success
            ? { success: true }
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
    cloudSync: {
      syncAllGames: async () => {
        const result = await SyncAllGames();
        return result.success
          ? {
              success: true,
              data: result.data as {
                uploadedGames: number;
                downloadedGames: number;
                uploadedSessions: number;
                downloadedSessions: number;
                uploadedImages: number;
                downloadedImages: number;
                skippedGames: number;
              },
            }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      syncGame: async (gameId) => {
        const result = await SyncGame(gameId);
        return result.success
          ? {
              success: true,
              data: result.data as {
                uploadedGames: number;
                downloadedGames: number;
                uploadedSessions: number;
                downloadedSessions: number;
                uploadedImages: number;
                downloadedImages: number;
                skippedGames: number;
              },
            }
          : { success: false, message: result.error?.message ?? "エラー" };
      },
      deleteGame: async (gameId) => {
        try {
          const result = await DeleteCloudGame(gameId);
          return result.success
            ? { success: true }
            : { success: false, message: result.error?.message ?? "エラー" };
        } catch (error) {
          const message = error instanceof Error ? error.message : "削除に失敗しました";
          return { success: false, message };
        }
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
        void ReportError(payload).catch((error: unknown) => {
          console.error("ReportError failed", error, payload);
        });
      },
      reportLog: (payload) => {
        void ReportLog(payload).catch((error: unknown) => {
          console.error("ReportLog failed", error, payload);
        });
      },
    },
  };
};
