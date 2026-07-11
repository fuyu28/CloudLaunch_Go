/**
 * @fileoverview ゲーム一覧グリッドコンポーネント
 *
 * このコンポーネントは、ゲームカードをグリッド形式で表示します。
 */

import { memo } from "react";

import GameCard from "./GameCard";
import type { GameType } from "src/types/game";

type GameGridProps = {
  games: GameType[];
  onLaunchGame: (game: GameType) => void;
  /** 起動警告が必要なゲームID一覧 */
  warningGameIds?: ReadonlySet<string>;
};

const GameGrid = memo(function GameGrid({
  games,
  onLaunchGame,
  warningGameIds,
}: GameGridProps): React.JSX.Element {
  if (games.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center min-h-0">
        <div className="text-center">
          <p className="text-lg font-medium text-base-content/80 mb-2">
            ゲームが見つかりませんでした
          </p>
          <p className="text-sm text-base-content/60">
            検索条件を変更するか、新しいゲームを追加してください
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-auto scrollbar-thin scrollbar-thumb-base-content/30 scrollbar-track-transparent min-h-0">
      <div className="relative">
        <div
          className="grid gap-5 justify-center px-6 pb-6"
          style={{ gridTemplateColumns: "repeat(auto-fill, 220px)" }}
        >
          {games.map((game) => (
            <GameCard
              key={game.id}
              game={game}
              onLaunchGame={onLaunchGame}
              hasLaunchWarning={warningGameIds?.has(game.id) ?? false}
            />
          ))}
        </div>
      </div>
    </div>
  );
});

export default GameGrid;
