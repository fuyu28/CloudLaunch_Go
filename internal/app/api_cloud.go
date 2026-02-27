// @fileoverview クラウド関連のAPIを提供する。
package app

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"CloudLaunch_Go/internal/credentials"
	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/storage"
	"CloudLaunch_Go/internal/util"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// CloudDataItem はクラウドデータ一覧の要素を表す。
type CloudDataItem struct {
	Name         string    `json:"name"`
	TotalSize    int64     `json:"totalSize"`
	FileCount    int64     `json:"fileCount"`
	LastModified time.Time `json:"lastModified"`
	RemotePath   string    `json:"remotePath"`
}

// CloudDirectoryNode はディレクトリツリーのノードを表す。
type CloudDirectoryNode struct {
	Name         string               `json:"name"`
	Path         string               `json:"path"`
	IsDirectory  bool                 `json:"isDirectory"`
	Size         int64                `json:"size"`
	LastModified time.Time            `json:"lastModified"`
	Children     []CloudDirectoryNode `json:"children,omitempty"`
	ObjectKey    *string              `json:"objectKey,omitempty"`
}

type directoryNodeBuilder struct {
	node     CloudDirectoryNode
	children map[string]*directoryNodeBuilder
}

// CloudFileDetail はクラウドファイル詳細を表す。
type CloudFileDetail struct {
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
	Key          string    `json:"key"`
	RelativePath string    `json:"relativePath"`
}

// CloudFileDetailsResult はファイル詳細の結果を表す。
type CloudFileDetailsResult struct {
	Exists    bool              `json:"exists"`
	TotalSize int64             `json:"totalSize"`
	Files     []CloudFileDetail `json:"files"`
}

// ListCloudData はクラウドデータ一覧を取得する。
func (app *App) ListCloudData() result.ApiResult[[]CloudDataItem] {
	ctx := app.context()
	client, bucket, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return errorResultWithLog[[]CloudDataItem](app, "クラウドデータ取得に失敗しました", error, "operation", "ListCloudData.getDefaultS3Client")
	}

	objects, error := storage.ListObjects(ctx, client, bucket, "")
	if error != nil {
		return errorResultWithLog[[]CloudDataItem](app, "クラウドデータ取得に失敗しました", error, "operation", "ListCloudData.listObjects", "bucket", bucket)
	}

	grouped := map[string]*CloudDataItem{}
	for _, obj := range objects {
		groupKey := detectGamePrefix(obj.Key)
		item, exists := grouped[groupKey]
		if !exists {
			item = &CloudDataItem{
				Name:       groupKey,
				RemotePath: groupKey,
			}
			grouped[groupKey] = item
		}
		item.FileCount++
		item.TotalSize += obj.Size
		if obj.LastModified > 0 {
			last := time.UnixMilli(obj.LastModified)
			if item.LastModified.IsZero() || last.After(item.LastModified) {
				item.LastModified = last
			}
		}
	}

	items := make([]CloudDataItem, 0, len(grouped))
	for _, item := range grouped {
		items = append(items, *item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return result.OkResult(items)
}

// GetDirectoryTree はクラウドのディレクトリツリーを取得する。
func (app *App) GetDirectoryTree() result.ApiResult[[]CloudDirectoryNode] {
	ctx := app.context()
	client, bucket, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return errorResultWithLog[[]CloudDirectoryNode](app, "ディレクトリツリー取得に失敗しました", error, "operation", "GetDirectoryTree.getDefaultS3Client")
	}
	objects, error := storage.ListObjects(ctx, client, bucket, "")
	if error != nil {
		return errorResultWithLog[[]CloudDirectoryNode](app, "ディレクトリツリー取得に失敗しました", error, "operation", "GetDirectoryTree.listObjects", "bucket", bucket)
	}

	root := map[string]*directoryNodeBuilder{}
	for _, obj := range objects {
		segments := strings.Split(obj.Key, "/")
		currentPath := ""
		currentMap := root
		for idx, seg := range segments {
			if seg == "" {
				continue
			}
			if currentPath == "" {
				currentPath = seg
			} else {
				currentPath = currentPath + "/" + seg
			}
			builder, exists := currentMap[seg]
			if !exists {
				builder = &directoryNodeBuilder{
					node: CloudDirectoryNode{
						Name:        seg,
						Path:        currentPath,
						IsDirectory: idx < len(segments)-1,
					},
					children: map[string]*directoryNodeBuilder{},
				}
				currentMap[seg] = builder
			}
			if idx == len(segments)-1 {
				builder.node.IsDirectory = false
				builder.node.Size = obj.Size
				if obj.LastModified > 0 {
					builder.node.LastModified = time.UnixMilli(obj.LastModified)
				}
				key := obj.Key
				builder.node.ObjectKey = &key
			}
			if builder.node.IsDirectory {
				if builder.children == nil {
					builder.children = map[string]*directoryNodeBuilder{}
				}
				currentMap = builder.children
			}
		}
	}

	return result.OkResult(flattenDirectoryBuilders(root))
}

