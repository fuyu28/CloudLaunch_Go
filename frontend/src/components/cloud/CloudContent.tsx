/**
 * @fileoverview クラウドコンテンツ表示コンポーネント
 *
 * このコンポーネントは、クラウドデータの表示部分を担当し、
 * カードビューとツリービューの切り替えを提供します。
 */

import { FiCloud, FiFolder } from "react-icons/fi";

import type { ViewMode } from "./CloudHeader";
import { DirectoryNodeCard } from "./CloudItemCard";
import CloudTreeNode from "./CloudTreeNode";
import type { CloudDirectoryNode } from "src/types/cloud";
import type { CloudPathSegment } from "@renderer/utils/cloudUtils";

type CloudContentProps = {
  viewMode: ViewMode;
  loading: boolean;
  gameLoading?: boolean;
  loadingGameIds?: Set<string>;
  directoryTree: CloudDirectoryNode[];
  currentPath: CloudPathSegment[];
  currentDirectoryNodes: CloudDirectoryNode[];
  expandedNodes: Set<string>;
  onToggleExpand: (path: string) => void;
  onSelectNode: (node: CloudDirectoryNode) => void;
  onDelete: (item: CloudDirectoryNode) => void;
  onNavigateToDirectory: (node: CloudDirectoryNode) => void;
  onViewDetails: (item: CloudDirectoryNode) => void;
};

function EmptyState({
  icon: Icon,
  title,
  description,
}: {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  description: string;
}): React.JSX.Element {
  return (
    <div className="text-center py-12">
      <Icon className="text-6xl text-base-content/30 mx-auto mb-4" />
      <h3 className="text-xl font-medium text-base-content/70 mb-2">{title}</h3>
      <p className="text-base-content/50">{description}</p>
    </div>
  );
}

export function CloudContent({
  viewMode,
  loading,
  gameLoading = false,
  loadingGameIds,
  directoryTree,
  currentPath,
  currentDirectoryNodes,
  expandedNodes,
  onToggleExpand,
  onSelectNode,
  onDelete,
  onNavigateToDirectory,
  onViewDetails,
}: CloudContentProps): React.JSX.Element {
  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <div className="loading loading-spinner loading-lg"></div>
      </div>
    );
  }

  if (viewMode === "cards") {
    return (
      <div>
        {currentPath.length === 0 ? (
          // ルート（ゲーム単位）だけ削除可。配下ファイルには onDelete を渡さない。
          directoryTree.length === 0 ? (
            <EmptyState
              icon={FiCloud}
              title="クラウドデータがありません"
              description="ゲームのセーブデータをアップロードすると、ここに表示されます"
            />
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {directoryTree.map((node, index) => (
                <DirectoryNodeCard
                  key={`${node.path}-${index}`}
                  node={node}
                  onDelete={() => onDelete(node)}
                  onViewDetails={onViewDetails}
                  onNavigate={node.isDirectory ? () => onNavigateToDirectory(node) : undefined}
                />
              ))}
            </div>
          )
        ) : gameLoading ? (
          <div className="flex justify-center py-12">
            <div className="loading loading-spinner loading-lg"></div>
          </div>
        ) : // サブディレクトリ（セーブファイル階層）- onDelete を渡さず削除ボタンを非表示
        currentDirectoryNodes.length === 0 ? (
          <EmptyState
            icon={FiFolder}
            title="このディレクトリは空です"
            description="ファイルやサブディレクトリがありません"
          />
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {currentDirectoryNodes.map((node, index) => (
              <DirectoryNodeCard
                key={`${node.path}-${index}`}
                node={node}
                onNavigate={node.isDirectory ? () => onNavigateToDirectory(node) : undefined}
                onViewDetails={onViewDetails}
                // onDelete は渡さない：サブノードの個別削除は履歴破壊になるため不可
              />
            ))}
          </div>
        )}
      </div>
    );
  }

  return (
    <div className="bg-base-100 rounded-lg border border-base-300 p-4">
      {directoryTree.length === 0 ? (
        <EmptyState
          icon={FiCloud}
          title="クラウドデータがありません"
          description="ゲームのセーブデータをアップロードすると、ここに表示されます"
        />
      ) : (
        <div className="space-y-1">
          {directoryTree.map((node, index) => (
            <div key={`${node.path}-${index}`} className="group">
              <CloudTreeNode
                node={node}
                level={0}
                expandedNodes={expandedNodes}
                loadingGameIds={loadingGameIds}
                onToggleExpand={onToggleExpand}
                onDelete={onDelete}
                onSelect={onSelectNode}
              />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
