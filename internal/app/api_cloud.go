// クラウド関連のAPIを提供する。
package app

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"CloudLaunch_Go/internal/infrastructure/credentials"
	"CloudLaunch_Go/internal/infrastructure/storage"
	"CloudLaunch_Go/internal/result"
	"CloudLaunch_Go/internal/services"
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

// ListCloudData はクラウドデータ一覧（ゲーム単位の論理ビュー）を取得する。
func (app *App) ListCloudData() result.ApiResult[[]CloudDataItem] {
	ctx := app.context()
	views, err := app.ContentSyncService.ListCloudGameViews(ctx)
	if err != nil {
		return errorResultWithLog[[]CloudDataItem](app, "クラウドデータ取得に失敗しました", err, "operation", "ListCloudData.ListCloudGameViews")
	}

	items := make([]CloudDataItem, 0, len(views))
	for _, view := range views {
		items = append(items, CloudDataItem{
			Name:         view.Title,
			TotalSize:    view.TotalSize,
			FileCount:    int64(len(view.Files)),
			LastModified: view.LastModified,
			RemotePath:   view.GameID,
		})
	}
	return result.OkResult(items)
}

// CloudGameSummaryItem はクラウドデータ一覧の軽量サマリ要素を表す。
// FileCount / TotalSize は commit メタからのキャッシュで、旧 commit では 0。
type CloudGameSummaryItem struct {
	Name         string    `json:"name"`
	RemotePath   string    `json:"remotePath"`
	LastModified time.Time `json:"lastModified"`
	FileCount    int64     `json:"fileCount"`
	TotalSize    int64     `json:"totalSize"`
}

// ListCloudGameSummaries はクラウド上の全ゲームの軽量サマリ（タイトル一覧）を取得する。
// 初期表示用。ファイル数・サイズは含めず、各ゲームの詳細は GetGameDirectoryNode で遅延取得する。
func (app *App) ListCloudGameSummaries() result.ApiResult[[]CloudGameSummaryItem] {
	ctx := app.context()
	summaries, err := app.ContentSyncService.ListCloudGameSummaries(ctx)
	if err != nil {
		return errorResultWithLog[[]CloudGameSummaryItem](app, "クラウドデータ取得に失敗しました", err, "operation", "ListCloudGameSummaries.ListCloudGameSummaries")
	}

	items := make([]CloudGameSummaryItem, 0, len(summaries))
	for _, s := range summaries {
		items = append(items, CloudGameSummaryItem{
			Name:         s.Title,
			RemotePath:   s.GameID,
			LastModified: s.LastModified,
			FileCount:    s.FileCount,
			TotalSize:    s.TotalSize,
		})
	}
	return result.OkResult(items)
}

// GetGameDirectoryNode は1ゲームの論理ディレクトリツリー（ファイル一覧・サイズ付き）を取得する。
// クラウドデータ管理画面で対象ゲームを開いたときに遅延取得される。
func (app *App) GetGameDirectoryNode(gameID string) result.ApiResult[CloudDirectoryNode] {
	ctx := app.context()
	trimmed := strings.TrimSpace(gameID)
	if trimmed == "" {
		return result.ErrorResult[CloudDirectoryNode]("ゲームIDが不正です", "game id is empty")
	}
	view, err := app.ContentSyncService.GetCloudGameView(ctx, trimmed)
	if err != nil {
		return errorResultWithLog[CloudDirectoryNode](app, "ディレクトリツリー取得に失敗しました", err, "operation", "GetGameDirectoryNode.GetCloudGameView", "gameId", trimmed)
	}
	if view == nil {
		// クラウドにデータが無い（HEAD 未設定）。空の子を持つゲームノードを返す。
		return result.OkResult(CloudDirectoryNode{Name: trimmed, Path: trimmed, IsDirectory: true, Children: []CloudDirectoryNode{}})
	}
	node := buildGameDirectoryNode(*view)
	if node.Children == nil {
		node.Children = []CloudDirectoryNode{}
	}
	return result.OkResult(node)
}

