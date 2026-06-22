/**
 * @fileoverview クラウドデータ削除確認ロジック
 *
 * このフックは、クラウドデータの削除確認に関するロジックを
 * カプセル化し、再利用可能な形で提供します。
 *
 * 削除はゲーム単位のみ。content-addressed ストレージ（git スタイル）では
 * ブロブ単位削除は履歴破壊になるため、個別ファイル削除 UI は廃止。
 */

import { useMemo } from "react";

import type { CloudDataItem, CloudDirectoryNode } from "src/types/cloud";
import {
  formatFileSize,
  countFilesRecursively,
  sumSizesRecursively,
} from "@renderer/utils/cloudUtils";
import type { WarningItem } from "@renderer/components/ConfirmModal";

/**
 * 削除確認ロジックのフック
 *
 * @param item 削除対象のアイテム（ゲーム単位の CloudDataItem or CloudDirectoryNode）
 * @param cloudData 全クラウドデータ（全削除時の合計計算用）
 * @returns 削除確認に必要な情報
 */
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

    // 全削除センチネル（path === "*"）
    if ("path" in item && (item as CloudDirectoryNode).path === "*") {
      return "全てのゲームのクラウドデータを完全に削除しますか？";
    }

    return `「${item.name}」のクラウドデータをすべて削除しますか？`;
  }, [item]);

  const fileCountMessage = useMemo((): string => {
    if (!item) {
      return "0 個のファイルが削除されます";
    }

    // 全削除センチネル
    if ("path" in item && (item as CloudDirectoryNode).path === "*") {
      const totalFiles = cloudData.reduce((sum, cloudItem) => sum + cloudItem.fileCount, 0);
      return `全ての ${totalFiles} 個のファイルが削除されます`;
    }

    if ("fileCount" in item) {
      return `${item.fileCount} 個のファイルが削除されます`;
    }

    // CloudDirectoryNode（ツリービューからゲームノードを選択した場合）
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

    // 全削除センチネルにはサブテキスト不要
    if ("path" in item && (item as CloudDirectoryNode).path === "*") {
      return undefined;
    }

    // remotePath（CloudDataItem）またはpath（CloudDirectoryNode）を表示
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