// DeleteCloudData は指定パス配下を削除する。
func (app *App) DeleteCloudData(path string) result.ApiResult[bool] {
	ctx := app.context()
	client, bucket, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return errorResultWithLog[bool](app, "削除に失敗しました", error, "operation", "DeleteCloudData.getDefaultS3Client")
	}

	prefix := strings.TrimSpace(path)
	if prefix == "*" || prefix == "" {
		prefix = ""
	}
	if error := storage.DeleteObjectsByPrefix(ctx, client, bucket, prefix); error != nil {
		return errorResultWithLog[bool](app, "削除に失敗しました", error, "operation", "DeleteCloudData.deleteByPrefix", "prefix", prefix)
	}
	return result.OkResult(true)
}

// DeleteFile は単一ファイルを削除する。
func (app *App) DeleteFile(key string) result.ApiResult[bool] {
	ctx := app.context()
	client, bucket, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return errorResultWithLog[bool](app, "削除に失敗しました", error, "operation", "DeleteFile.getDefaultS3Client")
	}
	if error := storage.DeleteObject(ctx, client, bucket, key); error != nil {
		return errorResultWithLog[bool](app, "削除に失敗しました", error, "operation", "DeleteFile.deleteObject", "key", key)
	}
	return result.OkResult(true)
}

// GetCloudFileDetails はプレフィックス配下の詳細を取得する。
func (app *App) GetCloudFileDetails(prefix string) result.ApiResult[[]CloudFileDetail] {
	ctx := app.context()
	client, bucket, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return errorResultWithLog[[]CloudFileDetail](app, "詳細取得に失敗しました", error, "operation", "GetCloudFileDetails.getDefaultS3Client")
	}
	objects, error := storage.ListObjects(ctx, client, bucket, prefix)
	if error != nil {
		return errorResultWithLog[[]CloudFileDetail](app, "詳細取得に失敗しました", error, "operation", "GetCloudFileDetails.listObjects", "prefix", prefix)
	}
	files := make([]CloudFileDetail, 0, len(objects))
	for _, obj := range objects {
		relative := strings.TrimPrefix(obj.Key, prefix)
		relative = strings.TrimPrefix(relative, "/")
		files = append(files, CloudFileDetail{
			Name:         filepath.Base(obj.Key),
			Size:         obj.Size,
			LastModified: time.UnixMilli(obj.LastModified),
			Key:          obj.Key,
			RelativePath: relative,
		})
	}
	return result.OkResult(files)
}

// GetCloudFileDetailsByGame はゲームIDから詳細を取得する。
func (app *App) GetCloudFileDetailsByGame(gameID string) result.ApiResult[CloudFileDetailsResult] {
	ctx := app.context()
	game, error := app.Database.GetGameByID(ctx, gameID)
	if error != nil {
		return errorResultWithLog[CloudFileDetailsResult](app, "ゲーム取得に失敗しました", error, "operation", "GetCloudFileDetailsByGame.getGameByID", "gameId", gameID)
	}
	if game == nil {
		return result.OkResult(CloudFileDetailsResult{Exists: false, Files: []CloudFileDetail{}})
	}
	prefix := createGamePrefix(game.ID)
	client, bucket, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return errorResultWithLog[CloudFileDetailsResult](app, "詳細取得に失敗しました", error, "operation", "GetCloudFileDetailsByGame.getDefaultS3Client", "gameId", gameID)
	}
	objects, error := storage.ListObjects(ctx, client, bucket, prefix)
	if error != nil {
		return errorResultWithLog[CloudFileDetailsResult](app, "詳細取得に失敗しました", error, "operation", "GetCloudFileDetailsByGame.listObjects", "prefix", prefix)
	}
	files := make([]CloudFileDetail, 0, len(objects))
	var total int64
	for _, obj := range objects {
		files = append(files, CloudFileDetail{
			Name:         filepath.Base(obj.Key),
			Size:         obj.Size,
			LastModified: time.UnixMilli(obj.LastModified),
			Key:          obj.Key,
			RelativePath: strings.TrimPrefix(obj.Key, prefix),
		})
		total += obj.Size
	}
	return result.OkResult(CloudFileDetailsResult{Exists: len(files) > 0, TotalSize: total, Files: files})
}

