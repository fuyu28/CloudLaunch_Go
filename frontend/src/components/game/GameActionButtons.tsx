/**
 * @fileoverview ゲーム基本操作ボタンコンポーネント
 *
 * このコンポーネントは、ゲーム詳細ページで使用される基本的な操作ボタン群を提供します。
 */

import { FaTrash } from "react-icons/fa";
import { IoIosPlay } from "react-icons/io";
import { MdEdit } from "react-icons/md";

export type GameActionButtonsProps = {
  gameId: string;
  saveDataFolderPath?: string;
  onLaunchGame: () => void;
  onEditGame: () => void;
  onDeleteGame: () => void;
  isLaunching?: boolean;
};

export function GameActionButtons({
  onLaunchGame,
  onEditGame,
  onDeleteGame,
  isLaunching,
}: GameActionButtonsProps): React.JSX.Element {
  return (
    <div className="space-y-3">
      <div className="flex gap-3">
        <button
          onClick={onLaunchGame}
          className="btn btn-primary btn-md flex-1"
          disabled={isLaunching}
        >
          <IoIosPlay size={24} />
          {isLaunching ? "起動中..." : "ゲームを起動"}
        </button>
        <button className="btn btn-outline btn-md flex-1" onClick={onEditGame}>
          <MdEdit /> 編集
        </button>
        <button className="btn btn-error btn-md flex-1" onClick={onDeleteGame}>
          <FaTrash /> 登録を解除
        </button>
      </div>
    </div>
  );
}

export default GameActionButtons;
