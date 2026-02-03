// @fileoverview 外部サイトからのゲーム取り込み情報を定義する。
package models

// GameImport は外部サイトから取得したゲーム情報を表す。
type GameImport struct {
	ErogameScapeID string `json:"erogameScapeId"`
	Title          string `json:"title"`
	Brand          string `json:"brand"`
	ImagePath      string `json:"imagePath"`
	ImageURL       string `json:"imageUrl,omitempty"`
}
