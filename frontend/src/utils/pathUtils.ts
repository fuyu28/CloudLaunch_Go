/**
 * @fileoverview パス操作ユーティリティ
 */

/**
 * パスの親ディレクトリを取得
 * @param filePath - ファイルパス
 * @returns 親ディレクトリのパス
 */
export function getParentDirectory(filePath: string): string {
  const normalized = normalizeWindowsPath(filePath);
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

function normalizeWindowsPath(input: string): string {
  let normalized = input.replace(/\\/g, "/");
  normalized = normalized.replace(/\/{2,}/g, "/");
  return normalized;
}
