// @fileoverview 批評空間（ErogameScape）からゲーム情報を取得する。
package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"CloudLaunch_Go/internal/config"
	"CloudLaunch_Go/internal/models"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/image/draw"
)

const erogameScapeShortEdgePx = 200

var erogameScapeGameIDRegex = regexp.MustCompile(`game=(\d+)`)

// ErogameScapeService は批評空間から情報を取得する。
type ErogameScapeService struct {
	appDataDir string
	logger     *slog.Logger
	httpClient *http.Client
}

// NewErogameScapeService は ErogameScapeService を生成する。
func NewErogameScapeService(cfg config.Config, logger *slog.Logger) *ErogameScapeService {
	return &ErogameScapeService{
		appDataDir: cfg.AppDataDir,
		logger:     logger,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// FetchFromErogameScape は批評空間のURLからゲーム情報を取得する。
func (service *ErogameScapeService) FetchFromErogameScape(ctx context.Context, gamePageURL string) (models.GameImport, error) {
	gameID, error := extractErogameScapeID(gamePageURL)
	if error != nil {
		return models.GameImport{}, error
	}

	pageHTML, error := service.fetchHTML(ctx, gamePageURL)
	if error != nil {
		return models.GameImport{}, error
	}

	doc, error := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if error != nil {
		return models.GameImport{}, ParseError{Field: "document", Err: error}
	}

	title := strings.TrimSpace(doc.Find("#game_title > a").First().Text())
	if title == "" {
		return models.GameImport{}, ParseError{Field: "title", Err: errors.New("title not found")}
	}

	brand := strings.TrimSpace(doc.Find("#brand > td > a").First().Text())
	if brand == "" {
		return models.GameImport{}, ParseError{Field: "brand", Err: errors.New("brand not found")}
	}

	imageSrc, ok := doc.Find("#main_image img").First().Attr("src")
	if !ok || strings.TrimSpace(imageSrc) == "" {
		return models.GameImport{}, ParseError{Field: "imageUrl", Err: errors.New("image url not found")}
	}

	imageURL, error := resolveURL(gamePageURL, strings.TrimSpace(imageSrc))
	if error != nil {
		return models.GameImport{}, ParseError{Field: "imageUrl", Err: error}
	}

	imagePath, error := service.downloadAndSaveImage(ctx, imageURL, gameID)
	if error != nil {
		return models.GameImport{}, error
	}

	return models.GameImport{
		ErogameScapeID: gameID,
		Title:          title,
		Brand:          brand,
		ImagePath:      imagePath,
		ImageURL:       imageURL,
	}, nil
}

func extractErogameScapeID(gamePageURL string) (string, error) {
	matches := erogameScapeGameIDRegex.FindStringSubmatch(gamePageURL)
	if len(matches) < 2 {
		return "", InvalidUrlError{URL: gamePageURL}
	}
	return matches[1], nil
}

func (service *ErogameScapeService) fetchHTML(ctx context.Context, gamePageURL string) (string, error) {
	html, error := service.fetchHTMLOnce(ctx, gamePageURL)
	if error == nil {
		return html, nil
	}

	parsed, parseErr := url.Parse(gamePageURL)
	if parseErr != nil {
		return "", error
	}
	if strings.EqualFold(parsed.Scheme, "https") {
		parsed.Scheme = "http"
		fallbackURL := parsed.String()
		fallbackHTML, fallbackErr := service.fetchHTMLOnce(ctx, fallbackURL)
		if fallbackErr == nil {
			return fallbackHTML, nil
		}
	}
	return "", error
}

func (service *ErogameScapeService) fetchHTMLOnce(ctx context.Context, gamePageURL string) (string, error) {
	request, error := http.NewRequestWithContext(ctx, http.MethodGet, gamePageURL, nil)
	if error != nil {
		return "", FetchError{URL: gamePageURL, Err: error}
	}
	request.Header.Set("User-Agent", "CloudLaunch/1.0")

	response, error := service.httpClient.Do(request)
	if error != nil {
		return "", FetchError{URL: gamePageURL, Err: error}
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			service.logger.Warn("批評空間HTMLレスポンスのクローズに失敗", "error", closeErr)
		}
	}()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", FetchError{URL: gamePageURL, StatusCode: response.StatusCode, Err: errors.New(response.Status)}
	}

	body, error := io.ReadAll(response.Body)
	if error != nil {
		return "", FetchError{URL: gamePageURL, Err: error}
	}
	return string(body), nil
}

