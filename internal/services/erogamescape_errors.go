// @fileoverview 批評空間取り込み用のエラー型を定義する。
package services

import "fmt"

// InvalidUrlError はURLからIDを抽出できない場合のエラーを表す。
type InvalidUrlError struct {
	URL string
}

func (err InvalidUrlError) Error() string {
	return fmt.Sprintf("invalid erogamescape url: %s", err.URL)
}

// FetchError はHTML取得に失敗した場合のエラーを表す。
type FetchError struct {
	URL        string
	StatusCode int
	Err        error
}

func (err FetchError) Error() string {
	if err.StatusCode != 0 {
		return fmt.Sprintf("failed to fetch page: %s (status=%d)", err.URL, err.StatusCode)
	}
	return fmt.Sprintf("failed to fetch page: %s", err.URL)
}

func (err FetchError) Unwrap() error {
	return err.Err
}

// ParseError はDOM解析に失敗した場合のエラーを表す。
type ParseError struct {
	Field string
	Err   error
}

func (err ParseError) Error() string {
	if err.Field == "" {
		return "failed to parse erogamescape page"
	}
	return fmt.Sprintf("failed to parse erogamescape page (%s)", err.Field)
}

func (err ParseError) Unwrap() error {
	return err.Err
}

// ImageError は画像取得・保存に失敗した場合のエラーを表す。
type ImageError struct {
	URL string
	Err error
}

func (err ImageError) Error() string {
	return fmt.Sprintf("failed to handle image: %s", err.URL)
}

func (err ImageError) Unwrap() error {
	return err.Err
}
