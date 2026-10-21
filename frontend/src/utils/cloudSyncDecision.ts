/**
 * @fileoverview ゲーム起動前・プレイ終了後に同期状態からどの確認を出すかを決めるヘルパ。
 * セーブの転送は確認モーダルで承認されたときだけ行い、セーブが同一のときは
 * メタデータ（ゲーム情報・セッション）だけを確認なしで同期する。
 */

import type { SyncStatusDetail } from "src/wailsBridge";

export type LaunchSyncAction = "launch" | "syncMetadata" | "confirmDownload" | "resolveConflict";

export function decideLaunchSyncAction(detail: SyncStatusDetail): LaunchSyncAction {
  switch (detail.status) {
    case "conflict":
      return "resolveConflict";
    case "pull_needed":
      return detail.savesDiffer ? "confirmDownload" : "syncMetadata";
    case "push_needed":
      // セーブ差分ありの push_needed は、前回のアップロード確認で「しない」を選んだ状態。
      // 起動時に黙って送らない。
      return detail.savesDiffer ? "launch" : "syncMetadata";
    default:
      return "launch";
  }
}

export type AfterSessionSyncAction = "none" | "confirmUpload" | "resolveConflict";

export function decideAfterSessionSyncAction(detail: SyncStatusDetail): AfterSessionSyncAction {
  switch (detail.status) {
    case "never_synced":
      return "confirmUpload";
    case "push_needed":
      return detail.savesDiffer ? "confirmUpload" : "none";
    case "conflict":
      // 通常 Push はリモート更新を検知して必ず失敗するので、競合解決の選択に回す。
      return detail.savesDiffer ? "resolveConflict" : "none";
    default:
      return "none";
  }
}
