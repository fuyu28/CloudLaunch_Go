/**
 * @fileoverview 「最新のリクエストのみ反映」パターン用フック
 *
 * 非同期処理が並行で走ったときに、古い応答の結果で新しい応答（および新しい状態）を上書き
 * してしまわないようにするための monotonic counter。
 *
 * 使い方:
 * ```ts
 * const { next, isLatest, reset } = useLatestRequestId();
 * const doFetch = async () => {
 *   const id = next();
 *   const data = await api.fetch();
 *   if (!isLatest(id)) return; // 古い応答は捨てる
 *   setState(data);
 * };
 * ```
 */

import { useCallback, useMemo, useRef } from "react";

export type UseLatestRequestIdReturn = {
  /** 新しいリクエストIDを発行してカウンタを進める。 */
  next: () => number;
  /** 指定 ID が現在の最新であれば true。 */
  isLatest: (id: number) => boolean;
  /** カウンタを 0 にリセットする（モーダルを閉じたときなど）。 */
  reset: () => void;
};

/**
 * monotonic counter を返すフック。
 * 主にモーダルや検索など「並行に投げうるリクエストのうち最新1件だけを反映したい」箇所で使う。
 *
 * 戻り値オブジェクトは useMemo で安定させる。毎レンダー新規オブジェクトを返すと、
 * `useEffect(..., [request])` かつ閉じているときに setState する呼び出し元で
 * Maximum update depth exceeded になる。
 */
export function useLatestRequestId(): UseLatestRequestIdReturn {
  const ref = useRef<number>(0);

  const next = useCallback((): number => {
    ref.current += 1;
    return ref.current;
  }, []);

  const isLatest = useCallback((id: number): boolean => ref.current === id, []);

  const reset = useCallback((): void => {
    ref.current = 0;
  }, []);

  return useMemo(() => ({ next, isLatest, reset }), [next, isLatest, reset]);
}
