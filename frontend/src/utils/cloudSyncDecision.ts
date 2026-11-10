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

export type ManualTransferAction = "run" | "showStatus" | "resolveConflict";

// 押した方向がそのまま通る状態だけ実行し、それ以外は同期状態モーダルで理由を見せる。
// 状態を見ずに pull すると、このPCの未アップロードの変更を確認なしで上書きしてしまう。
export function decideManualTransfer(
  detail: SyncStatusDetail,
  direction: "upload" | "download",
): ManualTransferAction {
  if (detail.status === "conflict") {
    return "resolveConflict";
  }
  if (direction === "upload") {
    return detail.status === "push_needed" || detail.status === "never_synced"
      ? "run"
      : "showStatus";
  }
  // 基準なしの pull_needed は、このPC固有のセーブを置き換える可能性があるので説明を挟む。
  return detail.status === "pull_needed" && detail.hasLocalBaseline ? "run" : "showStatus";
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
