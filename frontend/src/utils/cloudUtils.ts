/**
 * @fileoverview クラウドデータ関連のユーティリティ関数
 *
 * このファイルは、クラウドデータ管理で使用される共通の
 * ユーティリティ関数を提供します。
 */

/**
 * ファイルサイズを人間が読みやすい形式に変換
 * @param bytes バイト数
 * @returns 読みやすい形式の文字列
 */
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

/**
 * 日時を読みやすい形式に変換
 * @param date 日時
 * @returns 読みやすい形式の文字列
 */
export function formatDate(date: Date): string {
  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(date));
}

/**
 * クラウドディレクトリツリーノードの型定義
 */
export type CloudDirectoryNode = {
  name: string;
  path: string;
  isDirectory: boolean;
  size: number;
  lastModified: Date;
  children?: CloudDirectoryNode[];
  objectKey?: string;
};

/**
 * ディレクトリノードから再帰的にファイル数を計算
 * @param node ディレクトリノード
 * @returns ファイル数
 */
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

/**
 * ディレクトリノードから再帰的に合計サイズを計算
 * @param node ディレクトリノード
 * @returns 合計サイズ
 */
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

/**
 * ディレクトリノードから最新の更新日時を取得
 * @param node ディレクトリノード
 * @returns 最新の更新日時
 */
export function latestModifiedRecursively(node: CloudDirectoryNode): Date {
  const baseTime = node.lastModified instanceof Date ? node.lastModified.getTime() : 0;
  let latest = Number.isFinite(baseTime) ? baseTime : 0;

  if (node.children && node.children.length > 0) {
    node.children.forEach((child) => {
      const childTime = latestModifiedRecursively(child).getTime();
      if (childTime > latest) {
        latest = childTime;
      }
    });
  }

  return new Date(latest);
}

/**
 * 指定したパスの子ディレクトリ・ファイルを取得
 * @param tree ディレクトリツリー
 * @param path パス配列
 * @returns 子ノード配列
 */
export function getNodesByPath(tree: CloudDirectoryNode[], path: string[]): CloudDirectoryNode[] {
  if (path.length === 0) {
    return tree;
  }

  let currentNodes = tree;
  for (const pathSegment of path) {
    const targetNode = currentNodes.find((node) => node.name === pathSegment && node.isDirectory);
    if (!targetNode || !targetNode.children) {
      return [];
    }
    currentNodes = targetNode.children;
  }
  return currentNodes;
}
