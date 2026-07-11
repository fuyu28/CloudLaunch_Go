/**
 * @fileoverview メモ作成・編集ページ
 *
 * 新しいメモの作成と既存メモの編集を行うページです。
 * @uiw/react-md-editorを使用してmarkdownでメモを作成・編集できます。
 */

import { useParams } from "react-router-dom";

import MemoForm from "@renderer/components/memo/MemoForm";

import { useMemoNavigation } from "@renderer/hooks/useMemoNavigation";

export default function MemoEditor(): React.JSX.Element {
  const { gameId, memoId } = useParams<{ gameId?: string; memoId?: string }>();
  const { handleBack, handleSaveSuccess } = useMemoNavigation();

  const mode = memoId ? "edit" : "create";
  const pageTitle = mode === "edit" ? "メモを編集" : "新しいメモ";

  // memoId 単位で MemoForm をリマウントすることで内部状態（title/content/isInitializedRef 等）を初期化する。
  // これにより /memo/edit/A → /memo/edit/B の遷移でも新しい memoId のデータを確実に再フェッチできる。
  return (
    <MemoForm
      key={memoId ?? "create"}
      mode={mode}
      memoId={memoId}
      preSelectedGameId={gameId}
      showGameSelector={false}
      pageTitle={pageTitle}
      backTo={handleBack}
      onSaveSuccess={(effectiveGameId) => handleSaveSuccess(effectiveGameId, mode, memoId)}
    />
  );
}