// GetCloudSaveHash はクラウド上のセーブデータハッシュを取得する。
func (app *App) GetCloudSaveHash(gameID string) result.ApiResult[*storage.SaveHashMetadata] {
	trimmed := strings.TrimSpace(gameID)
	if trimmed == "" {
		app.Logger.Warn("ゲームIDが不正です", "operation", "GetCloudSaveHash", "gameId", gameID)
		return result.ErrorResult[*storage.SaveHashMetadata]("ゲームIDが不正です", "gameID is empty")
	}
	ctx := app.context()
	client, bucket, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return errorResultWithLog[*storage.SaveHashMetadata](app, "取得に失敗しました", error, "operation", "GetCloudSaveHash.getDefaultS3Client", "gameId", trimmed)
	}
	key := createSaveHashPath(trimmed)
	metadata, error := storage.LoadSaveHash(ctx, client, bucket, key)
	if error != nil {
		if storage.IsNotFoundError(error) {
			return result.OkResult[*storage.SaveHashMetadata](nil)
		}
		return errorResultWithLog[*storage.SaveHashMetadata](app, "取得に失敗しました", error, "operation", "GetCloudSaveHash.loadSaveHash", "key", key)
	}
	return result.OkResult(metadata)
}

// SaveCloudSaveHash はクラウドにセーブデータハッシュを保存する。
func (app *App) SaveCloudSaveHash(gameID string, hash string) result.ApiResult[bool] {
	trimmed := strings.TrimSpace(gameID)
	trimmedHash := strings.TrimSpace(hash)
	if trimmed == "" || trimmedHash == "" {
		app.Logger.Warn("入力が不正です", "operation", "SaveCloudSaveHash", "gameId", gameID)
		return result.ErrorResult[bool]("入力が不正です", "gameID or hash is empty")
	}
	ctx := app.context()
	client, bucket, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return errorResultWithLog[bool](app, "保存に失敗しました", error, "operation", "SaveCloudSaveHash.getDefaultS3Client", "gameId", trimmed)
	}
	key := createSaveHashPath(trimmed)
	metadata := storage.SaveHashMetadata{
		Hash:      trimmedHash,
		UpdatedAt: time.Now(),
	}
	if error := storage.SaveSaveHash(ctx, client, bucket, key, metadata); error != nil {
		return errorResultWithLog[bool](app, "保存に失敗しました", error, "operation", "SaveCloudSaveHash.saveSaveHash", "key", key)
	}
	return result.OkResult(true)
}

// DownloadSaveData はクラウドからダウンロードする。
func (app *App) DownloadSaveData(localPath string, remotePath string) result.ApiResult[bool] {
	ctx := app.context()
	client, bucket, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return errorResultWithLog[bool](app, "ダウンロードに失敗しました", error, "operation", "DownloadSaveData.getDefaultS3Client", "remotePath", remotePath)
	}
	if error := storage.DownloadPrefix(
		ctx,
		client,
		bucket,
		remotePath,
		localPath,
		app.Config.S3UploadConcurrency,
		app.Config.S3TransferRetryCount,
	); error != nil {
		return errorResultWithLog[bool](app, "ダウンロードに失敗しました", error, "operation", "DownloadSaveData.downloadPrefix", "remotePath", remotePath, "localPath", localPath)
	}
	return result.OkResult(true)
}

// LoadImageFromLocal はローカル画像をBase64で返す。
func (app *App) LoadImageFromLocal(filePath string) result.ApiResult[string] {
	content, error := os.ReadFile(filePath)
	if error != nil {
		return errorResultWithLog[string](app, "画像読み込みに失敗しました", error, "operation", "LoadImageFromLocal.readFile", "path", filePath)
	}
	encoded := base64.StdEncoding.EncodeToString(content)
	mime := detectImageMime(filePath)
	return result.OkResult("data:" + mime + ";base64," + encoded)
}

