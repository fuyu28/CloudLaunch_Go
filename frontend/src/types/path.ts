/**
 * @fileoverview パス関連型定義
 *
 * このファイルは、アプリケーション全体で使用されるパス関連の型を定義します。
 */

export enum PathType {
  FILE = "file",
  DIRECTORY = "directory",
  ANY = "any",
}

export enum FilePathType {
  EXECUTABLE = "executable",
  IMAGE = "image",
  CONFIG = "config",
  DATA = "data",
  GENERAL = "general",
}

export type PathValidationResult = {
  isValid: boolean;
  message?: string;
  normalizedPath?: string;
  detectedType?: PathType;
};

export type FileInfo = {
  path: string;
  name: string;
  nameWithoutExtension: string;
  extension: string;
  directory: string;
  size?: number;
  lastModified?: Date;
};

export type PathOptions = {
  normalize?: boolean;
  allowRelative?: boolean;
  platformSpecific?: boolean;
};

export type FileFilter = {
  name: string;
  extensions: string[];
};

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
