/**
 * @fileoverview wailsBridge の公開型定義。
 *
 * `WindowApi` およびクラウド同期関連の型をここで一元管理する。
 * 後方互換のため `src/wailsBridge` から re-export される。
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

export type WindowApi = {
  window: {
    minimize: () => Promise<void>;
    toggleMaximize: () => Promise<void>;
    close: () => Promise<void>;
    openFolder: (path: string) => Promise<void>;
    /** 実行プラットフォーム（"windows" / "darwin" / "linux"）を返す */
    getPlatform: () => Promise<string>;
  };
  settings: {
    updateAutoTracking: (enabled: boolean) => Promise<ApiResult<void>>;
    updateOfflineMode: (enabled: boolean) => Promise<ApiResult<void>>;
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
    /** 全ゲームの軽量サマリ（タイトル一覧のみ。ファイル数・サイズを含まない）を取得する */
    getCloudGameSummaries: () => Promise<ApiResult<CloudDataItem[]>>;
    getDirectoryTree: () => Promise<ApiResult<CloudDirectoryNode[]>>;
    /** 1ゲームの論理ディレクトリツリー（ファイル一覧・サイズ付き）を遅延取得する */
    getGameDirectoryNode: (gameId: string) => Promise<ApiResult<CloudDirectoryNode>>;
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
