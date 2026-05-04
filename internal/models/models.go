// @fileoverview データベースとAPIで使う基本モデルを定義する。
package models

import "time"

// PlayStatus はゲームのプレイ状態を表す。
type PlayStatus string

const (
	PlayStatusUnplayed PlayStatus = "unplayed"
	PlayStatusPlaying  PlayStatus = "playing"
	PlayStatusPlayed   PlayStatus = "played"
)

// Game はゲーム基本情報を表す。
type Game struct {
	ID                     string     `json:"id"`
	Title                  string     `json:"title"`
	Publisher              string     `json:"publisher"`
	ImagePath              *string    `json:"imagePath,omitempty"`
	ExePath                string     `json:"exePath"`
	SaveFolderPath         *string    `json:"saveFolderPath,omitempty"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
	LocalSaveHash          *string    `json:"localSaveHash,omitempty"`
	LocalSaveHashUpdatedAt *time.Time `json:"localSaveHashUpdatedAt,omitempty"`
	PlayStatus             PlayStatus `json:"playStatus"`
	TotalPlayTime          int64      `json:"totalPlayTime"`
	LastPlayed             *time.Time `json:"lastPlayed,omitempty"`
	ClearedAt              *time.Time `json:"clearedAt,omitempty"`
}

// PlaySession はプレイセッションを表す。
type PlaySession struct {
	ID          string    `json:"id"`
	GameID      string    `json:"gameId"`
	PlayRouteID *string   `json:"playRouteId,omitempty"`
	PlayedAt    time.Time `json:"playedAt"`
	Duration    int64     `json:"duration"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// PlayRoute はゲームごとのプレイルートを表す。
type PlayRoute struct {
	ID        string    `json:"id"`
	GameID    string    `json:"gameId"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sortOrder"`
	CreatedAt time.Time `json:"createdAt"`
}

// Memo はメモ情報を表す。
type Memo struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	GameID    string    `json:"gameId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// MonitoringGameStatus はゲーム監視の状態を表す。
type MonitoringGameStatus struct {
	GameID            string `json:"gameId"`
	GameTitle         string `json:"gameTitle"`
	ExeName           string `json:"exeName"`
	IsPlaying         bool   `json:"isPlaying"`
	PlayTime          int64  `json:"playTime"`
	IsPaused          bool   `json:"isPaused"`
	NeedsConfirmation bool   `json:"needsConfirmation"`
	NeedsResume       bool   `json:"needsResume"`
}

// ProcessSnapshotItem はプロセス監視デバッグ用の情報を表す。
type ProcessSnapshotItem struct {
	Name           string `json:"name"`
	Pid            int    `json:"pid"`
	Cmd            string `json:"cmd"`
	NormalizedName string `json:"normalizedName"`
	NormalizedCmd  string `json:"normalizedCmd"`
}

// ProcessSnapshot はプロセス取得結果を表す。
type ProcessSnapshot struct {
	Source string                `json:"source"`
	Items  []ProcessSnapshotItem `json:"items"`
}
