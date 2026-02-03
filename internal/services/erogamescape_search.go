// @fileoverview 批評空間の検索結果を取得する。
package services

import (
	"context"
	"net/url"
	"strings"

	"CloudLaunch_Go/internal/models"

	"github.com/PuerkitoBio/goquery"
)

const erogameScapeSearchBaseURL = "https://erogamescape.dyndns.org/~ap2/ero/toukei_kaiseki/kensaku.php"

// SearchErogameScape はタイトル検索を行い、候補一覧を返す。
func (service *ErogameScapeService) SearchErogameScape(
	ctx context.Context,
	query string,
	pageURL string,
) (models.ErogameScapeSearchResult, error) {
	if strings.TrimSpace(query) == "" && strings.TrimSpace(pageURL) == "" {
		return models.ErogameScapeSearchResult{}, InvalidUrlError{URL: "empty query"}
	}

	targetURL := pageURL
	if strings.TrimSpace(targetURL) == "" {
		params := url.Values{}
		params.Set("category", "game")
		params.Set("word_category", "name")
		params.Set("word", query)
		params.Set("mode", "normal")
		targetURL = erogameScapeSearchBaseURL + "?" + params.Encode()
	}

	html, error := service.fetchHTML(ctx, targetURL)
	if error != nil {
		return models.ErogameScapeSearchResult{}, error
	}

	doc, error := goquery.NewDocumentFromReader(strings.NewReader(html))
	if error != nil {
		return models.ErogameScapeSearchResult{}, ParseError{Field: "searchDocument", Err: error}
	}

	items := make([]models.ErogameScapeSearchItem, 0, 50)
	doc.Find("table tr").Each(func(_ int, row *goquery.Selection) {
		link := row.Find("a[href*=\"game.php?game=\"]").First()
		if link.Length() == 0 {
			return
		}
		href, ok := link.Attr("href")
		if !ok || strings.TrimSpace(href) == "" {
			return
		}
		gameURL, error := resolveURL(targetURL, href)
		if error != nil {
			return
		}
		gameID, error := extractErogameScapeID(gameURL)
		if error != nil {
			return
		}
		title := strings.TrimSpace(link.Text())
		if title == "" {
			return
		}
		brand := strings.TrimSpace(row.Find("td").Eq(2).Text())
		items = append(items, models.ErogameScapeSearchItem{
			ErogameScapeID: gameID,
			Title:          title,
			Brand:          brand,
			GameURL:        gameURL,
		})
	})

	nextURL, _ := doc.Find("a:contains(\"次\")").Attr("href")
	if nextURL != "" {
		resolved, err := resolveURL(targetURL, nextURL)
		if err == nil {
			nextURL = resolved
		}
	}

	return models.ErogameScapeSearchResult{
		Items:       items,
		NextPageURL: nextURL,
	}, nil
}
