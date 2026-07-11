/**
 * @fileoverview 認証情報関連 atoms
 *
 * 有効なクレデンシャルが設定済みかのフラグなど。
 */

import { atom } from "jotai";

export const isValidCredsAtom = atom(false);
