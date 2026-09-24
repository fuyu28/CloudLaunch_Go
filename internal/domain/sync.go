// コンテンツアドレッシング同期のドメインモデル（スナップショット・同期状態）を定義する。
package domain

import "time"

type BlobHash = string

// SaveSnapshot はセーブフォルダのファイル一覧とそのハッシュを表す（git のツリー相当）。
type SaveSnapshot struct {
	Files map[string]BlobHash `json:"files"`
}

// 同期プロトコルのスキーマ版。v1 は SchemaVersion 未設定（0）で routes.json を持たない。
// v2 は Route 本体をコミット構成要素に含める（H8）。
const (
	SyncSchemaVersionV1 = 0
	SyncSchemaVersionV2 = 2
)

// MetaSnapshot はある時点のゲームデータ全体を表す（git のコミット相当）。
//
// FileCount / TotalSize はクラウド一覧で「セーブツリーを別途取得せずに」
// ファイル数と総サイズを表示するためのスナップショット時点のキャッシュ。
// 同期判定（contentFingerprint）には含まれず、欠落しても整合性に影響しない。
// 旧クライアントが書いた commit には値が無いため omitempty + ゼロ値時は
// 表示側で「未取得」として扱う。
//
// SchemaVersion / RoutesJSON は v2 から。旧 commit では欠落し、v1 互換経路で扱う。
type MetaSnapshot struct {
	SchemaVersion int       `json:"schemaVersion,omitempty"`
	GameJSON      BlobHash  `json:"game.json"`
	SessionsJSON  BlobHash  `json:"sessions.json"`
	RoutesJSON    BlobHash  `json:"routes.json,omitempty"`
	Saves         BlobHash  `json:"saves"`
	DeviceName    string    `json:"deviceName"`
	CreatedAt     time.Time `json:"createdAt"`
	FileCount     int64     `json:"fileCount,omitempty"`
	TotalSize     int64     `json:"totalSize,omitempty"`
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

// PendingPush は Push のリモート HEAD 更新後にローカル baseline 確定が未完了の記録。
// expectedRemoteHead は初回 Push や force 時は空文字になり得る。
type PendingPush struct {
	GameID             string
	ExpectedRemoteHead string
	NewCommitHash      string
	ContentFingerprint string
	SaveTree           string
}

// PullOperationStatus はセーブディレクトリ交換ジャーナルの状態。
type PullOperationStatus string

const (
	// PullOperationPrepared は rename 直前〜DB 未反映。起動時は旧 live へ戻す。
	PullOperationPrepared PullOperationStatus = "PREPARED"
	// PullOperationApplied は DB 反映済み。backup 削除と journal 消去が残る。
	PullOperationApplied PullOperationStatus = "APPLIED"
)

// PullOperation は Pull の同ボリューム stage/backup 交換を回復可能にするためのジャーナル。
// live/stage/backup は絶対パス（saveDir の兄弟）を保持する。
// HadLive は交換前に live が存在したか。PREPARED 復旧で「新 live を捨てて空に戻す」判定に使う。
type PullOperation struct {
	OperationID string
	GameID      string
	LivePath    string
	StagePath   string
	BackupPath  string
	CommitHash  string
	Status      PullOperationStatus
	HadLive     bool
}
