/**
 * @fileoverview エラー関連型定義
 *
 * AWS SDK エラーやファイル検証エラーの TypeScript 型。
 */

export type AwsSdkError = {
  Code: string;
  message: string;
};

export enum FileValidationError {
  NotFound = "NotFound",
  NoPermission = "NoPermission",
  InvalidExtension = "InvalidExtension",
  NotDir = "NotADirectory",
  Unknown = "Unknown",
}
