// 批評空間連携で外部取得を許可するURLを検証する。
package services

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

type erogameScapeURLKind int

const (
	erogameScapePageURL erogameScapeURLKind = iota
	erogameScapeImageURL
)

var (
	erogameScapePageHosts = map[string]struct{}{
		"erogamescape.dyndns.org": {},
		"erogamescape.org":        {},
	}
	erogameScapeImageHosts = map[string]struct{}{
		"img.dlsite.jp":    {},
		"pics.dmm.co.jp":   {},
		"www.suruga-ya.jp": {},
	}
)

func validateErogameScapeURL(rawURL string, kind erogameScapeURLKind) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || !parsed.IsAbs() || parsed.User != nil || parsed.Port() != "" {
		return InvalidUrlError{URL: rawURL}
	}

	scheme := strings.ToLower(parsed.Scheme)
	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" || !strings.EqualFold(parsed.Host, parsed.Hostname()) {
		return InvalidUrlError{URL: rawURL}
	}

	var allowedHosts map[string]struct{}
	switch kind {
	case erogameScapePageURL:
		if scheme != "http" && scheme != "https" {
			return InvalidUrlError{URL: rawURL}
		}
		allowedHosts = erogameScapePageHosts
	case erogameScapeImageURL:
		if scheme != "https" {
			return InvalidUrlError{URL: rawURL}
		}
		allowedHosts = erogameScapeImageHosts
	default:
		return InvalidUrlError{URL: rawURL}
	}

	if _, allowed := allowedHosts[hostname]; !allowed {
		return InvalidUrlError{URL: rawURL}
	}
	return nil
}

func (service *ErogameScapeService) clientForErogameScapeURL(kind erogameScapeURLKind) *http.Client {
	client := *service.httpClient
	previousCheckRedirect := client.CheckRedirect
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if err := validateErogameScapeURL(request.URL.String(), kind); err != nil {
			return err
		}
		if previousCheckRedirect != nil {
			return previousCheckRedirect(request, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
	return &client
}
