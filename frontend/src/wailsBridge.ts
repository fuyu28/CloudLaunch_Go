/**
 * @fileoverview Electron IPC互換のWailsブリッジを提供する。
 */

import type { ApiResult } from "src/types/result"
import type { InputGameData, GameType, PlaySessionType, PlayStatus } from "src/types/game"
import type { SortOption, FilterOption, SortDirection } from "src/types/menu"
import type { Chapter, ChapterStats } from "src/types/chapter"
import type { MemoType, CreateMemoData, UpdateMemoData, MemoSyncResult, CloudMemoInfo } from "src/types/memo"
import type { Creds } from "src/types/creds"
import type { CloudDataItem, CloudDirectoryNode, CloudFileDetail } from "./hooks/useCloudData"

import {
  CreateChapter,
  CreateGame,
  CreateMemo,
  CreateSession,
  CreateUpload,
  CheckDirectoryExists,
  CheckFileExists,
  DeleteCloudData,
  DeleteChapter,
  DeleteCredential,
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
  GetCloudFileDetails,
  GetCloudFileDetailsByGame,
  GetDirectoryTree,
  GetGameByID,
  GetMonitoringStatus,
  GetChapterStats,
  LaunchGame,
  ListChaptersByGame,
  ListCloudData,
  ListAllMemos,
  ListGames,
  ListMemosByGame,
  ListSessionsByGame,
  ListUploadsByGame,
  LoadImageFromLocal,
  LoadCloudMetadata,
  LoadCredential,
  DownloadSaveData,
  OpenFolder,
  OpenLogsDirectory,
  SaveCloudMetadata,
  SaveCredential,
  SelectFile,
  SelectFolder,
  SetCurrentChapter,
  UpdateAutoTracking,
  UpdateChapter,
  UpdateChapterOrders,
  UpdateGame,
  UpdateMemo,
  UpdateSessionChapter,
  UpdateSessionName,
  UploadFolder,
  ValidateCredential
} from "../wailsjs/go/app/App"
import { WindowMinimise, WindowToggleMaximise, Quit } from "../wailsjs/runtime/runtime"

