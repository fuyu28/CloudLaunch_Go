package services

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"CloudLaunch_Go/internal/config"
)

type erogameScapeRoundTripperFunc func(*http.Request) (*http.Response, error)

func (fn erogameScapeRoundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func newErogameScapeTestService(t *testing.T, transport http.RoundTripper) *ErogameScapeService {
	t.Helper()
	service := NewErogameScapeService(
		config.Config{AppDataDir: t.TempDir()},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	service.httpClient.Transport = transport
	return service
}

func TestValidateErogameScapeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rawURL  string
		kind    erogameScapeURLKind
		wantErr bool
	}{
		{name: "dyndns HTTPS page", rawURL: "https://erogamescape.dyndns.org/~ap2/ero/toukei_kaiseki/game.php?game=1", kind: erogameScapePageURL},
		{name: "dyndns HTTP fallback page", rawURL: "http://erogamescape.dyndns.org/~ap2/ero/toukei_kaiseki/game.php?game=1", kind: erogameScapePageURL},
		{name: "current org page", rawURL: "https://erogamescape.org/~ap2/ero/toukei_kaiseki/game.php?game=1", kind: erogameScapePageURL},
		{name: "DMM image", rawURL: "https://pics.dmm.co.jp/mono/game/example/examplepl.jpg", kind: erogameScapeImageURL},
		{name: "DLsite image", rawURL: "https://img.dlsite.jp/modpub/images2/work/example.jpg", kind: erogameScapeImageURL},
		{name: "Surugaya image", rawURL: "https://www.suruga-ya.jp/database/pics/game/example.jpg", kind: erogameScapeImageURL},
		{name: "localhost", rawURL: "http://localhost/game.php?game=1", kind: erogameScapePageURL, wantErr: true},
		{name: "loopback address", rawURL: "http://127.0.0.1/game.php?game=1", kind: erogameScapePageURL, wantErr: true},
		{name: "suffix spoofing", rawURL: "https://erogamescape.dyndns.org.evil.example/game.php?game=1", kind: erogameScapePageURL, wantErr: true},
		{name: "userinfo spoofing", rawURL: "https://erogamescape.dyndns.org@evil.example/game.php?game=1", kind: erogameScapePageURL, wantErr: true},
		{name: "userinfo on allowed host", rawURL: "https://user@erogamescape.dyndns.org/game.php?game=1", kind: erogameScapePageURL, wantErr: true},
		{name: "unexpected HTTPS port", rawURL: "https://erogamescape.dyndns.org:443/game.php?game=1", kind: erogameScapePageURL, wantErr: true},
		{name: "explicit empty port", rawURL: "https://erogamescape.dyndns.org:/game.php?game=1", kind: erogameScapePageURL, wantErr: true},
		{name: "unexpected scheme", rawURL: "ftp://erogamescape.dyndns.org/game.php?game=1", kind: erogameScapePageURL, wantErr: true},
		{name: "image over HTTP", rawURL: "http://pics.dmm.co.jp/example.jpg", kind: erogameScapeImageURL, wantErr: true},
		{name: "unlisted image host", rawURL: "https://images.example.com/example.jpg", kind: erogameScapeImageURL, wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := validateErogameScapeURL(test.rawURL, test.kind)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateErogameScapeURL() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr {
				var invalidURLError InvalidUrlError
				if !errors.As(err, &invalidURLError) {
					t.Fatalf("error type = %T, want InvalidUrlError", err)
				}
			}
		})
	}
}

func TestFetchHTMLRejectsUnlistedInitialHostBeforeRequest(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	service := newErogameScapeTestService(t, erogameScapeRoundTripperFunc(func(*http.Request) (*http.Response, error) {
		requestCount.Add(1)
		return nil, errors.New("unexpected request")
	}))

	_, err := service.fetchHTML(context.Background(), "https://localhost/game.php?game=1")
	var invalidURLError InvalidUrlError
	if !errors.As(err, &invalidURLError) {
		t.Fatalf("error = %v, want InvalidUrlError", err)
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("request count = %d, want 0", got)
	}
}

func TestFetchHTMLRejectsRedirectOutsidePageAllowlist(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	service := newErogameScapeTestService(t, erogameScapeRoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		requestCount.Add(1)
		if request.URL.Hostname() != "erogamescape.dyndns.org" {
			t.Fatalf("request reached unlisted host %q", request.URL.Hostname())
		}
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": []string{"https://localhost/private"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    request,
		}, nil
	}))

	_, err := service.fetchHTMLOnce(
		context.Background(),
		"https://erogamescape.dyndns.org/~ap2/ero/toukei_kaiseki/game.php?game=1",
	)
	var invalidURLError InvalidUrlError
	if !errors.As(err, &invalidURLError) {
		t.Fatalf("error = %v, want wrapped InvalidUrlError", err)
	}
	if got := requestCount.Load(); got != 1 {
		t.Fatalf("request count = %d, want 1", got)
	}
}

func TestDownloadImageRejectsRedirectOutsideImageAllowlist(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	service := newErogameScapeTestService(t, erogameScapeRoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		requestCount.Add(1)
		if request.URL.Hostname() != "pics.dmm.co.jp" {
			t.Fatalf("request reached unlisted host %q", request.URL.Hostname())
		}
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": []string{"https://evil.example/image.jpg"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    request,
		}, nil
	}))

	_, err := service.downloadAndSaveImage(
		context.Background(),
		"https://pics.dmm.co.jp/mono/game/example/examplepl.jpg",
		"1",
	)
	var invalidURLError InvalidUrlError
	if !errors.As(err, &invalidURLError) {
		t.Fatalf("error = %v, want wrapped InvalidUrlError", err)
	}
	if got := requestCount.Load(); got != 1 {
		t.Fatalf("request count = %d, want 1", got)
	}
}

func TestDownloadImageRejectsUnlistedInitialHostBeforeRequest(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	service := newErogameScapeTestService(t, erogameScapeRoundTripperFunc(func(*http.Request) (*http.Response, error) {
		requestCount.Add(1)
		return nil, errors.New("unexpected request")
	}))

	_, err := service.downloadAndSaveImage(
		context.Background(),
		"https://localhost/image.jpg",
		"1",
	)
	var invalidURLError InvalidUrlError
	if !errors.As(err, &invalidURLError) {
		t.Fatalf("error = %v, want wrapped InvalidUrlError", err)
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("request count = %d, want 0", got)
	}
}

func TestSearchErogameScapeFiltersUnlistedResultAndPaginationURLs(t *testing.T) {
	t.Parallel()

	const searchHTML = `
<html><body>
<table>
  <tr><td><a href="game.php?game=1">Allowed Game</a></td><td></td><td>Allowed Brand</td></tr>
  <tr><td><a href="https://evil.example/game.php?game=2">Spoofed Game</a></td><td></td><td>Evil Brand</td></tr>
</table>
<a href="https://localhost/kensaku.php?page=2">次</a>
</body></html>`

	service := newErogameScapeTestService(t, erogameScapeRoundTripperFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(searchHTML)),
			Request:    request,
		}, nil
	}))

	result, err := service.SearchErogameScape(
		context.Background(),
		"",
		"https://erogamescape.dyndns.org/~ap2/ero/toukei_kaiseki/kensaku.php?page=1",
	)
	if err != nil {
		t.Fatalf("SearchErogameScape() error = %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].ErogameScapeID != "1" {
		t.Fatalf("items = %#v, want only allowed game", result.Items)
	}
	if result.NextPageURL != "" {
		t.Fatalf("NextPageURL = %q, want empty", result.NextPageURL)
	}
}
