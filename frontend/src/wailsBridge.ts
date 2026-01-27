/**
 * @fileoverview Electron IPC互換のWailsブリッジを提供する。
 */

import type { ApiResult } from "src/types/result"
import type { InputGameData, GameType, PlaySessionType, PlayStatus } from "src/types/game"
import type { SortOption, FilterOption, SortDirection } from "src/types/menu"
import type { Chapter, ChapterStats } from "src/types/chapter"
import type { MemoType, CreateMemoData, UpdateMemoData } from "src/types/memo"
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
  DeleteChapter,
  DeleteCredential,
  DeleteGame,
  DeleteMemo,
  DeleteSession,
  GetMemoByID,
  GetGameByID,
  ListChaptersByGame,
  ListAllMemos,
  ListGames,
  ListMemosByGame,
  ListSessionsByGame,
  ListUploadsByGame,
  LoadCloudMetadata,
  LoadCredential,
  OpenFolder,
  OpenLogsDirectory,
  SaveCloudMetadata,
  SaveCredential,
  SelectFile,
  SelectFolder,
  UpdateChapter,
  UpdateGame,
  UpdateMemo,
  UploadFolder
} from "../wailsjs/go/app/App"
import { WindowClose, WindowMinimise, WindowToggleMaximise } from "../wailsjs/runtime/runtime"

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
    syncMemosFromCloud: (gameId: string) => Promise<ApiResult<void>>
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
    getCloudFileDetails: (path: string) => Promise<ApiResult<CloudFileDetail>>
  }
  saveData: {
    upload: {
      uploadSaveDataFolder: (localPath: string, remotePath: string) => Promise<ApiResult<void>>
    }
    download: {
      downloadSaveData: (localPath: string, remotePath: string) => Promise<ApiResult<void>>
      getCloudFileDetails: (gameId: string) => Promise<ApiResult<CloudFileDetail>>
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
  const notImplemented = async <T,>(message: string): Promise<ApiResult<T>> => ({
    success: false,
    message
  })
  const okResult = <T,>(data: T): ApiResult<T> => ({ success: true, data })

  return {
    window: {
      minimize: async () => {
        await WindowMinimise()
      },
      toggleMaximize: async () => {
        await WindowToggleMaximise()
      },
      close: async () => {
        await WindowClose()
      },
      openFolder: async (path) => {
        await OpenFolder(path)
      }
    },
    settings: {
      updateAutoTracking: async () => notImplemented<void>("設定APIは未実装です")
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
      updateSessionChapter: async () => notImplemented<void>("セッション更新は未実装です"),
      updateSessionName: async () => notImplemented<void>("セッション更新は未実装です"),
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
      updateChapterOrders: async () => notImplemented<void>("章の並び替えは未実装です"),
      getChapterStats: async () => notImplemented<ChapterStats[]>("章統計は未実装です"),
      setCurrentChapter: async () => notImplemented<void>("現在章設定は未実装です")
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
      getMemoRootDir: async () => notImplemented<string>("メモ保存先は未実装です"),
      getMemoFilePath: async () => notImplemented<string>("メモファイルパスは未実装です"),
      syncMemosFromCloud: async () => notImplemented<void>("メモ同期は未実装です")
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
      validateCredential: async () => notImplemented<void>("認証情報検証は未実装です")
    },
    cloudData: {
      listCloudData: async () => okResult<CloudDataItem[]>([]),
      getDirectoryTree: async () => okResult<CloudDirectoryNode[]>([]),
      deleteCloudData: async () => notImplemented<void>("クラウド削除は未実装です"),
      deleteFile: async () => notImplemented<void>("クラウド削除は未実装です"),
      getCloudFileDetails: async () => notImplemented<CloudFileDetail>("クラウド詳細は未実装です")
    },
    saveData: {
      upload: {
        uploadSaveDataFolder: async (localPath, remotePath) => {
          const result = await UploadFolder("default", localPath, remotePath)
          return result.success ? { success: true } : { success: false, message: result.error?.message ?? "エラー" }
        }
      },
      download: {
        downloadSaveData: async () => notImplemented<void>("ダウンロードは未実装です"),
        getCloudFileDetails: async () => notImplemented<CloudFileDetail>("クラウド詳細は未実装です")
      }
    },
    loadImage: {
      loadImageFromLocal: async () => notImplemented<string>("画像読み込みは未実装です"),
      loadImageFromWeb: async (src) => ({ success: true, data: src })
    },
    processMonitor: {
      getMonitoringStatus: async () => ({ success: true, data: { isMonitoring: false } })
    },
    game: {
      launchGame: async () => notImplemented<void>("ゲーム起動は未実装です")
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
