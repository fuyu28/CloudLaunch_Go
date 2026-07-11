/**
 * @fileoverview クラウドページヘッダーコンポーネント
 *
 * このコンポーネントは、クラウドデータ管理ページのヘッダー部分を
 * 担当し、ビュー切り替えや操作ボタンを提供します。
 */

import { FiTrash2, FiRefreshCw, FiCloud, FiFolder, FiFolderPlus } from "react-icons/fi";

import type { CloudDataItem, CloudDirectoryNode } from "src/types/cloud";

export type ViewMode = "cards" | "tree";

/**
 * クラウドヘッダーのプロパティ
 */
type CloudHeaderProps = {
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  cloudData: CloudDataItem[];
  directoryTree: CloudDirectoryNode[];
  loading: boolean;
  onRefresh: () => void;
  onDeleteAll: () => void;
};

export function CloudHeader({
  viewMode,
  onViewModeChange,
  cloudData,
  directoryTree,
  loading,
  onRefresh,
  onDeleteAll,
}: CloudHeaderProps): React.JSX.Element {
  const hasData = cloudData.length > 0 || directoryTree.length > 0;

  return (
    <div className="flex items-center justify-between mb-6">
      <div className="flex items-center gap-3">
        <FiCloud className="text-3xl text-primary" />
        <div>
          <h1 className="text-2xl font-bold text-base-content">クラウドデータ管理</h1>
          <p className="text-base-content/80">クラウドストレージ上のゲームデータを管理できます</p>
        </div>
      </div>

      <div className="flex items-center gap-3">
        <div className="join">
          <button
            className={`btn join-item btn-sm ${viewMode === "cards" ? "btn-active" : ""}`}
            onClick={() => onViewModeChange("cards")}
          >
            <FiFolder className="mr-1" />
            カード
          </button>
          <button
            className={`btn join-item btn-sm ${viewMode === "tree" ? "btn-active" : ""}`}
            onClick={() => onViewModeChange("tree")}
          >
            <FiFolderPlus className="mr-1" />
            ツリー
          </button>
        </div>

        {hasData && (
          <button onClick={onDeleteAll} className="btn btn-error btn-sm gap-2" disabled={loading}>
            <FiTrash2 />
            全て削除
          </button>
        )}

        <button onClick={onRefresh} disabled={loading} className="btn btn-primary gap-2">
          <FiRefreshCw className={loading ? "animate-spin" : ""} />
          更新
        </button>
      </div>
    </div>
  );
}