// GetDirectoryTree はクラウドのディレクトリツリー（ゲーム単位の論理ビュー）を取得する。
func (app *App) GetDirectoryTree() result.ApiResult[[]CloudDirectoryNode] {
	ctx := app.context()
	views, err := app.ContentSyncService.ListCloudGameViews(ctx)
	if err != nil {
		return errorResultWithLog[[]CloudDirectoryNode](app, "ディレクトリツリー取得に失敗しました", err, "operation", "GetDirectoryTree.ListCloudGameViews")
	}

	nodes := make([]CloudDirectoryNode, 0, len(views))
	for _, view := range views {
		nodes = append(nodes, buildGameDirectoryNode(view))
	}
	return result.OkResult(nodes)
}

// buildGameDirectoryNode は1ゲームの論理ファイル一覧から階層ディレクトリツリーを構築する。
// トップノードはゲーム（IsDirectory=true, Path=gameID）で、配下にセーブファイルの階層を持つ。
func buildGameDirectoryNode(view services.CloudGameView) CloudDirectoryNode {
	root := &dirBuilder{
		node:     CloudDirectoryNode{Name: view.Title, Path: view.GameID, IsDirectory: true, LastModified: view.LastModified},
		children: map[string]*dirBuilder{},
	}
	for _, f := range view.Files {
		segments := strings.Split(f.RelPath, "/")
		cur := root
		curPath := view.GameID
		for idx, seg := range segments {
			if seg == "" {
				continue
			}
			curPath = curPath + "/" + seg
			isLeaf := idx == len(segments)-1
			child, ok := cur.children[seg]
			if !ok {
				child = &dirBuilder{
					node: CloudDirectoryNode{
						Name:        seg,
						Path:        curPath,
						IsDirectory: !isLeaf,
					},
					children: map[string]*dirBuilder{},
				}
				cur.children[seg] = child
			}
			if isLeaf {
				child.node.IsDirectory = false
				child.node.Size = f.Size
				child.node.LastModified = view.LastModified
			}
			cur = child
		}
	}
	return finalizeDirBuilder(root)
}

type dirBuilder struct {
	node     CloudDirectoryNode
	children map[string]*dirBuilder
}

// finalizeDirBuilder は dirBuilder を CloudDirectoryNode へ変換する。
// ディレクトリノードの Size には子の合計サイズを集約する（ObjectKey は設定しない）。
func finalizeDirBuilder(b *dirBuilder) CloudDirectoryNode {
	if len(b.children) == 0 {
		return b.node
	}
	names := make([]string, 0, len(b.children))
	for name := range b.children {
		names = append(names, name)
	}
	sort.Strings(names)
	children := make([]CloudDirectoryNode, 0, len(names))
	var total int64
	for _, name := range names {
		c := finalizeDirBuilder(b.children[name])
		total += c.Size
		children = append(children, c)
	}
	b.node.Children = children
	if b.node.IsDirectory {
		b.node.Size = total
	}
	return b.node
}

// DeleteCloudData は指定パス配下を削除する。
func (app *App) DeleteCloudData(path string) result.ApiResult[bool] {
	exactKey, childPrefix, ok := normalizeDeletePrefix(path)
	if !ok {
		return result.ErrorResult[bool]("削除対象のパスが不正です", "delete prefix is empty or wildcard")
	}

	ctx := app.context()
	client, bucket, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return errorResultWithLog[bool](app, "削除に失敗しました", error, "operation", "DeleteCloudData.getDefaultS3Client")
	}

	if error := storage.DeleteObject(ctx, client, bucket, exactKey); error != nil {
		return errorResultWithLog[bool](app, "削除に失敗しました", error, "operation", "DeleteCloudData.deleteExactObject", "key", exactKey)
	}
	if error := storage.DeleteObjectsByPrefix(ctx, client, bucket, childPrefix); error != nil {
		return errorResultWithLog[bool](app, "削除に失敗しました", error, "operation", "DeleteCloudData.deleteByPrefix", "prefix", childPrefix)
	}
	return result.OkResult(true)
}

// DeleteFile は単一ファイルを削除する。
func (app *App) DeleteFile(key string) result.ApiResult[bool] {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return result.ErrorResult[bool]("削除対象のファイルが不正です", "delete object key is empty")
	}

	ctx := app.context()
	client, bucket, error := app.getDefaultS3Client(ctx)
	if error != nil {
		return errorResultWithLog[bool](app, "削除に失敗しました", error, "operation", "DeleteFile.getDefaultS3Client")
	}
	if error := storage.DeleteObject(ctx, client, bucket, trimmed); error != nil {
		return errorResultWithLog[bool](app, "削除に失敗しました", error, "operation", "DeleteFile.deleteObject", "key", trimmed)
	}
	return result.OkResult(true)
}

