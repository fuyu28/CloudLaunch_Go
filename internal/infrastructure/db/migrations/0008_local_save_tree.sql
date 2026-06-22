-- localSaveTree は前回同期した SaveSnapshot の JSON（path→hash）を保持する。
-- Pull 時に「同期管理下にあったファイル（tracked）」と「ローカル固有の未追跡ファイル
-- （untracked）」を区別し、未追跡ファイルを無確認で削除しないための基準（base tree）。
ALTER TABLE "Game" ADD COLUMN "localSaveTree" TEXT;
