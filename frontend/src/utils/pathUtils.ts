/**
 * @fileoverview パス操作ユーティリティ
 *
 * このファイルは、アプリケーション全体で使用されるパス操作関数を提供します。
 * 主な機能：
 * - ファイルパスの検証
 * - パス文字列の正規化
 * - パス操作のヘルパー関数
 * - プラットフォーム固有のパス処理
 */

import { sanitizeFilename } from "./stringUtils";

/**
 * パス検証の種類を表す列挙型
 */
export enum PathType {
  FILE = "file",
  DIRECTORY = "directory",
  ANY = "any",
}

/**
 * パス検証の結果
 */
export type PathValidationResult = {
  /** 検証が成功したかどうか */
  isValid: boolean;
  /** エラーメッセージ（失敗時） */
  message?: string;
  /** 正規化されたパス */
  normalizedPath?: string;
};

/**
 * ファイルパスの基本的な検証
 * @param filePath - 検証するファイルパス
 * @param pathType - パスの種類（ファイル、ディレクトリ、どちらでも）
 * @returns 検証結果
 */
export function validatePath(
  filePath: string,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  _pathType: PathType = PathType.ANY,
): PathValidationResult {
  // 空文字チェック
  if (!filePath || filePath.trim() === "") {
    return {
      isValid: false,
      message: "パスが指定されていません",
    };
  }

  // 危険な文字のチェック（基本的なセキュリティ）
  if (filePath.includes("..")) {
    return {
      isValid: false,
      message: "パスに相対参照（..）を含むことはできません",
    };
  }

  // 元のパス形式を判定（正規化前）
  const isWindowsPath = /^[a-zA-Z]:\\/.test(filePath) || filePath.startsWith("\\\\");
  const isUnixPath = filePath.startsWith("/");

  if (!isWindowsPath && !isUnixPath) {
    return {
      isValid: false,
      message: "絶対パスを指定してください",
    };
  }

  // パスの正規化（形式判定後）
  const normalizedPath = normalizePath(filePath, isUnixPath);

  return { isValid: true, normalizedPath };
}

/**
 * ファイル拡張子を取得
 * @param filePath - ファイルパス
 * @returns 拡張子（ドット含む、例: ".txt"）
 */
export function getFileExtension(filePath: string): string {
  const fileName = filePath.replace(/\\/g, "/").split("/").pop() ?? "";
  const index = fileName.lastIndexOf(".");
  return index >= 0 ? fileName.slice(index) : "";
}

/**
 * ファイル名（拡張子なし）を取得
 * @param filePath - ファイルパス
 * @returns ファイル名（拡張子なし）
 */
export function getFileNameWithoutExtension(filePath: string): string {
  const fileName = filePath.replace(/\\/g, "/").split("/").pop() ?? "";
  const extension = getFileExtension(fileName);
  return extension.length > 0 ? fileName.slice(0, -extension.length) : fileName;
}

/**
 * パスの親ディレクトリを取得
 * @param filePath - ファイルパス
 * @returns 親ディレクトリのパス
 */
export function getParentDirectory(filePath: string): string {
  const normalized = normalizePath(filePath, false);
  if (normalized === "" || normalized === "/" || normalized.endsWith(":/")) {
    return normalized;
  }
  const trimmed = normalized.replace(/\/+$/, "");
  const lastSlash = trimmed.lastIndexOf("/");
  if (lastSlash <= 0) {
    return trimmed;
  }
  return trimmed.slice(0, lastSlash);
}

/**
 * 複数のパス要素を結合
 * @param pathSegments - パス要素の配列
 * @returns 結合されたパス（Unix形式のスラッシュを使用）
 */
export function joinPaths(...pathSegments: string[]): string {
  if (pathSegments.length === 0) {
    return "";
  }
  const filtered = pathSegments.filter(Boolean);
  if (filtered.length === 0) {
    return "";
  }
  const joined = filtered.join("/");
  return joined.replace(/\\/g, "/").replace(/\/{2,}/g, "/");
}

/**
 * ファイル名をサニタイズしてパスセーフにする
 * @param fileName - サニタイズするファイル名
 * @returns サニタイズされたファイル名
 */
export function sanitizeFileName(fileName: string): string {
  return sanitizeFilename(fileName);
}

/**
 * S3キー用のパスを生成（常にUnix形式のスラッシュを使用）
 * @param pathSegments - パス要素の配列
 * @returns S3キー形式のパス
 */
export function createS3Key(...pathSegments: string[]): string {
  return pathSegments.filter(Boolean).join("/");
}

/**
 * ローカルパスをS3キー形式に変換
 * @param localPath - ローカルファイルパス
 * @returns S3キー形式のパス
 */
export function localPathToS3Key(localPath: string): string {
  // バックスラッシュをスラッシュに変換（Windows対応）
  return localPath.replace(/\\/g, "/");
}

/**
 * 相対パスかどうかを判定
 * @param filePath - チェックするパス
 * @returns 相対パスの場合 true
 */
export function isRelativePath(filePath: string): boolean {
  return (
    !/^[a-zA-Z]:[\\/]/.test(filePath) && !filePath.startsWith("/") && !filePath.startsWith("\\\\")
  );
}

/**
 * 指定された拡張子を持つファイルかどうかを判定
 * @param filePath - チェックするファイルパス
 * @param extensions - 許可する拡張子の配列（ドットなし、例: ["jpg", "png"]）
 * @returns 指定された拡張子を持つ場合 true
 */
export function hasValidExtension(filePath: string, extensions: readonly string[]): boolean {
  const fileExtension = getFileExtension(filePath).toLowerCase().slice(1); // ドットを除去
  return extensions.map((ext) => ext.toLowerCase()).includes(fileExtension);
}

function normalizePath(input: string, preserveUnix: boolean): string {
  let normalized = input.replace(/\\/g, "/");
  normalized = normalized.replace(/\/{2,}/g, "/");
  if (preserveUnix && input.startsWith("/")) {
    return normalized;
  }
  return normalized;
}
