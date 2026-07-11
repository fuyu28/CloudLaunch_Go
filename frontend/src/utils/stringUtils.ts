/**
 * @fileoverview 文字列操作ユーティリティ
 *
 */

export function createRemotePath(gameId: string): string {
  return `games/${gameId}/save_data`;
}