func (service *ErogameScapeService) downloadAndSaveImage(ctx context.Context, imageURL string, gameID string) (string, error) {
	request, error := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if error != nil {
		return "", ImageError{URL: imageURL, Err: error}
	}
	request.Header.Set("User-Agent", "CloudLaunch/1.0")

	response, error := service.httpClient.Do(request)
	if error != nil {
		return "", ImageError{URL: imageURL, Err: error}
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			service.logger.Warn("批評空間画像レスポンスのクローズに失敗", "error", closeErr)
		}
	}()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", ImageError{URL: imageURL, Err: errors.New(response.Status)}
	}

	raw, error := io.ReadAll(response.Body)
	if error != nil {
		return "", ImageError{URL: imageURL, Err: error}
	}

	decoded, format, error := image.Decode(bytes.NewReader(raw))
	if error != nil {
		return "", ImageError{URL: imageURL, Err: error}
	}

	resized := resizeToShortEdge(decoded, erogameScapeShortEdgePx)
	ext := chooseImageExtension(imageURL, response.Header.Get("Content-Type"), format)
	if ext == "" {
		ext = ".jpg"
	}

	var encoded bytes.Buffer
	if error := encodeImage(&encoded, resized, ext); error != nil {
		return "", ImageError{URL: imageURL, Err: error}
	}
	hash := sha256.Sum256(encoded.Bytes())
	hashHex := hex.EncodeToString(hash[:])

	targetDir := filepath.Join(service.appDataDir, "thumbnails")
	if error := os.MkdirAll(targetDir, 0o700); error != nil {
		return "", ImageError{URL: imageURL, Err: error}
	}

	filename := fmt.Sprintf("%s_%s%s", hashHex, gameID, ext)
	fullPath := filepath.Join(targetDir, filename)

	file, error := os.Create(fullPath)
	if error != nil {
		return "", ImageError{URL: imageURL, Err: error}
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			service.logger.Warn("サムネイル保存後のファイルクローズに失敗", "error", closeErr)
		}
	}()

	if _, error := encoded.WriteTo(file); error != nil {
		return "", ImageError{URL: imageURL, Err: error}
	}

	return fullPath, nil
}

func resizeToShortEdge(source image.Image, shortEdge int) image.Image {
	bounds := source.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width == 0 || height == 0 {
		return source
	}

	var targetWidth int
	var targetHeight int
	if width <= height {
		targetWidth = shortEdge
		targetHeight = int(float64(height) * (float64(shortEdge) / float64(width)))
	} else {
		targetHeight = shortEdge
		targetWidth = int(float64(width) * (float64(shortEdge) / float64(height)))
	}

	if targetWidth <= 0 || targetHeight <= 0 {
		return source
	}

	target := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.CatmullRom.Scale(target, target.Bounds(), source, bounds, draw.Over, nil)
	return target
}

func chooseImageExtension(imageURL string, contentType string, format string) string {
	parsed, err := url.Parse(imageURL)
	if err == nil {
		ext := strings.ToLower(path.Ext(parsed.Path))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" {
			if ext == ".jpeg" {
				return ".jpg"
			}
			return ext
		}
	}

	if contentType != "" {
		if exts, err := mime.ExtensionsByType(contentType); err == nil {
			for _, ext := range exts {
				ext = strings.ToLower(ext)
				if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" {
					if ext == ".jpeg" {
						return ".jpg"
					}
					return ext
				}
			}
		}
	}

	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		return ".jpg"
	case "png":
		return ".png"
	case "gif":
		return ".gif"
	default:
		return ""
	}
}

func encodeImage(writer io.Writer, img image.Image, ext string) error {
	switch ext {
	case ".jpg":
		return jpeg.Encode(writer, img, &jpeg.Options{Quality: 90})
	case ".png":
		return png.Encode(writer, img)
	case ".gif":
		return gif.Encode(writer, img, nil)
	default:
		return jpeg.Encode(writer, img, &jpeg.Options{Quality: 90})
	}
}

func resolveURL(baseURL string, ref string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	target, err := url.Parse(ref)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(target).String(), nil
}
