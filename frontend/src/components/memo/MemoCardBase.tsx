/**
 * @fileoverview メモカード基本コンポーネント
 *
 * メモカードの基本構造を提供する共通コンポーネントです。
 * MemoCardとMemoListで共通使用されます。
 */

import { memo, useMemo, useCallback } from "react";
import { FaGamepad } from "react-icons/fa";

import { useTimeFormat } from "@renderer/hooks/useTimeFormat";

import MemoDropdownMenu from "./MemoDropdownMenu";
import type { MemoType } from "src/types/memo";

type MemoCardBaseProps = {
  memo: MemoType;
  onClick: (memoId: string) => void;
  isDropdownOpen: boolean;
  onDropdownToggle: (memoId: string, event: React.MouseEvent) => void;
  onEdit: (memoId: string, event: React.MouseEvent) => void;
  onDelete: (memoId: string, event: React.MouseEvent) => void;
  onSyncFromCloud?: (event: React.MouseEvent) => void;
  className?: string;
  titleMaxLength?: number;
  contentMaxLength?: number;
  showGameTitle?: boolean;
  dropdownPosition?: string;
};

function MemoCardBase({
  memo,
  onClick,
  isDropdownOpen,
  onDropdownToggle,
  onEdit,
  onDelete,
  onSyncFromCloud,
  className = "border border-base-300 rounded-lg p-3",
  titleMaxLength,
  contentMaxLength = 80,
  showGameTitle = true,
  dropdownPosition = "absolute top-2 right-2",
}: MemoCardBaseProps): React.JSX.Element {
  const { formatDateWithTime } = useTimeFormat();

  const truncatedTexts = useMemo(() => {
    const truncatedTitle =
      titleMaxLength && memo.title.length > titleMaxLength
        ? `${memo.title.substring(0, titleMaxLength)}...`
        : memo.title;

    const truncatedContent =
      memo.content.length > contentMaxLength
        ? `${memo.content.substring(0, contentMaxLength)}...`
        : memo.content;

    return { truncatedTitle, truncatedContent };
  }, [memo.title, memo.content, titleMaxLength, contentMaxLength]);

  const formattedDate = useMemo(() => {
    return formatDateWithTime(memo.updatedAt);
  }, [memo.updatedAt, formatDateWithTime]);

  const handleCardClick = useCallback(() => {
    onClick(memo.id);
  }, [onClick, memo.id]);

  const cardClassName = useMemo(() => {
    return `${className} cursor-pointer hover:bg-base-200 transition-colors duration-200 relative`;
  }, [className]);

  return (
    <div className={cardClassName} onClick={handleCardClick}>
      <h3 className="font-semibold text-sm truncate mb-1 pr-8">{truncatedTexts.truncatedTitle}</h3>

      {showGameTitle && memo.gameTitle && (
        <div className="flex items-center gap-2 text-xs text-base-content/60 mb-2">
          <FaGamepad className="text-xs flex-shrink-0" />
          <span className="truncate">{memo.gameTitle}</span>
        </div>
      )}

      <p className="text-xs text-base-content/60 line-clamp-3 mb-2 leading-relaxed">
        {truncatedTexts.truncatedContent}
      </p>

      <div className="flex justify-between items-center mt-auto">
        <span className="text-xs text-base-content/50 font-medium">{formattedDate}</span>
        <span className="text-xs text-base-content/40">{memo.content.length}文字</span>
      </div>

      <MemoDropdownMenu
        memoId={memo.id}
        isOpen={isDropdownOpen}
        onToggle={onDropdownToggle}
        onEdit={onEdit}
        onDelete={onDelete}
        onSyncFromCloud={onSyncFromCloud}
        className={dropdownPosition}
      />
    </div>
  );
}

// ハンドラ参照は親が毎レンダー新規矢印を渡すので比較しない（入れると memo が常に無効）。
export default memo(MemoCardBase, (prevProps, nextProps) => {
  return (
    prevProps.memo.id === nextProps.memo.id &&
    prevProps.memo.title === nextProps.memo.title &&
    prevProps.memo.content === nextProps.memo.content &&
    new Date(prevProps.memo.updatedAt).getTime() === new Date(nextProps.memo.updatedAt).getTime() &&
    prevProps.memo.gameTitle === nextProps.memo.gameTitle &&
    prevProps.isDropdownOpen === nextProps.isDropdownOpen &&
    prevProps.className === nextProps.className &&
    prevProps.titleMaxLength === nextProps.titleMaxLength &&
    prevProps.contentMaxLength === nextProps.contentMaxLength &&
    prevProps.showGameTitle === nextProps.showGameTitle
  );
});
