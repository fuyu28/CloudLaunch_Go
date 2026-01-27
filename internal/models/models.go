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
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	Publisher      string     `json:"publisher"`
	ImagePath      *string    `json:"imagePath,omitempty"`
	ExePath        string     `json:"exePath"`
	SaveFolderPath *string    `json:"saveFolderPath,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	PlayStatus     PlayStatus `json:"playStatus"`
	TotalPlayTime  int64      `json:"totalPlayTime"`
	LastPlayed     *time.Time `json:"lastPlayed,omitempty"`
	ClearedAt      *time.Time `json:"clearedAt,omitempty"`
	CurrentChapter *string    `json:"currentChapter,omitempty"`
}

// PlaySession はプレイセッションを表す。
type PlaySession struct {
	ID          string    `json:"id"`
	GameID      string    `json:"gameId"`
	PlayedAt    time.Time `json:"playedAt"`
	Duration    int64     `json:"duration"`
	SessionName *string   `json:"sessionName,omitempty"`
	ChapterID   *string   `json:"chapterId,omitempty"`
	UploadID    *string   `json:"uploadId,omitempty"`
}

// Chapter は章情報を表す。
type Chapter struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Order     int64     `json:"order"`
	GameID    string    `json:"gameId"`
	CreatedAt time.Time `json:"createdAt"`
}

// Upload はクラウドアップロード履歴を表す。
type Upload struct {
	ID        string    `json:"id"`
	ClientID  *string   `json:"clientId,omitempty"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"createdAt"`
	GameID    string    `json:"gameId"`
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

// ChapterStat は章統計の出力を表す。
type ChapterStat struct {
	ChapterID    string  `json:"chapterId"`
	ChapterName  string  `json:"chapterName"`
	TotalTime    int64   `json:"totalTime"`
	SessionCount int64   `json:"sessionCount"`
	AverageTime  float64 `json:"averageTime"`
	Order        int64   `json:"order"`
}
