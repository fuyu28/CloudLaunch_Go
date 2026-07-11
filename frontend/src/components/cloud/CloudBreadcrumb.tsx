/**
 * @fileoverview クラウドパンくずリストコンポーネント
 *
 * このコンポーネントは、クラウドデータ管理画面でのナビゲーション用
 * パンくずリストを提供します。
 */

import React from "react";
import { FiHome, FiChevronRight, FiArrowLeft } from "react-icons/fi";

import type { CloudPathSegment } from "@renderer/utils/cloudUtils";

type CloudBreadcrumbProps = {
  currentPath: CloudPathSegment[];
  onNavigateToPath: (path: CloudPathSegment[]) => void;
  onNavigateBack: () => void;
};

export function CloudBreadcrumb({
  currentPath,
  onNavigateToPath,
  onNavigateBack,
}: CloudBreadcrumbProps): React.JSX.Element | null {
  if (currentPath.length === 0) {
    return null;
  }

  return (
    <div className="flex items-center gap-2 mb-4 p-3 bg-base-200 rounded-lg">
      <button
        onClick={() => onNavigateToPath([])}
        className="btn btn-sm btn-ghost"
        title="ルートに戻る"
      >
        <FiHome className="text-sm" />
      </button>

      <FiChevronRight className="text-base-content/50" />

      {currentPath.map((segment, index) => (
        <React.Fragment key={`${segment.id}-${index}`}>
          <button
            onClick={() => {
              const newPath = currentPath.slice(0, index + 1);
              onNavigateToPath(newPath);
            }}
            className="btn btn-sm btn-ghost text-sm"
          >
            {segment.name}
          </button>
          {index < currentPath.length - 1 && <FiChevronRight className="text-base-content/50" />}
        </React.Fragment>
      ))}

      <div className="ml-auto">
        <button
          onClick={onNavigateBack}
          className="btn btn-sm btn-ghost"
          title="一つ上のディレクトリに戻る"
        >
          <FiArrowLeft className="text-sm mr-1" />
          戻る
        </button>
      </div>
    </div>
  );
}
