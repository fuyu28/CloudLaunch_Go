export function ListGames(arg1: string, arg2: string, arg3: string, arg4: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function GetGameByID(arg1: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function CreateGame(arg1: unknown): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function UpdateGame(arg1: string, arg2: unknown): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function UpdatePlayTime(arg1: string, arg2: number, arg3: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function DeleteGame(arg1: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function ListChaptersByGame(arg1: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function CreateChapter(arg1: unknown): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function UpdateChapter(arg1: string, arg2: unknown): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function DeleteChapter(arg1: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function CreateSession(arg1: unknown): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function ListSessionsByGame(arg1: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function DeleteSession(arg1: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function CreateMemo(arg1: unknown): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function GetMemoByID(arg1: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function ListAllMemos(): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function UpdateMemo(arg1: string, arg2: unknown): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function ListMemosByGame(arg1: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function DeleteMemo(arg1: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function SelectFile(arg1: unknown): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function SelectFolder(): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function CheckFileExists(arg1: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function CheckDirectoryExists(arg1: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function OpenFolder(arg1: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function OpenLogsDirectory(): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function CreateUpload(arg1: unknown): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function ListUploadsByGame(arg1: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function SaveCredential(arg1: string, arg2: unknown): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function LoadCredential(arg1: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function DeleteCredential(arg1: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function UploadFolder(arg1: string, arg2: string, arg3: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function SaveCloudMetadata(arg1: string, arg2: unknown): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
export function LoadCloudMetadata(arg1: string): Promise<{ success: boolean; data?: unknown; error?: { message: string; detail?: string } }>
