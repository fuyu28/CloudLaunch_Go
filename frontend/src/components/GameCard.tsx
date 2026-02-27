import { memo, useCallback } from "react";
import { IoIosPlay } from "react-icons/io";
import { Link } from "react-router-dom";

import DynamicImage from "./DynamicImage";
import type { GameType } from "src/types/game";

type GameCardProps = {
  game: GameType;
  onLaunchGame: (game: GameType) => void;
  hasLaunchWarning?: boolean;
};

const GameCard = memo(function GameCard({
  game,
  onLaunchGame,
  hasLaunchWarning = false,
}: GameCardProps): React.JSX.Element {
  const handleLaunchClick = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      onLaunchGame(game);
    },
    [game, onLaunchGame],
  );

  return (
    <div
      className="
        bg-base-100 rounded-xl overflow-hidden
        shadow-lg transform transition
        hover:shadow-xl
      "
    >
      <Link to={`/game/${game.id}`}>
        <div className="group relative h-40 w-full bg-base-200">
          <DynamicImage
            src={game.imagePath || ""}
            alt={game.title}
            className="h-full w-full object-cover"
            loading="lazy"
          />
          <div
            className="
            absolute inset-0
            flex items-center justify-center
            opacity-0 group-hover:opacity-100
            transition-opacity
          "
          >
            <button
              type="button"
              onClick={handleLaunchClick}
              aria-label="ゲームを起動"
              title={
                hasLaunchWarning ? "実行ファイルまたはセーブ保存先に問題があります" : "ゲームを起動"
              }
              className="bg-base-100/80
              rounded-full p-2 shadow-md
              flex items-center justify-center
              hover:bg-base-100/90 focus:outline-none
              focus:ring-2 focus:ring-primary
              transition relative"
            >
              <IoIosPlay size={32} className="pl-1 text-base-content" />
              {hasLaunchWarning && (
                <span className="absolute -top-1 -right-1 h-5 min-w-5 rounded-full bg-error text-error-content text-xs font-bold leading-5 text-center px-1">
                  !
                </span>
              )}
            </button>
          </div>
        </div>
        <div className="p-2 h-20">
          <h3 className="text-base font-semibold line-clamp-2">{game.title}</h3>
          <p className="text-sm text-base-content line-clamp-2">{game.publisher}</p>
        </div>
      </Link>
    </div>
  );
});

export default GameCard;
