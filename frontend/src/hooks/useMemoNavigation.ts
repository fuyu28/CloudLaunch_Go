/**
 * @fileoverview メモナビゲーションフック
 *
 * メモページ間のナビゲーション処理を統一します。
 * クエリパラメータによる適切な戻り先の判定と遷移を提供します。
 */

import { useNavigate, useSearchParams } from "react-router-dom";

type UseMemoNavigationReturn = {
  handleBack: () => void;
  handleSaveSuccess: (effectiveGameId: string, mode: "create" | "edit", memoId?: string) => void;
  searchParams: URLSearchParams;
  isFromGame: boolean;
  gameIdParam: string | null;
};

export function useMemoNavigation(): UseMemoNavigationReturn {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  const fromParam = searchParams.get("from");
  const gameIdParam = searchParams.get("gameId");
  const isFromGame = fromParam === "game" && gameIdParam !== null;

  const handleBack = (): void => {
    if (isFromGame && gameIdParam) {
      // history.back だとメモ一覧等に飛ばず、ゲーム詳細の文脈を失うことがあるため明示遷移する。
      navigate(`/game/${gameIdParam}`);
    } else {
      navigate(-1);
    }
  };

  const handleSaveSuccess = (
    effectiveGameId: string,
    mode: "create" | "edit",
    memoId?: string,
  ): void => {
    if (mode === "create") {
      if (isFromGame && gameIdParam) {
        // ゲーム詳細から新規作成したときは一覧ではなく元の詳細へ戻す。
        navigate(`/game/${gameIdParam}`);
      } else {
        navigate(`/memo/list/${effectiveGameId}`);
      }
    } else if (isFromGame && gameIdParam && memoId) {
      // from=game を付けないと閲覧ページの「戻る」がゲーム文脈を捨てる。
      navigate(`/memo/view/${memoId}?from=game&gameId=${gameIdParam}`);
    } else {
      navigate(-1);
    }
  };

  return {
    handleBack,
    handleSaveSuccess,
    searchParams,
    isFromGame,
    gameIdParam,
  };
}
