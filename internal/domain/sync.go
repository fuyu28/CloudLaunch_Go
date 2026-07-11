// コンテンツアドレッシング同期のドメインモデル（スナップショット・同期状態）を定義する。
package domain

import "time"

type BlobHash = string

// SaveSnapshot はセーブフォルダのファイル一覧とそのハッシュを表す（git のツリー相当）。
type SaveSnapshot struct {
	Files map[string]BlobHash `json:"files"`
}

// MetaSnapshot はある時点のゲームデータ全体を表す（git のコミット相当）。
//
// FileCount / TotalSize はクラウド一覧で「セーブツリーを別途取得せずに」
// ファイル数と総サイズを表示するためのスナップショット時点のキャッシュ。
// 同期判定（contentFingerprint）には含まれず、欠落しても整合性に影響しない。
// 旧クライアントが書いた commit には値が無いため omitempty + ゼロ値時は
// 表示側で「未取得」として扱う。
type MetaSnapshot struct {
	GameJSON     BlobHash  `json:"game.json"`
	SessionsJSON BlobHash  `json:"sessions.json"`
	Saves        BlobHash  `json:"saves"`
	DeviceName   string    `json:"deviceName"`
	CreatedAt    time.Time `json:"createdAt"`
	FileCount    int64     `json:"fileCount,omitempty"`
	TotalSize    int64     `json:"totalSize,omitempty"`
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
//
// SavesDiffer は Saves コンポーネントのみの差分を示す。fingerprint 比較の Status とは
// 独立に、セッションメタデータ差分（sessions.json / game.json の変化）を除外して
// セーブファイル内容だけの差分を確認するためのシグナル。
// セッション終了後のアップロード確認プロンプトなど、「セーブ不変ならプロンプト不要」
// を判定したい呼び出し側で status とあわせて狭窄条件に使う。
type SyncStatusDetail struct {
	Status      SyncStatus    `json:"status"`
	SavesDiffer bool          `json:"savesDiffer"`
	LocalMeta   *MetaSnapshot `json:"localMeta,omitempty"`
	RemoteMeta  *MetaSnapshot `json:"remoteMeta,omitempty"`
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
