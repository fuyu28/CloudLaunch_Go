/**
 * @fileoverview クラウドディレクトリツリーノードコンポーネント
 *
 * ツリービューでのディレクトリ・ファイル表示とインタラクション機能を提供します。
 */

import { FiFolder, FiFile, FiTrash2, FiChevronRight, FiChevronDown } from "react-icons/fi";

import type { CloudDirectoryNode } from "src/types/cloud";
import {
  formatFileSize,
  formatDate,
  countFilesRecursively,
  isCloudNodeLoaded,
  sumSizesRecursively,
  latestModifiedRecursively,
} from "@renderer/utils/cloudUtils";

/**
 * ツリーノードコンポーネントのプロパティ
 */
type CloudTreeNodeProps = {
  node: CloudDirectoryNode;
  level: number;
  expandedNodes: Set<string>;
  /** ファイル一覧を遅延取得中のゲームID集合 */
  loadingGameIds?: Set<string>;
  onToggleExpand: (path: string) => void;
  onDelete: (node: CloudDirectoryNode) => void;
  onSelect: (node: CloudDirectoryNode) => void;
};

/**
 * クラウドディレクトリツリーノードコンポーネント
 */
export default function CloudTreeNode({
  node,
  level,
  expandedNodes,
  loadingGameIds,
  onToggleExpand,
  onDelete,
  onSelect,
}: CloudTreeNodeProps): React.JSX.Element {
  const isExpanded = expandedNodes.has(node.path);
  const hasChildren = node.children && node.children.length > 0;
  // ゲーム（トップレベルのディレクトリ）はファイル一覧を遅延取得するため、
  // 未取得（children が undefined）のあいだは数値を「—」で表示し、展開で取得を促す。
  const isLoaded = isCloudNodeLoaded(node);
  const isLoading = loadingGameIds?.has(node.path) ?? false;
  // 未取得のゲームでも展開ボタンを表示してファイル取得をトリガーできるようにする。
  const isExpandable = node.isDirectory && (hasChildren || !isLoaded);
  const displaySize = node.isDirectory ? sumSizesRecursively(node) : node.size;
  const displayLastModified = node.isDirectory
    ? latestModifiedRecursively(node)
    : node.lastModified;

  return (
    <>
      <div
        className={`flex items-center gap-2 px-3 py-2 hover:bg-base-200 cursor-pointer rounded-md ${
          level > 0 ? "ml-" + level * 4 : ""
        }`}
        style={{ paddingLeft: `${level * 1.5 + 0.75}rem` }}
      >
        {/* 展開/折りたたみボタン */}
        <button
          onClick={() => isExpandable && onToggleExpand(node.path)}
          className={`w-4 h-4 flex items-center justify-center ${!isExpandable ? "invisible" : ""}`}
        >
          {isExpandable &&
            (isExpanded ? (
              <FiChevronDown className="text-xs" />
            ) : (
              <FiChevronRight className="text-xs" />
            ))}
        </button>

        {/* アイコン */}
        <div className="flex-shrink-0">
          {node.isDirectory ? (
            <FiFolder className="text-primary" />
          ) : (
            <FiFile className="text-base-content/60" />
          )}
        </div>

        {/* ファイル/フォルダ名 */}
        <div
          className="flex-1 min-w-0 flex items-center justify-between group"
          onClick={() => onSelect(node)}
        >
          <div className="flex-1 min-w-0">
            <div className="truncate font-medium text-sm" title={node.name}>
              {node.name}
            </div>
            <div className="text-xs text-base-content/60">
              {isLoaded ? formatFileSize(displaySize) : "—"} • {formatDate(displayLastModified)}
              {node.isDirectory && (
                <span className="ml-2">
                  ({isLoaded ? `${countFilesRecursively(node)} ファイル` : "— ファイル"})
                </span>
              )}
            </div>
          </div>

          {/* 削除ボタン：トップレベル（level === 0）のゲームノードのみ表示。
              深度で判定することで、path にスラッシュが含まれるかどうかに依存せず
              レンダリング構造から明確にゲーム単位を識別できる。 */}
          {level === 0 && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onDelete(node);
              }}
              className="btn btn-sm btn-ghost btn-error opacity-0 group-hover:opacity-100 transition-opacity ml-2"
              title={`${node.name} のクラウドデータを削除`}
            >
              <FiTrash2 className="text-xs" />
            </button>
          )}
        </div>
      </div>

      {/* 子ノード */}
      {isExpanded && hasChildren && (
        <div>
          {node.children!.map((child, index) => (
            <CloudTreeNode
              key={`${child.path}-${index}`}
              node={child}
              level={level + 1}
              expandedNodes={expandedNodes}
              loadingGameIds={loadingGameIds}
              onToggleExpand={onToggleExpand}
              onDelete={onDelete}
              onSelect={onSelect}
            />
          ))}
        </div>
      )}

      {/* 未取得ゲームを展開した直後はファイル一覧を遅延取得中 */}
      {isExpanded && !isLoaded && (
        <div
          className="flex items-center gap-2 px-3 py-2 text-xs text-base-content/60"
          style={{ paddingLeft: `${(level + 1) * 1.5 + 0.75}rem` }}
        >
          <span className="loading loading-spinner loading-xs"></span>
          {isLoading ? "読み込み中..." : "ファイル一覧を取得します"}
        </div>
      )}
    </>
  );
}
