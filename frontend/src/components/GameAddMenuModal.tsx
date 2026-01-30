/**
 * @fileoverview ゲーム追加メニューのモーダル
 *
 * 新規登録とクラウドからの追加を選択するためのモーダル。
 */

import { BaseModal } from "./BaseModal";

type GameAddMenuModalProps = {
  isOpen: boolean;
  onClose: () => void;
  onSelectNew: () => void;
  onSelectCloud: () => void;
};

export default function GameAddMenuModal({
  isOpen,
  onClose,
  onSelectNew,
  onSelectCloud,
}: GameAddMenuModalProps): React.JSX.Element {
  return (
    <BaseModal
      id="game-add-menu-modal"
      isOpen={isOpen}
      onClose={onClose}
      title="ゲームの追加"
      size="md"
      footer={
        <button type="button" className="btn" onClick={onClose}>
          閉じる
        </button>
      }
    >
      <div className="space-y-4">
        <button type="button" className="btn btn-primary w-full" onClick={onSelectNew}>
          新規ゲームを登録する
        </button>
        <button type="button" className="btn btn-outline w-full" onClick={onSelectCloud}>
          クラウドから既存ゲームを追加
        </button>
      </div>
    </BaseModal>
  );
}
