/**
 * @fileoverview クラウドデータ関連のユーティリティ関数
 *
 * このファイルは、クラウドデータ管理で使用される共通の
 * ユーティリティ関数を提供します。
 */

export function formatFileSize(bytes: number): string {
  // null / undefined / NaN / 負値 / Infinity は "0 B" にフォールバックする（NaN / undefined 対策）。
  if (bytes == null || !Number.isFinite(bytes) || bytes < 0) return "0 B";
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  // 巨大バイト数で sizes 配列をはみ出さないようインデックスをクランプする。
  const i = Math.min(sizes.length - 1, Math.floor(Math.log(bytes) / Math.log(k)));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

export function formatDate(date: Date | string | null | undefined): string {
  const normalized = date instanceof Date ? date : new Date(date ?? Number.NaN);
  if (Number.isNaN(normalized.getTime()) || normalized.getTime() <= 0) {
    return "不明";
  }
  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(normalized);
}

import type { CloudDirectoryNode } from "src/types/cloud";
export type { CloudDirectoryNode } from "src/types/cloud";

/**
 * ノードが「中身まで取得済み」かを判定する。
 * ファイルは常に取得済み。ディレクトリは children が undefined（=未取得）の
 * 間は false、空配列以上が入った時点で true。クラウドデータ管理ページの
 * 遅延取得ゲームに対するファイル数 / サイズ表示の出し分けに使う。
 */
export function isCloudNodeLoaded(node: CloudDirectoryNode): boolean {
  return !node.isDirectory || node.children !== undefined;
}

/**
 * クラウドノードの表示用メトリクス。
 * - childrenLoaded: ナビゲーションで配下を取得済みか
 * - count: 表示するファイル数（取得済み→配下集計、未取得→commit メタ由来）
 * - size: 表示する合計サイズ（同上）
 * - hasMetrics: 値を表示すべきか。ファイル／取得済みディレクトリ／キャッシュ持ちは true
 */
export type CloudNodeMetrics = {
  childrenLoaded: boolean;
  count: number;
  size: number;
  hasMetrics: boolean;
};

/**
 * カード／ツリー／集計の 3 箇所で同じ「子取得済みなら配下集計、未取得ならサマリ値」
 * の判定を繰り返していたため共通化する。表示判定のルールがブレないよう
 * 必ずこの関数を経由する。
 */
export function computeCloudNodeMetrics(node: CloudDirectoryNode): CloudNodeMetrics {
  if (!node.isDirectory) {
    return { childrenLoaded: true, count: 1, size: node.size, hasMetrics: true };
  }
  const childrenLoaded = node.children !== undefined;
  const count = childrenLoaded ? countFilesRecursively(node) : (node.fileCount ?? 0);
  const size = childrenLoaded ? sumSizesRecursively(node) : node.size;
  return {
    childrenLoaded,
    count,
    size,
    hasMetrics: childrenLoaded || count > 0,
  };
}

export function countFilesRecursively(node: CloudDirectoryNode): number {
  if (!node.isDirectory) {
    return 1;
  }

  let fileCount = 0;
  if (node.children) {
    node.children.forEach((child) => {
      fileCount += countFilesRecursively(child);
    });
  }
  return fileCount;
}

export function sumSizesRecursively(node: CloudDirectoryNode): number {
  if (!node.isDirectory) {
    return node.size;
  }

  let totalSize = 0;
  if (node.children) {
    node.children.forEach((child) => {
      totalSize += sumSizesRecursively(child);
    });
  }
  return totalSize;
}

export function latestModifiedRecursively(node: CloudDirectoryNode): Date {
  const baseDate =
    node.lastModified instanceof Date ? node.lastModified : new Date(node.lastModified);
  const baseTime = baseDate.getTime();
  let latest = Number.isFinite(baseTime) && baseTime > 0 ? baseTime : 0;

  if (node.children && node.children.length > 0) {
    node.children.forEach((child) => {
      const childTime = latestModifiedRecursively(child).getTime();
      if (childTime > latest) {
        latest = childTime;
      }
    });
  }

  return latest > 0 ? new Date(latest) : new Date(Number.NaN);
}

/**
 * ナビゲーション用のパスセグメント。
 * 同じ表示名のゲームが複数存在するケースを区別するため、
 * `id`（CloudDirectoryNode.path。ルートではゲームID）で識別し、
 * `name` は表示用（パンくず等）にのみ使う。
 */
export type CloudPathSegment = {
  id: string;
  name: string;
};

/**
 * 指定パスの子ノードを返す。
 * 表示名ではなく `node.path`（一意）で解決し、同名ゲームの混同を避ける。
 */
export function getNodesByPath(
  tree: CloudDirectoryNode[],
  path: CloudPathSegment[],
): CloudDirectoryNode[] {
  if (path.length === 0) {
    return tree;
  }

  let currentNodes = tree;
  for (const segment of path) {
    const targetNode = currentNodes.find((node) => node.path === segment.id && node.isDirectory);
    if (!targetNode || !targetNode.children) {
      return [];
    }
    currentNodes = targetNode.children;
  }
  return currentNodes;
}
