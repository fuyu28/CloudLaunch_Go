/**
 * @fileoverview パス操作ユーティリティ。Windows のバックスラッシュ区切りを正規化した上で親ディレクトリを求める。
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
