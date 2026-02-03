// @fileoverview 批評空間の検索結果モデルを定義する。
package models

// ErogameScapeSearchItem は検索結果のゲーム情報を表す。
type ErogameScapeSearchItem struct {
	ErogameScapeID string `json:"erogameScapeId"`
	Title          string `json:"title"`
	Brand          string `json:"brand,omitempty"`
	GameURL        string `json:"gameUrl"`
}

// ErogameScapeSearchResult は検索結果一覧を表す。
type ErogameScapeSearchResult struct {
	Items       []ErogameScapeSearchItem `json:"items"`
	NextPageURL string                   `json:"nextPageUrl,omitempty"`
}
