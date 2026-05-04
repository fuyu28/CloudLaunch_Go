/**
 * @fileoverview プレイステータス表示コンポーネント
 *
 * このコンポーネントは、ゲームのプレイステータスを表示するUIを提供します。
 *
 * 主な機能：
 * - プレイステータスの視覚的表示（バッジ形式）
 * - 各ステータスに応じた色分け
 *
 * 使用例：
 * ```tsx
 * <PlayStatusSelector currentStatus={game.playStatus} />
 * ```
 */

import { FaGamepad, FaPlay, FaCheck } from "react-icons/fa";

/**
 * プレイステータスの型定義
 */
export type PlayStatus = "unplayed" | "playing" | "played";

/**
 * プレイステータス表示コンポーネントのprops
 */
export type PlayStatusSelectorProps = {
  /** 現在のプレイステータス */
  currentStatus: PlayStatus;
};

/**
 * プレイステータス情報の設定
 */
const STATUS_CONFIG = {
  unplayed: {
    label: "未プレイ",
    icon: FaGamepad,
    badgeClass: "badge-neutral",
    description: "まだプレイしていない",
  },
  playing: {
    label: "プレイ中",
    icon: FaPlay,
    badgeClass: "badge-primary",
    description: "現在プレイしている",
  },
  played: {
    label: "クリア済み",
    icon: FaCheck,
    badgeClass: "badge-success",
    description: "クリア済み",
  },
} as const;

/**
 * プレイステータス表示コンポーネント
 *
 * @param props コンポーネントのprops
 * @returns プレイステータス表示要素
 */
export function PlayStatusSelector({ currentStatus }: PlayStatusSelectorProps): React.JSX.Element {
  const currentConfig = STATUS_CONFIG[currentStatus];

  return (
    <div className="flex items-center gap-2">
      <span className="text-sm font-medium">プレイステータス:</span>
      <div className={`badge ${currentConfig.badgeClass} gap-2`}>
        <currentConfig.icon className="w-3 h-3" />
        {currentConfig.label}
      </div>
    </div>
  );
}

export default PlayStatusSelector;
