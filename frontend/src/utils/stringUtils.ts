/**
 * @fileoverview 文字列操作ユーティリティ
 */

/**
 * ゲームIDからリモートパスを生成
 * @param gameId - ゲームID
 * @returns リモートパス（games/{gameId}/save_data）
 */
export function createRemotePath(gameId: string): string {
  return `games/${gameId}/save_data`;
}