// ValidateCredential は認証情報の検証を行う。
func (app *App) ValidateCredential(input CredentialValidationInput) result.ApiResult[bool] {
	ctx := app.context()
	cfg := storage.S3Config{
		Endpoint:       input.Endpoint,
		Region:         input.Region,
		Bucket:         input.BucketName,
		ForcePathStyle: app.Config.S3ForcePathStyle,
		UseTLS:         app.Config.S3UseTLS,
	}
	client, error := storage.NewClient(ctx, cfg, credentials.Credential{
		AccessKeyID:     input.AccessKeyID,
		SecretAccessKey: input.SecretAccessKey,
	})
	if error != nil {
		return errorResultWithLog[bool](app, "認証情報検証に失敗しました", error, "operation", "ValidateCredential.newClient", "bucket", cfg.Bucket)
	}
	_, error = client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &cfg.Bucket})
	if error != nil {
		return errorResultWithLog[bool](app, "認証情報検証に失敗しました", error, "operation", "ValidateCredential.headBucket", "bucket", cfg.Bucket)
	}
	return result.OkResult(true)
}

// ValidateSavedCredential は保存済み認証情報の検証を行う。
func (app *App) ValidateSavedCredential(key string) result.ApiResult[bool] {
	ctx := app.context()
	cfg, credential, error := app.resolveS3Config(ctx)
	if error != nil {
		return errorResultWithLog[bool](app, "認証情報検証に失敗しました", error, "operation", "ValidateSavedCredential.resolveS3Config")
	}
	client, error := storage.NewClient(ctx, cfg, credential)
	if error != nil {
		return errorResultWithLog[bool](app, "認証情報検証に失敗しました", error, "operation", "ValidateSavedCredential.newClient", "bucket", cfg.Bucket)
	}
	_, error = client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &cfg.Bucket})
	if error != nil {
		return errorResultWithLog[bool](app, "認証情報検証に失敗しました", error, "operation", "ValidateSavedCredential.headBucket", "bucket", cfg.Bucket)
	}
	return result.OkResult(true)
}

// CredentialValidationInput は検証用の認証情報を表す。
type CredentialValidationInput struct {
	BucketName      string `json:"bucketName"`
	Region          string `json:"region"`
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
}

func (app *App) getDefaultS3Client(ctx context.Context) (*s3.Client, string, error) {
	cfg, credential, error := app.resolveS3Config(ctx)
	if error != nil {
		return nil, "", error
	}
	client, error := storage.NewClient(ctx, cfg, credential)
	if error != nil {
		return nil, "", error
	}
	return client, cfg.Bucket, nil
}

func (app *App) resolveS3Config(ctx context.Context) (storage.S3Config, credentials.Credential, error) {
	credResult := app.CredentialService.LoadCredential(ctx, "default")
	if !credResult.Success || credResult.Data == nil {
		return storage.S3Config{}, credentials.Credential{}, errors.New("認証情報がありません")
	}
	credential := *credResult.Data
	return storage.S3Config{
		Endpoint:       util.FirstNonEmpty(credential.Endpoint, app.Config.S3Endpoint),
		Region:         util.FirstNonEmpty(credential.Region, app.Config.S3Region),
		Bucket:         util.FirstNonEmpty(credential.BucketName, app.Config.S3Bucket),
		ForcePathStyle: app.Config.S3ForcePathStyle,
		UseTLS:         app.Config.S3UseTLS,
	}, credential, nil
}

func detectGamePrefix(key string) string {
	parts := strings.Split(key, "/")
	if len(parts) >= 2 && parts[0] == "games" {
		return parts[0] + "/" + parts[1]
	}
	if len(parts) > 0 {
		return parts[0]
	}
	return key
}

func flattenDirectoryBuilders(nodes map[string]*directoryNodeBuilder) []CloudDirectoryNode {
	result := make([]CloudDirectoryNode, 0, len(nodes))
	for _, node := range nodes {
		if len(node.children) > 0 {
			node.node.Children = flattenDirectoryBuilders(node.children)
		}
		result = append(result, node.node)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func createGamePrefix(gameID string) string {
	return "games/" + strings.TrimSpace(gameID) + "/"
}

func createSaveHashPath(gameID string) string {
	return "games/" + strings.TrimSpace(gameID) + "/save_hash.json"
}

func detectImageMime(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return "image/jpeg"
	}
}
