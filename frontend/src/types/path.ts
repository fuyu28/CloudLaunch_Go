/**
 * @fileoverview パス関連型定義
 *
 * このファイルは、アプリケーション全体で使用されるパス関連の型を定義します。
 */

/**
 * パスの種類を表す列挙型
 */
export enum PathType {
  FILE = "file",
  DIRECTORY = "directory",
  ANY = "any",
}

/**
 * ファイルパスの処理タイプ
 */
export enum FilePathType {
  EXECUTABLE = "executable",
  IMAGE = "image",
  CONFIG = "config",
  DATA = "data",
  GENERAL = "general",
}

/**
 * パス検証の結果
 */
export type PathValidationResult = {
  isValid: boolean;
  message?: string;
  normalizedPath?: string;
  detectedType?: PathType;
};

/**
 * ファイル情報
 */
export type FileInfo = {
  path: string;
  name: string;
  nameWithoutExtension: string;
  extension: string;
  directory: string;
  size?: number;
  lastModified?: Date;
};

/**
 * パス操作のオプション
 */
export type PathOptions = {
  normalize?: boolean;
  allowRelative?: boolean;
  platformSpecific?: boolean;
};

/**
 * ファイル選択フィルター
 */
export type FileFilter = {
  name: string;
  extensions: string[];
};

/**
 * S3キー情報
 */
export type S3KeyInfo = {
  key: string;
  bucket?: string;
  region?: string;
  prefix?: string;
};

export type RemotePathConfig = {
  basePath: string;
  gameNameTemplate?: string;
  saveDataFolder?: string;
  dateFormat?: string;
};
