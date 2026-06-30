package domain

import "time"

type BlobHash = string

// SaveSnapshot はセーブフォルダのファイル一覧とそのハッシュを表す（git のツリー相当）。
type SaveSnapshot struct {
	Files map[string]BlobHash `json:"files"`
}

// MetaSnapshot はある時点のゲームデータ全体を表す（git のコミット相当）。
type MetaSnapshot struct {
	GameJSON     BlobHash  `json:"game.json"`
	SessionsJSON BlobHash  `json:"sessions.json"`
	Saves        BlobHash  `json:"saves"`
	DeviceName   string    `json:"deviceName"`
	CreatedAt    time.Time `json:"createdAt"`
}

type SyncStatus string

const (
	SyncStatusNeverSynced SyncStatus = "never_synced"
	SyncStatusInSync      SyncStatus = "in_sync"
	SyncStatusPushNeeded  SyncStatus = "push_needed"
	SyncStatusPullNeeded  SyncStatus = "pull_needed"
	SyncStatusConflict    SyncStatus = "conflict"
)

// SyncStatusDetail は同期状態の詳細を表す。
type SyncStatusDetail struct {
	Status     SyncStatus    `json:"status"`
	LocalMeta  *MetaSnapshot `json:"localMeta,omitempty"`
	RemoteMeta *MetaSnapshot `json:"remoteMeta,omitempty"`
}

// PullResult は Pull / ResolveConflict(リモート採用) の結果を表す。
//
// Applied=false かつ UntrackedDeletes が非空のときは「未追跡ファイルの削除確認待ち」を表し、
// この時点ではローカルに一切変更を加えていない。呼び出し側は一覧をユーザーに提示し、
// 承認されたら deleteUntracked=true で再実行する。
type PullResult struct {
	Applied          bool     `json:"applied"`
	UntrackedDeletes []string `json:"untrackedDeletes,omitempty"`
}
