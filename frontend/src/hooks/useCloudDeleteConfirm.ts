/**
 * @fileoverview クラウドデータ削除確認の表示文言・警告を組み立てる。
 */

import { useMemo } from "react";

import type { CloudDataItem, CloudDirectoryNode } from "src/types/cloud";
import {
  formatFileSize,
  countFilesRecursively,
  sumSizesRecursively,
} from "@renderer/utils/cloudUtils";
import type { WarningItem } from "@renderer/components/common/ConfirmModal";

export function useCloudDeleteConfirm(
  item: CloudDataItem | CloudDirectoryNode | null,
  cloudData: CloudDataItem[],
): {
  deleteMessage: string;
  fileCountMessage: string;
  subText: string | undefined;
  sizeMessage: string;
  additionalWarnings: WarningItem[];
} {
  const deleteMessage = useMemo((): string => {
    if (!item) {
      return "";
    }

    // path==="*" は全削除センチネル。通常パスと分岐を分ける。
    if ("path" in item && (item as CloudDirectoryNode).path === "*") {
      return "全てのゲームのクラウドデータを完全に削除しますか？";
    }

    return `「${item.name}」のクラウドデータをすべて削除しますか？`;
  }, [item]);

  const fileCountMessage = useMemo((): string => {
    if (!item) {
      return "0 個のファイルが削除されます";
    }

    // path==="*" は全削除センチネル。
    if ("path" in item && (item as CloudDirectoryNode).path === "*") {
      const totalFiles = cloudData.reduce((sum, cloudItem) => sum + cloudItem.fileCount, 0);
      return `全ての ${totalFiles} 個のファイルが削除されます`;
    }

    if ("fileCount" in item) {
      return `${item.fileCount} 個のファイルが削除されます`;
    }

    // ツリーから選んだゲームノードは CloudDirectoryNode 形。
    const node = item as CloudDirectoryNode;
    if (node.isDirectory) {
      const fileCount = countFilesRecursively(node);
      return `${fileCount} 個のファイルが削除されます`;
    }

    return "1 個のファイルが削除されます";
  }, [item, cloudData]);

  const subText = useMemo((): string | undefined => {
    if (!item) {
      return undefined;
    }

    // 全削除では個別パス一覧を出さない（ノイズになる）。
    if ("path" in item && (item as CloudDirectoryNode).path === "*") {
      return undefined;
    }

    // カードとツリーでキー名が違うので remotePath / path のどちらも見る。
    const path = "remotePath" in item ? item.remotePath : (item as CloudDirectoryNode).path;
    return `GameID: ${path}`;
  }, [item]);

  const sizeMessage = useMemo((): string => {
    if (!item) {
      return "総容量: 0 B";
    }

    let size: number;
    if ("totalSize" in item) {
      size = item.totalSize;
    } else {
      const node = item as CloudDirectoryNode;
      size = node.isDirectory ? sumSizesRecursively(node) : node.size;
    }

    return `総容量: ${formatFileSize(size)}`;
  }, [item]);

  const additionalWarnings = useMemo((): WarningItem[] => {
    return [];
  }, []);

  return {
    deleteMessage,
    fileCountMessage,
    subText,
    sizeMessage,
    additionalWarnings,
  };
}
