package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"CloudLaunch_Go/internal/infrastructure/storage"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const maxRemoteImageBytes = 10 << 20

var remoteImageHTTPClient = &http.Client{Timeout: 15 * time.Second}

func (service *CloudSyncService) downloadCloudImagePath(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	cloud storage.CloudGameMetadata,
) (*string, int, error) {
	imagePath := (*string)(nil)
	downloadedImages := 0
	if cloud.ImageKey != nil && strings.TrimSpace(*cloud.ImageKey) != "" {
		path, downloaded, err := service.downloadImageIfNeeded(ctx, client, bucket, cloud.ID, *cloud.ImageKey)
		if err != nil {
			return nil, 0, err
		}
		imagePath = &path
		if downloaded {
			downloadedImages++
		}
	}
	return imagePath, downloadedImages, nil
}

func (service *CloudSyncService) uploadImageIfNeeded(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	gameID string,
	imagePath string,
	existing *storage.CloudGameMetadata,
) (string, bool, error) {
	payload, ext, contentType, err := service.imageLoader.Load(imagePath)
	if err != nil {
		return "", false, err
	}
	key := cloudImageObjectKey(gameID, payload, ext, contentType)

	if existing != nil && existing.ImageKey != nil && *existing.ImageKey == key {
		return key, false, nil
	}

	if err := service.cloudStorage.UploadBytes(ctx, client, bucket, key, payload, contentType); err != nil {
		return "", false, err
	}
	return key, true, nil
}

func cloudImageObjectKey(gameID string, payload []byte, ext string, contentType string) string {
	hash := sha256.Sum256(payload)
	hashHex := hex.EncodeToString(hash[:])
	normalizedExt := normalizeImageExt(ext, contentType)
	return fmt.Sprintf("games/%s/thumbnail/%s%s", gameID, hashHex, normalizedExt)
}

func (service *CloudSyncService) downloadImageIfNeeded(
	ctx context.Context,
	client *s3.Client,
	bucket string,
	gameID string,
	key string,
) (string, bool, error) {
	targetPath, err := cloudImageLocalPath(service.config.AppDataDir, gameID, key)
	if err != nil {
		return "", false, err
	}
	targetDir := filepath.Join(service.config.AppDataDir, "thumbnails")
	if err := service.imageFiles.EnsureDir(targetDir); err != nil {
		return "", false, err
	}
	exists, err := service.imageFiles.Exists(targetPath)
	if err != nil {
		return "", false, err
	}
	if exists {
		return targetPath, false, nil
	}

	payload, err := service.cloudStorage.DownloadObject(ctx, client, bucket, key)
	if err != nil {
		return "", false, err
	}
	if err := service.imageFiles.WriteFile(targetPath, payload, 0o600); err != nil {
		return "", false, err
	}
	return targetPath, true, nil
}

func cloudImageLocalPath(appDataDir string, gameID string, key string) (string, error) {
	baseName := filepath.Base(key)
	if baseName == "" {
		return "", errors.New("image key is empty")
	}
	ext := filepath.Ext(baseName)
	hash := strings.TrimSuffix(baseName, ext)
	if hash == "" {
		return "", errors.New("image hash is empty")
	}
	return filepath.Join(appDataDir, "thumbnails", fmt.Sprintf("%s_%s%s", hash, gameID, ext)), nil
}

func loadImagePayload(path string) ([]byte, string, string, error) {
	if isURL(path) {
		parsed, err := url.Parse(path)
		if err != nil {
			return nil, "", "", err
		}
		if err := validateRemoteImageURL(parsed); err != nil {
			return nil, "", "", err
		}

		request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, parsed.String(), nil)
		if err != nil {
			return nil, "", "", err
		}
		response, err := remoteImageHTTPClient.Do(request)
		if err != nil {
			return nil, "", "", err
		}
		defer func() {
			if closeErr := response.Body.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
		}()
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			return nil, "", "", fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}
		if response.ContentLength > maxRemoteImageBytes {
			return nil, "", "", fmt.Errorf("image is too large: %d", response.ContentLength)
		}

		payload, err := io.ReadAll(io.LimitReader(response.Body, maxRemoteImageBytes+1))
		if err != nil {
			return nil, "", "", err
		}
		if len(payload) > maxRemoteImageBytes {
			return nil, "", "", fmt.Errorf("image is too large: %d", len(payload))
		}
		ext := filepath.Ext(response.Request.URL.Path)
		contentType := strings.TrimSpace(response.Header.Get("Content-Type"))
		if contentType == "" {
			contentType = http.DetectContentType(payload)
		}
		return payload, ext, contentType, nil
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, "", "", err
	}
	ext := filepath.Ext(path)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = http.DetectContentType(payload)
	}
	return payload, ext, contentType, nil
}

func normalizeImageExt(ext string, contentType string) string {
	trimmed := strings.ToLower(strings.TrimSpace(ext))
	if trimmed != "" {
		if strings.HasPrefix(trimmed, ".") {
			return trimmed
		}
		return "." + trimmed
	}
	if strings.Contains(contentType, "png") {
		return ".png"
	}
	if strings.Contains(contentType, "gif") {
		return ".gif"
	}
	if strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg") {
		return ".jpg"
	}
	return ".png"
}

func isURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func validateRemoteImageURL(parsed *url.URL) error {
	if parsed == nil {
		return errors.New("url is nil")
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return errors.New("url hostname is empty")
	}
	if strings.EqualFold(host, "localhost") {
		return errors.New("localhost is not allowed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		if isPrivateOrLocalAddr(addr) {
			return fmt.Errorf("private or local address is not allowed: %s", addr.String())
		}
	}
	return nil
}

func isPrivateOrLocalAddr(addr netip.Addr) bool {
	return addr.IsLoopback() ||
		addr.IsPrivate() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsMulticast() ||
		addr.IsUnspecified()
}
