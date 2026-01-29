export namespace app {
	
	export class CloudDataItem {
	    name: string;
	    totalSize: number;
	    fileCount: number;
	    lastModified: time.Time;
	    remotePath: string;
	
	    static createFrom(source: any = {}) {
	        return new CloudDataItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.totalSize = source["totalSize"];
	        this.fileCount = source["fileCount"];
	        this.lastModified = this.convertValues(source["lastModified"], time.Time);
	        this.remotePath = source["remotePath"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CloudDirectoryNode {
	    name: string;
	    path: string;
	    isDirectory: boolean;
	    size: number;
	    lastModified: time.Time;
	    children?: CloudDirectoryNode[];
	    objectKey?: string;
	
	    static createFrom(source: any = {}) {
	        return new CloudDirectoryNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDirectory = source["isDirectory"];
	        this.size = source["size"];
	        this.lastModified = this.convertValues(source["lastModified"], time.Time);
	        this.children = this.convertValues(source["children"], CloudDirectoryNode);
	        this.objectKey = source["objectKey"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CloudFileDetail {
	    name: string;
	    size: number;
	    lastModified: time.Time;
	    key: string;
	    relativePath: string;
	
	    static createFrom(source: any = {}) {
	        return new CloudFileDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.size = source["size"];
	        this.lastModified = this.convertValues(source["lastModified"], time.Time);
	        this.key = source["key"];
	        this.relativePath = source["relativePath"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CloudFileDetailsResult {
	    exists: boolean;
	    totalSize: number;
	    files: CloudFileDetail[];
	
	    static createFrom(source: any = {}) {
	        return new CloudFileDetailsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.exists = source["exists"];
	        this.totalSize = source["totalSize"];
	        this.files = this.convertValues(source["files"], CloudFileDetail);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CloudMemoInfo {
	    key: string;
	    fileName: string;
	    gameTitle: string;
	    memoTitle: string;
	    memoId: string;
	    lastModified: time.Time;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new CloudMemoInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.fileName = source["fileName"];
	        this.gameTitle = source["gameTitle"];
	        this.memoTitle = source["memoTitle"];
	        this.memoId = source["memoId"];
	        this.lastModified = this.convertValues(source["lastModified"], time.Time);
	        this.size = source["size"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CredentialValidationInput {
	    bucketName: string;
	    region: string;
	    endpoint: string;
	    accessKeyId: string;
	    secretAccessKey: string;
	
	    static createFrom(source: any = {}) {
	        return new CredentialValidationInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bucketName = source["bucketName"];
	        this.region = source["region"];
	        this.endpoint = source["endpoint"];
	        this.accessKeyId = source["accessKeyId"];
	        this.secretAccessKey = source["secretAccessKey"];
	    }
	}
	export class FileFilterInput {
	    name: string;
	    extensions: string[];
	
	    static createFrom(source: any = {}) {
	        return new FileFilterInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.extensions = source["extensions"];
	    }
	}
	export class MemoSyncResult {
	    success: boolean;
	    uploaded: number;
	    localOverwritten: number;
	    cloudOverwritten: number;
	    created: number;
	    updated: number;
	    skipped: number;
	    error?: string;
	    details: string[];
	
	    static createFrom(source: any = {}) {
	        return new MemoSyncResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.uploaded = source["uploaded"];
	        this.localOverwritten = source["localOverwritten"];
	        this.cloudOverwritten = source["cloudOverwritten"];
	        this.created = source["created"];
	        this.updated = source["updated"];
	        this.skipped = source["skipped"];
	        this.error = source["error"];
	        this.details = source["details"];
	    }
	}

}

export namespace models {
	
	export class Chapter {
	    id: string;
	    name: string;
	    order: number;
	    gameId: string;
	    createdAt: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new Chapter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.order = source["order"];
	        this.gameId = source["gameId"];
	        this.createdAt = this.convertValues(source["createdAt"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ChapterStat {
	    chapterId: string;
	    chapterName: string;
	    totalTime: number;
	    sessionCount: number;
	    averageTime: number;
	    order: number;
	
	    static createFrom(source: any = {}) {
	        return new ChapterStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapterId = source["chapterId"];
	        this.chapterName = source["chapterName"];
	        this.totalTime = source["totalTime"];
	        this.sessionCount = source["sessionCount"];
	        this.averageTime = source["averageTime"];
	        this.order = source["order"];
	    }
	}
	export class Game {
	    id: string;
	    title: string;
	    publisher: string;
	    imagePath?: string;
	    exePath: string;
	    saveFolderPath?: string;
	    createdAt: time.Time;
	    playStatus: string;
	    totalPlayTime: number;
	    lastPlayed?: time.Time;
	    clearedAt?: time.Time;
	    currentChapter?: string;
	
	    static createFrom(source: any = {}) {
	        return new Game(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.publisher = source["publisher"];
	        this.imagePath = source["imagePath"];
	        this.exePath = source["exePath"];
	        this.saveFolderPath = source["saveFolderPath"];
	        this.createdAt = this.convertValues(source["createdAt"], time.Time);
	        this.playStatus = source["playStatus"];
	        this.totalPlayTime = source["totalPlayTime"];
	        this.lastPlayed = this.convertValues(source["lastPlayed"], time.Time);
	        this.clearedAt = this.convertValues(source["clearedAt"], time.Time);
	        this.currentChapter = source["currentChapter"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Memo {
	    id: string;
	    title: string;
	    content: string;
	    gameId: string;
	    createdAt: time.Time;
	    updatedAt: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new Memo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.gameId = source["gameId"];
	        this.createdAt = this.convertValues(source["createdAt"], time.Time);
	        this.updatedAt = this.convertValues(source["updatedAt"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MonitoringGameStatus {
	    gameId: string;
	    gameTitle: string;
	    exeName: string;
	    isPlaying: boolean;
	    playTime: number;
	
	    static createFrom(source: any = {}) {
	        return new MonitoringGameStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gameId = source["gameId"];
	        this.gameTitle = source["gameTitle"];
	        this.exeName = source["exeName"];
	        this.isPlaying = source["isPlaying"];
	        this.playTime = source["playTime"];
	    }
	}
	export class PlaySession {
	    id: string;
	    gameId: string;
	    playedAt: time.Time;
	    duration: number;
	    sessionName?: string;
	    chapterId?: string;
	    uploadId?: string;
	
	    static createFrom(source: any = {}) {
	        return new PlaySession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.gameId = source["gameId"];
	        this.playedAt = this.convertValues(source["playedAt"], time.Time);
	        this.duration = source["duration"];
	        this.sessionName = source["sessionName"];
	        this.chapterId = source["chapterId"];
	        this.uploadId = source["uploadId"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Upload {
	    id: string;
	    clientId?: string;
	    comment: string;
	    createdAt: time.Time;
	    gameId: string;
	
	    static createFrom(source: any = {}) {
	        return new Upload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.clientId = source["clientId"];
	        this.comment = source["comment"];
	        this.createdAt = this.convertValues(source["createdAt"], time.Time);
	        this.gameId = source["gameId"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace result {
	
	export class ApiError {
	    message: string;
	    detail: string;
	    at: time.Time;
	
	    static createFrom(source: any = {}) {
	        return new ApiError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.detail = source["detail"];
	        this.at = this.convertValues(source["at"], time.Time);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult__CloudLaunch_Go_internal_models_Chapter_ {
	    success: boolean;
	    data?: models.Chapter;
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult__CloudLaunch_Go_internal_models_Chapter_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], models.Chapter);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult__CloudLaunch_Go_internal_models_Game_ {
	    success: boolean;
	    data?: models.Game;
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult__CloudLaunch_Go_internal_models_Game_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], models.Game);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult__CloudLaunch_Go_internal_models_Memo_ {
	    success: boolean;
	    data?: models.Memo;
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult__CloudLaunch_Go_internal_models_Memo_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], models.Memo);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult__CloudLaunch_Go_internal_models_PlaySession_ {
	    success: boolean;
	    data?: models.PlaySession;
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult__CloudLaunch_Go_internal_models_PlaySession_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], models.PlaySession);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult__CloudLaunch_Go_internal_models_Upload_ {
	    success: boolean;
	    data?: models.Upload;
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult__CloudLaunch_Go_internal_models_Upload_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], models.Upload);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult__CloudLaunch_Go_internal_services_CredentialOutput_ {
	    success: boolean;
	    data?: services.CredentialOutput;
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult__CloudLaunch_Go_internal_services_CredentialOutput_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], services.CredentialOutput);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult__CloudLaunch_Go_internal_storage_CloudMetadata_ {
	    success: boolean;
	    data?: storage.CloudMetadata;
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult__CloudLaunch_Go_internal_storage_CloudMetadata_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], storage.CloudMetadata);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult_CloudLaunch_Go_internal_app_CloudFileDetailsResult_ {
	    success: boolean;
	    data?: app.CloudFileDetailsResult;
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult_CloudLaunch_Go_internal_app_CloudFileDetailsResult_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], app.CloudFileDetailsResult);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult_CloudLaunch_Go_internal_app_MemoSyncResult_ {
	    success: boolean;
	    data?: app.MemoSyncResult;
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult_CloudLaunch_Go_internal_app_MemoSyncResult_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], app.MemoSyncResult);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult_CloudLaunch_Go_internal_storage_UploadSummary_ {
	    success: boolean;
	    data?: storage.UploadSummary;
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult_CloudLaunch_Go_internal_storage_UploadSummary_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], storage.UploadSummary);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult___CloudLaunch_Go_internal_app_CloudDataItem_ {
	    success: boolean;
	    data?: app.CloudDataItem[];
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult___CloudLaunch_Go_internal_app_CloudDataItem_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], app.CloudDataItem);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult___CloudLaunch_Go_internal_app_CloudDirectoryNode_ {
	    success: boolean;
	    data?: app.CloudDirectoryNode[];
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult___CloudLaunch_Go_internal_app_CloudDirectoryNode_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], app.CloudDirectoryNode);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult___CloudLaunch_Go_internal_app_CloudFileDetail_ {
	    success: boolean;
	    data?: app.CloudFileDetail[];
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult___CloudLaunch_Go_internal_app_CloudFileDetail_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], app.CloudFileDetail);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult___CloudLaunch_Go_internal_app_CloudMemoInfo_ {
	    success: boolean;
	    data?: app.CloudMemoInfo[];
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult___CloudLaunch_Go_internal_app_CloudMemoInfo_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], app.CloudMemoInfo);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult___CloudLaunch_Go_internal_models_ChapterStat_ {
	    success: boolean;
	    data?: models.ChapterStat[];
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult___CloudLaunch_Go_internal_models_ChapterStat_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], models.ChapterStat);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult___CloudLaunch_Go_internal_models_Chapter_ {
	    success: boolean;
	    data?: models.Chapter[];
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult___CloudLaunch_Go_internal_models_Chapter_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], models.Chapter);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult___CloudLaunch_Go_internal_models_Game_ {
	    success: boolean;
	    data?: models.Game[];
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult___CloudLaunch_Go_internal_models_Game_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], models.Game);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult___CloudLaunch_Go_internal_models_Memo_ {
	    success: boolean;
	    data?: models.Memo[];
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult___CloudLaunch_Go_internal_models_Memo_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], models.Memo);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult___CloudLaunch_Go_internal_models_MonitoringGameStatus_ {
	    success: boolean;
	    data?: models.MonitoringGameStatus[];
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult___CloudLaunch_Go_internal_models_MonitoringGameStatus_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], models.MonitoringGameStatus);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult___CloudLaunch_Go_internal_models_PlaySession_ {
	    success: boolean;
	    data?: models.PlaySession[];
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult___CloudLaunch_Go_internal_models_PlaySession_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], models.PlaySession);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult___CloudLaunch_Go_internal_models_Upload_ {
	    success: boolean;
	    data?: models.Upload[];
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult___CloudLaunch_Go_internal_models_Upload_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = this.convertValues(source["data"], models.Upload);
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult_bool_ {
	    success: boolean;
	    data?: boolean;
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult_bool_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = source["data"];
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ApiResult_string_ {
	    success: boolean;
	    data?: string;
	    error?: ApiError;
	
	    static createFrom(source: any = {}) {
	        return new ApiResult_string_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.data = source["data"];
	        this.error = this.convertValues(source["error"], ApiError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace services {
	
	export class ChapterInput {
	    Name: string;
	    Order: number;
	    GameID: string;
	
	    static createFrom(source: any = {}) {
	        return new ChapterInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Order = source["Order"];
	        this.GameID = source["GameID"];
	    }
	}
	export class ChapterOrderUpdate {
	    ID: string;
	    Order: number;
	
	    static createFrom(source: any = {}) {
	        return new ChapterOrderUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Order = source["Order"];
	    }
	}
	export class ChapterUpdateInput {
	    Name: string;
	    Order: number;
	
	    static createFrom(source: any = {}) {
	        return new ChapterUpdateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Order = source["Order"];
	    }
	}
	export class CredentialInput {
	    AccessKeyID: string;
	    SecretAccessKey: string;
	    BucketName: string;
	    Region: string;
	    Endpoint: string;
	
	    static createFrom(source: any = {}) {
	        return new CredentialInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.AccessKeyID = source["AccessKeyID"];
	        this.SecretAccessKey = source["SecretAccessKey"];
	        this.BucketName = source["BucketName"];
	        this.Region = source["Region"];
	        this.Endpoint = source["Endpoint"];
	    }
	}
	export class CredentialOutput {
	    AccessKeyID: string;
	    BucketName: string;
	    Region: string;
	    Endpoint: string;
	
	    static createFrom(source: any = {}) {
	        return new CredentialOutput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.AccessKeyID = source["AccessKeyID"];
	        this.BucketName = source["BucketName"];
	        this.Region = source["Region"];
	        this.Endpoint = source["Endpoint"];
	    }
	}
	export class GameInput {
	    Title: string;
	    Publisher: string;
	    ImagePath?: string;
	    ExePath: string;
	    SaveFolderPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new GameInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Title = source["Title"];
	        this.Publisher = source["Publisher"];
	        this.ImagePath = source["ImagePath"];
	        this.ExePath = source["ExePath"];
	        this.SaveFolderPath = source["SaveFolderPath"];
	    }
	}
	export class GameUpdateInput {
	    Title: string;
	    Publisher: string;
	    ImagePath?: string;
	    ExePath: string;
	    SaveFolderPath?: string;
	    PlayStatus: string;
	    ClearedAt?: time.Time;
	    CurrentChapter?: string;
	
	    static createFrom(source: any = {}) {
	        return new GameUpdateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Title = source["Title"];
	        this.Publisher = source["Publisher"];
	        this.ImagePath = source["ImagePath"];
	        this.ExePath = source["ExePath"];
	        this.SaveFolderPath = source["SaveFolderPath"];
	        this.PlayStatus = source["PlayStatus"];
	        this.ClearedAt = this.convertValues(source["ClearedAt"], time.Time);
	        this.CurrentChapter = source["CurrentChapter"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MemoInput {
	    Title: string;
	    Content: string;
	    GameID: string;
	
	    static createFrom(source: any = {}) {
	        return new MemoInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Title = source["Title"];
	        this.Content = source["Content"];
	        this.GameID = source["GameID"];
	    }
	}
	export class MemoUpdateInput {
	    Title: string;
	    Content: string;
	
	    static createFrom(source: any = {}) {
	        return new MemoUpdateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Title = source["Title"];
	        this.Content = source["Content"];
	    }
	}
	export class SessionInput {
	    GameID: string;
	    PlayedAt: time.Time;
	    Duration: number;
	    SessionName?: string;
	    ChapterID?: string;
	    UploadID?: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.GameID = source["GameID"];
	        this.PlayedAt = this.convertValues(source["PlayedAt"], time.Time);
	        this.Duration = source["Duration"];
	        this.SessionName = source["SessionName"];
	        this.ChapterID = source["ChapterID"];
	        this.UploadID = source["UploadID"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UploadInput {
	    ClientID?: string;
	    Comment: string;
	    GameID: string;
	
	    static createFrom(source: any = {}) {
	        return new UploadInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ClientID = source["ClientID"];
	        this.Comment = source["Comment"];
	        this.GameID = source["GameID"];
	    }
	}

}

export namespace storage {
	
	export class CloudGameMetadata {
	    id: string;
	    title: string;
	    publisher: string;
	    imageKey: string;
	    totalPlayTime: number;
	    playStatus: string;
	    tags: string[];
	
	    static createFrom(source: any = {}) {
	        return new CloudGameMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.publisher = source["publisher"];
	        this.imageKey = source["imageKey"];
	        this.totalPlayTime = source["totalPlayTime"];
	        this.playStatus = source["playStatus"];
	        this.tags = source["tags"];
	    }
	}
	export class CloudMetadata {
	    games: CloudGameMetadata[];
	
	    static createFrom(source: any = {}) {
	        return new CloudMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.games = this.convertValues(source["games"], CloudGameMetadata);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UploadSummary {
	    FileCount: number;
	    TotalBytes: number;
	    Keys: string[];
	
	    static createFrom(source: any = {}) {
	        return new UploadSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.FileCount = source["FileCount"];
	        this.TotalBytes = source["TotalBytes"];
	        this.Keys = source["Keys"];
	    }
	}

}

export namespace time {
	
	export class Time {
	
	
	    static createFrom(source: any = {}) {
	        return new Time(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

