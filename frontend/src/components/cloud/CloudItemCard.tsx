/**
 * @fileoverview クラウドアイテムカードコンポーネント
 *
 * カードビューでのクラウドデータアイテム表示とアクション機能を提供します。
 */

import { FiFolder, FiFile, FiTrash2 } from "react-icons/fi";

import type { CloudDataItem, CloudDirectoryNode } from "src/types/cloud";
import {
  formatFileSize,
  formatDate,
  countFilesRecursively,
  isCloudNodeLoaded,
  sumSizesRecursively,
} from "@renderer/utils/cloudUtils";

/**
 * クラウドデータアイテムカードのプロパティ
 */
type CloudItemCardProps = {
  item: CloudDataItem;
  onDelete: (item: CloudDataItem) => void;
  onViewDetails: (item: CloudDataItem) => void;
  onNavigate?: (directoryName: string) => void;
};

/**
 * クラウドデータアイテムカードコンポーネント
 */
export function CloudItemCard({
  item,
  onDelete,
  onViewDetails,
  onNavigate,
}: CloudItemCardProps): React.JSX.Element {
  const handleClick = (): void => {
    if (onNavigate) {
      onNavigate(item.name);
    }
  };

  return (
    <div
      className={`bg-base-100 rounded-lg shadow-md hover:shadow-lg transition-shadow p-4 border border-base-300 ${
        onNavigate ? "cursor-pointer" : ""
      }`}
      onClick={handleClick}
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <FiFolder className="text-2xl text-primary flex-shrink-0" />
          <div className="flex-1 min-w-0">
            <h3 className="font-medium text-base-content truncate" title={item.name}>
              {item.name}
            </h3>
            <div className="text-sm text-base-content/70 space-y-1">
              <div className="flex items-center gap-2">
                <FiFile className="text-xs" />
                <span>{item.fileCount} ファイル</span>
              </div>
              <div>{formatFileSize(item.totalSize)}</div>
            </div>
          </div>
        </div>

        <div className="flex gap-2 flex-shrink-0">
          <button
            onClick={(e) => {
              e.stopPropagation();
              onViewDetails(item);
            }}
            className="btn btn-sm btn-ghost tooltip"
            data-tip="詳細表示"
          >
            <FiFile className="text-base" />
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              onDelete(item);
            }}
            className="btn btn-sm btn-error btn-ghost tooltip"
            data-tip="削除"
          >
            <FiTrash2 className="text-base" />
          </button>
        </div>
      </div>

      <div className="text-xs text-base-content/60">最終更新: {formatDate(item.lastModified)}</div>
    </div>
  );
}

/**
 * ディレクトリノードカードのプロパティ
 */
type DirectoryNodeCardProps = {
  node: CloudDirectoryNode;
  onNavigate?: (directoryName: string) => void;
  /**
   * 削除ボタンを表示するかどうか。
   * ゲーム単位削除のみ許可するため、ルートレベル（isGameNode=true）のカードにのみ渡す。
   * サブディレクトリ・ファイルカードでは undefined のまま（ボタンを表示しない）。
   */
  onDelete?: (node: CloudDirectoryNode) => void;
  onViewDetails?: (node: CloudDirectoryNode) => void;
};

/**
 * ディレクトリノードカードコンポーネント
 */
export function DirectoryNodeCard({
  node,
  onNavigate,
  onDelete,
  onViewDetails,
}: DirectoryNodeCardProps): React.JSX.Element {
  const handleClick = (): void => {
    if (node.isDirectory && onNavigate) {
      onNavigate(node.name);
    }
  };

  // children を取得済みなら配下を集計、未取得なら commit メタ由来の
  // node.fileCount / node.size（=サマリの fileCount / totalSize）を使う。
  // ロード済みなら 0 件でも「0 ファイル / 0 B」を出し、未取得かつサマリも空のとき
  // （旧 commit など）だけ非表示にする。これでロード済み空ディレクトリと未取得を
  // 表示で区別する。
  const childrenLoaded = isCloudNodeLoaded(node);
  const displayCount = childrenLoaded ? countFilesRecursively(node) : (node.fileCount ?? 0);
  const displaySize = node.isDirectory && childrenLoaded ? sumSizesRecursively(node) : node.size;
  const hasMetrics = !node.isDirectory || childrenLoaded || displayCount > 0;

  return (
    <div
      className={`bg-base-100 rounded-lg shadow-md hover:shadow-lg transition-shadow p-4 border border-base-300 ${
        node.isDirectory && onNavigate ? "cursor-pointer" : ""
      }`}
      onClick={handleClick}
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3 flex-1 min-w-0">
          {node.isDirectory ? (
            <FiFolder className="text-2xl text-primary flex-shrink-0" />
          ) : (
            <FiFile className="text-2xl text-base-content/60 flex-shrink-0" />
          )}
          <div className="flex-1 min-w-0">
            <h3 className="font-medium text-base-content truncate" title={node.name}>
              {node.name}
            </h3>
            {hasMetrics && (
              <div className="text-sm text-base-content/70 space-y-1">
                {node.isDirectory && (
                  <div className="flex items-center gap-2">
                    <FiFile className="text-xs" />
                    <span>{displayCount} ファイル</span>
                  </div>
                )}
                <div>{formatFileSize(displaySize)}</div>
              </div>
            )}
          </div>
        </div>

        <div className="flex gap-2 flex-shrink-0">
          {onViewDetails && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onViewDetails(node);
              }}
              className="btn btn-sm btn-ghost tooltip"
              data-tip="詳細表示"
            >
              <FiFile className="text-base" />
            </button>
          )}
          {/* 削除ボタン：onDelete が渡された場合のみ表示。
              CloudContent はルートレベルのカードにのみ onDelete を渡すため、
              サブディレクトリ・ファイルカードには削除ボタンが表示されない。 */}
          {onDelete && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onDelete(node);
              }}
              className="btn btn-sm btn-error btn-ghost tooltip"
              data-tip="このゲームのクラウドデータを削除"
            >
              <FiTrash2 className="text-base" />
            </button>
          )}
        </div>
      </div>

      <div className="text-xs text-base-content/60">最終更新: {formatDate(node.lastModified)}</div>
    </div>
  );
}