// GetCloudFileDetails はプレフィックス（先頭セグメント=gameID、残り=サブパス）配下の
// 論理セーブファイル詳細を取得する。互換のため戻り型は []CloudFileDetail を維持する。
func (app *App) GetCloudFileDetails(prefix string) result.ApiResult[[]CloudFileDetail] {
	ctx := app.context()
	gameID, subPath := splitCloudPrefix(prefix)
	if gameID == "" {
		return result.OkResult([]CloudFileDetail{})
	}
	view, err := app.ContentSyncService.GetCloudGameView(ctx, gameID)
	if err != nil {
		return errorResultWithLog[[]CloudFileDetail](app, "詳細取得に失敗しました", err, "operation", "GetCloudFileDetails.GetCloudGameView", "gameId", gameID)
	}
	files := make([]CloudFileDetail, 0)
	if view != nil {
		for _, f := range view.Files {
			if subPath != "" && f.RelPath != subPath && !strings.HasPrefix(f.RelPath, subPath+"/") {
				continue
			}
			files = append(files, CloudFileDetail{
				Name:         path.Base(f.RelPath),
				Size:         f.Size,
				LastModified: view.LastModified,
				Key:          "",
				RelativePath: f.RelPath,
			})
		}
	}
	return result.OkResult(files)
}

// GetCloudFileDetailsByGame はゲームIDから論理セーブファイル詳細を取得する。
func (app *App) GetCloudFileDetailsByGame(gameID string) result.ApiResult[CloudFileDetailsResult] {
	ctx := app.context()
	view, err := app.ContentSyncService.GetCloudGameView(ctx, gameID)
	if err != nil {
		return errorResultWithLog[CloudFileDetailsResult](app, "詳細取得に失敗しました", err, "operation", "GetCloudFileDetailsByGame.GetCloudGameView", "gameId", gameID)
	}
	if view == nil {
		return result.OkResult(CloudFileDetailsResult{Exists: false, Files: []CloudFileDetail{}})
	}
	files := make([]CloudFileDetail, 0, len(view.Files))
	for _, f := range view.Files {
		files = append(files, CloudFileDetail{
			Name:         path.Base(f.RelPath),
			Size:         f.Size,
			LastModified: view.LastModified,
			Key:          "",
			RelativePath: f.RelPath,
		})
	}
	return result.OkResult(CloudFileDetailsResult{Exists: len(files) > 0, TotalSize: view.TotalSize, Files: files})
}

// splitCloudPrefix は "games/{gameID}/sub/path" or "{gameID}/sub/path" を gameID とサブパスに分解する。
func splitCloudPrefix(prefix string) (gameID, subPath string) {
	trimmed := strings.Trim(strings.TrimSpace(prefix), "/")
	trimmed = strings.TrimPrefix(trimmed, "games/")
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return "", ""
	}
	if idx := strings.Index(trimmed, "/"); idx >= 0 {
		return trimmed[:idx], trimmed[idx+1:]
	}
	return trimmed, ""
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
	credential, err := app.CredentialService.LoadCredential(ctx, "default")
	if err != nil || credential == nil {
		return storage.S3Config{}, credentials.Credential{}, errors.New("認証情報がありません")
	}
	return storage.S3Config{
		Endpoint:       util.FirstNonEmpty(credential.Endpoint, app.Config.S3Endpoint),
		Region:         util.FirstNonEmpty(credential.Region, app.Config.S3Region),
		Bucket:         util.FirstNonEmpty(credential.BucketName, app.Config.S3Bucket),
		ForcePathStyle: app.Config.S3ForcePathStyle,
		UseTLS:         app.Config.S3UseTLS,
	}, *credential, nil
}

func normalizeDeletePrefix(pathValue string) (exactKey string, childPrefix string, ok bool) {
	trimmed := strings.Trim(strings.TrimSpace(pathValue), "/")
	if trimmed == "" || trimmed == "*" {
		return "", "", false
	}
	return trimmed, trimmed + "/", true
}

func detectImageMime(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
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
