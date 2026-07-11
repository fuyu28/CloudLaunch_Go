/**
 * @fileoverview 認証情報関連 atoms
 *
 * 有効なクレデンシャルが設定済みかのフラグなど。
 */

import { atom } from "jotai";

// 有効なクレデンシャルが設定されているかのフラグ
export const isValidCredsAtom = atom(false);