export type WindowApi = {
  window: {
    minimize: () => Promise<void>
    toggleMaximize: () => Promise<void>
    close: () => Promise<void>
    openFolder: (path: string) => Promise<void>
  }
  settings: {
    updateAutoTracking: (enabled: boolean) => Promise<ApiResult<void>>
  }
  file: {
    selectFile: (filters?: { name: string; extensions: string[] }[]) => Promise<ApiResult<string>>
    selectFolder: () => Promise<ApiResult<string>>
    checkFileExists: (filePath: string) => Promise<boolean>
    checkDirectoryExists: (dirPath: string) => Promise<boolean>
    openLogsDirectory: () => Promise<ApiResult<string>>
  }
  database: {
    listGames: (
      searchWord: string,
      filter: FilterOption,
      sort: SortOption,
      sortDirection?: SortDirection
    ) => Promise<GameType[]>
    getGameById: (id: string) => Promise<GameType | undefined>
    createGame: (game: InputGameData) => Promise<ApiResult<void>>
    updateGame: (id: string, game: InputGameData) => Promise<ApiResult<void>>
    deleteGame: (id: string) => Promise<ApiResult<void>>
    updatePlayStatus: (gameId: string, playStatus: PlayStatus, clearedAt?: Date) => Promise<ApiResult<GameType>>
    createSession: (duration: number, gameId: string, sessionName?: string) => Promise<ApiResult<void>>
    getPlaySessions: (gameId: string) => Promise<ApiResult<PlaySessionType[]>>
    updateSessionChapter: (sessionId: string, chapterId: string | null) => Promise<ApiResult<void>>
    updateSessionName: (sessionId: string, sessionName: string) => Promise<ApiResult<void>>
    deletePlaySession: (sessionId: string) => Promise<ApiResult<void>>
  }
  chapter: {
    getChapters: (gameId: string) => Promise<ApiResult<Chapter[]>>
    createChapter: (input: { name: string; gameId: string }) => Promise<ApiResult<Chapter>>
    updateChapter: (chapterId: string, input: { name: string; order: number }) => Promise<ApiResult<Chapter>>
    deleteChapter: (chapterId: string) => Promise<ApiResult<void>>
    updateChapterOrders: (gameId: string, chapterOrders: { id: string; order: number }[]) => Promise<ApiResult<void>>
    getChapterStats: (gameId: string) => Promise<ApiResult<ChapterStats[]>>
    setCurrentChapter: (gameId: string, chapterId: string) => Promise<ApiResult<void>>
  }
  memo: {
    getAllMemos: () => Promise<ApiResult<MemoType[]>>
    getMemoById: (memoId: string) => Promise<ApiResult<MemoType>>
    getMemosByGameId: (gameId: string) => Promise<ApiResult<MemoType[]>>
    createMemo: (data: CreateMemoData) => Promise<ApiResult<void>>
    updateMemo: (memoId: string, data: UpdateMemoData) => Promise<ApiResult<void>>
    deleteMemo: (memoId: string) => Promise<ApiResult<void>>
    getMemoRootDir: () => Promise<ApiResult<string>>
    getMemoFilePath: (memoId: string) => Promise<ApiResult<string>>
    getGameMemoDir: (gameId: string) => Promise<ApiResult<string>>
    uploadMemoToCloud: (memoId: string) => Promise<ApiResult<void>>
    downloadMemoFromCloud: (gameTitle: string, memoFileName: string) => Promise<ApiResult<string>>
    getCloudMemos: () => Promise<ApiResult<CloudMemoInfo[]>>
    syncMemosFromCloud: (gameId?: string) => Promise<ApiResult<MemoSyncResult>>
  }
  credential: {
    upsertCredential: (creds: Creds) => Promise<ApiResult<void>>
    getCredential: () => Promise<ApiResult<Creds>>
    validateCredential: (creds: Creds) => Promise<ApiResult<void>>
  }
  cloudData: {
    listCloudData: () => Promise<ApiResult<CloudDataItem[]>>
    getDirectoryTree: () => Promise<ApiResult<CloudDirectoryNode[]>>
    deleteCloudData: (path: string) => Promise<ApiResult<void>>
    deleteFile: (path: string) => Promise<ApiResult<void>>
    getCloudFileDetails: (path: string) => Promise<ApiResult<CloudFileDetail[]>>
  }
  saveData: {
    upload: {
      uploadSaveDataFolder: (localPath: string, remotePath: string) => Promise<ApiResult<void>>
    }
    download: {
      downloadSaveData: (localPath: string, remotePath: string) => Promise<ApiResult<void>>
      getCloudFileDetails: (gameId: string) => Promise<ApiResult<{ exists: boolean; totalSize: number; files: CloudFileDetail[] }>>
    }
  }
  loadImage: {
    loadImageFromLocal: (path: string) => Promise<ApiResult<string>>
    loadImageFromWeb: (src: string) => Promise<ApiResult<string>>
  }
  processMonitor: {
    getMonitoringStatus: () => Promise<ApiResult<{ isMonitoring: boolean }>>
  }
  game: {
    launchGame: (exePath: string) => Promise<ApiResult<void>>
  }
  errorReport: {
    reportError: (payload: { message: string; stack?: string; level?: string; context?: string }) => void
    reportLog: (payload: { message: string; level?: string; context?: string }) => void
  }
}

