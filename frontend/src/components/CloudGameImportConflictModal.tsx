/**
 * @fileoverview クラウドゲームインポートの競合モーダル
 */

import { BaseModal } from "./BaseModal";
import type { CloudGameMetadata } from "src/types/cloud";
import type { GameType } from "src/types/game";

type CloudGameImportConflictModalProps = {
  isOpen: boolean;
  onClose: () => void;
  cloudGame: CloudGameMetadata | null;
  localMatches: GameType[];
  onImportDuplicate: () => void;
  onReplaceLocal: () => void;
};

export default function CloudGameImportConflictModal({
  isOpen,
  onClose,
  cloudGame,
  localMatches,
  onImportDuplicate,
  onReplaceLocal,
}: CloudGameImportConflictModalProps): React.JSX.Element {
  if (!cloudGame) {
    return (
      <BaseModal id="cloud-game-conflict-modal" isOpen={false} onClose={onClose}>
        <div />
      </BaseModal>
    );
  }

  return (
    <BaseModal
      id="cloud-game-conflict-modal"
      isOpen={isOpen}
      onClose={onClose}
      title="タイトルの競合が見つかりました"
      size="md"
      footer={
        <div className="flex flex-wrap justify-end gap-2">
          <button type="button" className="btn" onClick={onClose}>
            キャンセル
          </button>
          <button type="button" className="btn btn-outline" onClick={onImportDuplicate}>
            重複で追加
          </button>
          <button type="button" className="btn btn-error" onClick={onReplaceLocal}>
            ローカルを削除して追加
          </button>
        </div>
      }
    >
      <div className="space-y-3">
        <p className="text-sm text-base-content/70">
          クラウド側の「{cloudGame.title}」と同名のゲームがローカルに存在します。
        </p>
        <div className="rounded-lg border border-base-300 bg-base-100 p-3">
          <div className="text-xs text-base-content/60 mb-2">競合しているローカルゲーム</div>
          <ul className="space-y-1 text-sm">
            {localMatches.map((game) => (
              <li key={game.id} className="flex items-center justify-between gap-2">
                <span className="font-medium">{game.title}</span>
                <span className="text-base-content/60">{game.publisher}</span>
              </li>
            ))}
          </ul>
        </div>
        <p className="text-xs text-warning">
          「ローカルを削除して追加」を選ぶと、上記のローカルゲームが削除されます。
        </p>
      </div>
    </BaseModal>
  );
}