export const createWailsBridge = (): WindowApi => {
  return {
    window: {
      minimize: async () => {
        await WindowMinimise()
      },
      toggleMaximize: async () => {
        await WindowToggleMaximise()
      },
      close: async () => {
        await Quit()
      },
      openFolder: async (path) => {
        await OpenFolder(path)
      }
    },
    settings: {
      updateAutoTracking: async (enabled) => {
        const result = await UpdateAutoTracking(enabled)
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      }
    },
    file: {
      selectFile: async (filters) => {
        const result = await SelectFile(filters ?? [])
        if (!result.success) {
          return { success: false, message: result.error?.message ?? "ファイルが選択されませんでした" }
        }
        return { success: true, data: result.data as string }
      },
      selectFolder: async () => {
        const result = await SelectFolder()
        if (!result.success) {
          return { success: false, message: result.error?.message ?? "フォルダが選択されませんでした" }
        }
        return { success: true, data: result.data as string }
      },
      checkFileExists: async (filePath) => {
        const result = await CheckFileExists(filePath)
        return result.success ? Boolean(result.data) : false
      },
      checkDirectoryExists: async (dirPath) => {
        const result = await CheckDirectoryExists(dirPath)
        return result.success ? Boolean(result.data) : false
      },
      openLogsDirectory: async () => {
        const result = await OpenLogsDirectory()
        return result.success
          ? { success: true, data: result.data as string }
          : { success: false, message: result.error?.message ?? "ログフォルダの表示に失敗しました" }
      }
    },
    database: {
      listGames: async (searchWord, filter, sort, sortDirection) => {
        const result = await ListGames(searchWord, filter, sort, sortDirection ?? "asc")
        return result.success && result.data ? result.data : []
      },
      getGameById: async (id) => {
        const result = await GetGameByID(id)
        if (!result.success) {
          return undefined
        }
        return result.data ?? undefined
      },
      createGame: async (game) => {
        const payload = {
          Title: game.title,
          Publisher: game.publisher,
          ImagePath: game.imagePath ?? null,
          ExePath: game.exePath,
          SaveFolderPath: game.saveFolderPath ?? null
        }
        const result = await CreateGame(payload)
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      },
      updateGame: async (id, game) => {
        const payload = {
          Title: game.title,
          Publisher: game.publisher,
          ImagePath: game.imagePath ?? null,
          ExePath: game.exePath,
          SaveFolderPath: game.saveFolderPath ?? null,
          PlayStatus: game.playStatus ?? "unplayed",
          ClearedAt: null,
          CurrentChapter: null
        }
        const result = await UpdateGame(id, payload)
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      },
      deleteGame: async (id) => {
        const result = await DeleteGame(id)
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      },
      updatePlayStatus: async (gameId, playStatus, clearedAt) => {
        const current = await GetGameByID(gameId)
        if (!current.success || !current.data) {
          return { success: false, message: current.error?.message ?? "ゲーム取得に失敗しました" }
        }
        const game = current.data as GameType
        const updatePayload = {
          Title: game.title,
          Publisher: game.publisher,
          ImagePath: game.imagePath ?? null,
          ExePath: game.exePath,
          SaveFolderPath: game.saveFolderPath ?? null,
          PlayStatus: playStatus,
          ClearedAt: clearedAt ?? null,
          CurrentChapter: game.currentChapter ?? null
        }
        const result = await UpdateGame(gameId, updatePayload)
        if (!result.success) {
          return { success: false, message: result.error?.message ?? "エラー" }
        }
        const updated = await GetGameByID(gameId)
        if (!updated.success) {
          return { success: false, message: updated.error?.message ?? "エラー" }
        }
        return { success: true, data: updated.data as GameType }
      },
      createSession: async (duration, gameId, sessionName) => {
        const payload = {
          GameID: gameId,
          PlayedAt: new Date(),
          Duration: duration,
          SessionName: sessionName ?? null,
          ChapterID: null,
          UploadID: null
        }
        const result = await CreateSession(payload)
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      },
      getPlaySessions: async (gameId) => {
        const result = await ListSessionsByGame(gameId)
        return result.success
          ? { success: true, data: (result.data ?? []) as PlaySessionType[] }
          : { success: false, message: result.error?.message ?? "エラー" }
      },
      updateSessionChapter: async (sessionId, chapterId) => {
        const result = await UpdateSessionChapter(sessionId, chapterId)
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      },
      updateSessionName: async (sessionId, sessionName) => {
        const result = await UpdateSessionName(sessionId, sessionName)
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      },
      deletePlaySession: async (sessionId) => {
        const result = await DeleteSession(sessionId)
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      }
    },
    chapter: {
      getChapters: async (gameId) => {
        const result = await ListChaptersByGame(gameId)
        return result.success
          ? { success: true, data: (result.data ?? []) as Chapter[] }
          : { success: false, message: result.error?.message ?? "エラー" }
      },
      createChapter: async (input) => {
        const result = await CreateChapter({ Name: input.name, Order: 0, GameID: input.gameId })
        return result.success
          ? { success: true, data: result.data as Chapter }
          : { success: false, message: result.error?.message ?? "エラー" }
      },
      updateChapter: async (chapterId, input) => {
        const result = await UpdateChapter(chapterId, { Name: input.name, Order: input.order })
        return result.success
          ? { success: true, data: result.data as Chapter }
          : { success: false, message: result.error?.message ?? "エラー" }
      },
      deleteChapter: async (chapterId) => {
        const result = await DeleteChapter(chapterId)
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      },
      updateChapterOrders: async (gameId, chapterOrders) => {
        const result = await UpdateChapterOrders(gameId, chapterOrders)
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      },
      getChapterStats: async (gameId) => {
        const result = await GetChapterStats(gameId)
        return result.success
          ? { success: true, data: (result.data ?? []) as ChapterStats[] }
          : { success: false, message: result.error?.message ?? "エラー" }
      },
      setCurrentChapter: async (gameId, chapterId) => {
        const result = await SetCurrentChapter(gameId, chapterId)
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      }
    },
    memo: {
      getAllMemos: async () => {
        const result = await ListAllMemos()
        return result.success
          ? { success: true, data: (result.data ?? []) as MemoType[] }
          : { success: false, message: result.error?.message ?? "エラー" }
      },
      getMemoById: async (memoId) => {
        const result = await GetMemoByID(memoId)
        return result.success
          ? { success: true, data: result.data as MemoType }
          : { success: false, message: result.error?.message ?? "エラー" }
      },
      getMemosByGameId: async (gameId) => {
        const result = await ListMemosByGame(gameId)
        return result.success
          ? { success: true, data: (result.data ?? []) as MemoType[] }
          : { success: false, message: result.error?.message ?? "エラー" }
      },
      createMemo: async (data) => {
        const result = await CreateMemo({ Title: data.title, Content: data.content, GameID: data.gameId })
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      },
      updateMemo: async (memoId, data) => {
        const result = await UpdateMemo(memoId, { Title: data.title, Content: data.content })
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      },
      deleteMemo: async (memoId) => {
        const result = await DeleteMemo(memoId)
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      },
      getMemoRootDir: async () => {
        const result = await GetMemoRootDir()
        return result.success
          ? { success: true, data: result.data as string }
          : { success: false, message: result.error?.message ?? "エラー" }
      },
      getMemoFilePath: async (memoId) => {
        const result = await GetMemoFilePath(memoId)
        return result.success
          ? { success: true, data: result.data as string }
          : { success: false, message: result.error?.message ?? "エラー" }
      },
      getGameMemoDir: async (gameId) => {
        const result = await GetGameMemoDir(gameId)
        return result.success
          ? { success: true, data: result.data as string }
          : { success: false, message: result.error?.message ?? "エラー" }
      },
      uploadMemoToCloud: async (memoId) => {
        const result = await UploadMemoToCloud(memoId)
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      },
      downloadMemoFromCloud: async (gameTitle, memoFileName) => {
        const result = await DownloadMemoFromCloud(gameTitle, memoFileName)
        return result.success
          ? { success: true, data: result.data as string }
          : { success: false, message: result.error?.message ?? "エラー" }
      },
      getCloudMemos: async () => {
        const result = await GetCloudMemos()
        return result.success
          ? { success: true, data: (result.data ?? []) as CloudMemoInfo[] }
          : { success: false, message: result.error?.message ?? "エラー" }
      },
      syncMemosFromCloud: async (gameId) => {
        const result = await SyncMemosFromCloud(gameId ?? "")
        return result.success
          ? { success: true, data: result.data as MemoSyncResult }
          : { success: false, message: result.error?.message ?? "エラー" }
      }
    },
    credential: {
      upsertCredential: async (creds) => {
        const result = await SaveCredential("default", {
          AccessKeyID: creds.accessKeyId,
          SecretAccessKey: creds.secretAccessKey
        })
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      },
      getCredential: async () => {
        const result = await LoadCredential("default")
        if (!result.success || !result.data) {
          return { success: false, message: result.error?.message ?? "認証情報がありません" }
        }
        return {
          success: true,
          data: {
            accessKeyId: result.data.accessKeyID,
            secretAccessKey: "",
            bucketName: "",
            region: "",
            endpoint: ""
          }
        }
      },
      validateCredential: async (creds) => {
        const result = await ValidateCredential({
          bucketName: creds.bucketName,
          region: creds.region,
          endpoint: creds.endpoint,
          accessKeyId: creds.accessKeyId,
          secretAccessKey: creds.secretAccessKey
        })
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      }
    },
    cloudData: {
      listCloudData: async () => {
        const result = await ListCloudData()
        return result.success
          ? { success: true, data: (result.data ?? []) as CloudDataItem[] }
          : { success: false, message: result.error?.message ?? "エラー" }
      },
      getDirectoryTree: async () => {
        const result = await GetDirectoryTree()
        return result.success
          ? { success: true, data: (result.data ?? []) as CloudDirectoryNode[] }
          : { success: false, message: result.error?.message ?? "エラー" }
      },
      deleteCloudData: async (path) => {
        const result = await DeleteCloudData(path)
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      },
      deleteFile: async (path) => {
        const result = await DeleteFile(path)
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      },
      getCloudFileDetails: async (path) => {
        const result = await GetCloudFileDetails(path)
        return result.success
          ? { success: true, data: (result.data ?? []) as CloudFileDetail[] }
          : { success: false, message: result.error?.message ?? "エラー" }
      }
    },
    saveData: {
      upload: {
        uploadSaveDataFolder: async (localPath, remotePath) => {
          const result = await UploadFolder("default", localPath, remotePath)
          return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
        }
      },
      download: {
        downloadSaveData: async (localPath, remotePath) => {
          const result = await DownloadSaveData(localPath, remotePath)
          return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
        },
        getCloudFileDetails: async (gameId) => {
          const result = await GetCloudFileDetailsByGame(gameId)
          return result.success
            ? {
                success: true,
                data: result.data as { exists: boolean; totalSize: number; files: CloudFileDetail[] }
              }
            : { success: false, message: result.error?.message ?? "エラー" }
        }
      }
    },
    loadImage: {
      loadImageFromLocal: async (path) => {
        const result = await LoadImageFromLocal(path)
        return result.success
          ? { success: true, data: result.data as string }
          : { success: false, message: result.error?.message ?? "エラー" }
      },
      loadImageFromWeb: async (src) => ({ success: true, data: src })
    },
    processMonitor: {
      getMonitoringStatus: async () => {
        const result = await GetMonitoringStatus()
        return result.success
          ? { success: true, data: (result.data ?? { isMonitoring: false }) as { isMonitoring: boolean } }
          : { success: false, message: result.error?.message ?? "エラー" }
      }
    },
    game: {
      launchGame: async (exePath) => {
        const result = await LaunchGame(exePath)
        return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
      }
    },
    errorReport: {
      reportError: (payload) => {
        console.error(payload)
      },
      reportLog: (payload) => {
        console.log(payload)
      }
    }
  }
}
