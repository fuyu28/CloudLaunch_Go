package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"

	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/util"
)

func hashBytes(data []byte) domain.BlobHash {
	return util.Sha256Hex(data)
}

func hashFile(path string) (domain.BlobHash, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	return hashBytes(data), data, nil
}

// hashFileStream はファイルを逐次読みしながらハッシュのみを計算する（内容を RAM に保持しない）。
// 差分判定など、ハッシュだけ必要でファイル本体が不要な箇所に使う。
func hashFileStream(filePath string) (domain.BlobHash, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// validateSaveDir は saveDir が存在する通常のディレクトリであることを確認する。
func validateSaveDir(saveDir string) error {
	if saveDir == "" {
		return fmt.Errorf("セーブフォルダのパスが未設定です")
	}
	info, err := os.Stat(saveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("セーブフォルダが見つかりません: %s", saveDir)
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("セーブフォルダのパスがディレクトリではありません: %s", saveDir)
	}
	return nil
}

// walkSaveFiles は saveDir 配下の通常ファイルを走査し、各ファイルについて
// 絶対パスとスラッシュ区切りの相対パスで fn を呼ぶ。
// シンボリックリンクとディレクトリはスキップする（リンク先実体の漏洩・誤上書きを防ぐ）。
func walkSaveFiles(saveDir string, fn func(absPath, relPath string) error) error {
	return filepath.Walk(saveDir, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(saveDir, walkPath)
		if rerr != nil {
			return rerr
		}
		return fn(walkPath, filepath.ToSlash(rel))
	})
}

// buildSaveTree はセーブディレクトリを走査し、パス→ハッシュのみの SaveSnapshot を返す
// （ブロブ本体を RAM に保持しない）。状態確認や差分判定など、アップロード本体が不要な箇所に使う。
func buildSaveTree(saveDir string) (domain.SaveSnapshot, error) {
	if err := validateSaveDir(saveDir); err != nil {
		return domain.SaveSnapshot{}, err
	}
	files := make(map[string]domain.BlobHash)
	err := walkSaveFiles(saveDir, func(absPath, relPath string) error {
		hash, herr := hashFileStream(absPath)
		if herr != nil {
			return herr
		}
		files[relPath] = hash
		return nil
	})
	if err != nil {
		return domain.SaveSnapshot{}, err
	}
	return domain.SaveSnapshot{Files: files}, nil
}

// buildSaveSnapshot はセーブディレクトリを走査し SaveSnapshot とブロブマップを返す。
// saveFolderPath が未設定またはディレクトリが存在しない場合はエラー。
func buildSaveSnapshot(saveDir string) (domain.SaveSnapshot, map[domain.BlobHash][]byte, error) {
	if err := validateSaveDir(saveDir); err != nil {
		return domain.SaveSnapshot{}, nil, err
	}
	files := make(map[string]domain.BlobHash)
	blobs := make(map[domain.BlobHash][]byte)
	err := walkSaveFiles(saveDir, func(absPath, relPath string) error {
		hash, data, herr := hashFile(absPath)
		if herr != nil {
			return herr
		}
		files[relPath] = hash
		blobs[hash] = data
		return nil
	})
	if err != nil {
		return domain.SaveSnapshot{}, nil, err
	}
	return domain.SaveSnapshot{Files: files}, blobs, nil
}

// planDeletions は saveDir 配下でリモートスナップショットに存在しないファイルを洗い出し、
// baseTree（前回同期した SaveSnapshot のパス集合）に含まれていたか否かで二分する。
//
//   - tracked   : 以前は同期管理下にあったが新スナップショットから消えたファイル。
//     リモートで削除された等なので自動削除してよい。
//   - untracked : 同期が一度も認識していないローカル固有のファイル。
//     saveFolderPath の誤設定で混入した無関係ファイルもここに入る。確認なしで消さない。
//
// いずれも相対パス（スラッシュ区切り）を昇順で返す。この関数はファイルを削除しない。
func planDeletions(saveDir string, snapshot domain.SaveSnapshot, baseTree map[string]struct{}) (tracked, untracked []string, err error) {
	walkErr := walkSaveFiles(saveDir, func(_, relPath string) error {
		if _, ok := snapshot.Files[relPath]; ok {
			return nil // 新スナップショットに含まれる → 残す
		}
		if _, ok := baseTree[relPath]; ok {
			tracked = append(tracked, relPath)
		} else {
			untracked = append(untracked, relPath)
		}
		return nil
	})
	if walkErr != nil {
		return nil, nil, walkErr
	}
	sort.Strings(tracked)
	sort.Strings(untracked)
	return tracked, untracked, nil
}

// applyDeletions は指定された相対パス（スラッシュ区切り）のファイルを削除し、
// その削除によって空になった祖先ディレクトリのみを除去する。
// saveDir 全体は走査せず、ユーザーが元々置いていた無関係な空ディレクトリは残す。
func applyDeletions(saveDir string, relPaths []string) error {
	if len(relPaths) == 0 {
		return nil
	}
	// 削除したファイルの祖先ディレクトリを空チェック対象として収集する。
	dirsToCheck := make(map[string]struct{})
	for _, rel := range relPaths {
		// relPaths は planDeletions が saveDir 配下を走査して得たスラッシュ区切りパスなので安全。
		target := filepath.Join(saveDir, filepath.FromSlash(rel))
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return err
		}
		for d := path.Dir(rel); d != "." && d != "/"; d = path.Dir(d) {
			dirsToCheck[filepath.Join(saveDir, filepath.FromSlash(d))] = struct{}{}
		}
	}
	// 深い順に処理し、子が消えて空になった親も連鎖的に削除できるようにする。
	dirs := make([]string, 0, len(dirsToCheck))
	for d := range dirsToCheck {
		dirs = append(dirs, d)
	}
	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if len(entries) == 0 {
			if err := os.Remove(dir); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

// parseSaveTree は localSaveTree(JSON) をパスの集合へ変換する。空文字なら空集合を返す。
func parseSaveTree(treeJSON string) (map[string]struct{}, error) {
	set := make(map[string]struct{})
	if treeJSON == "" {
		return set, nil
	}
	var snap domain.SaveSnapshot
	if err := json.Unmarshal([]byte(treeJSON), &snap); err != nil {
		return nil, err
	}
	for relPath := range snap.Files {
		set[relPath] = struct{}{}
	}
	return set, nil
}

// logSamplePaths はログ出力用に最大 limit 件までのパスを返す。
// 超過分は "... (+N more)" に丸めてログの肥大化を防ぐ。
func logSamplePaths(paths []string, limit int) []string {
	if len(paths) <= limit {
		return paths
	}
	sample := make([]string, 0, limit+1)
	sample = append(sample, paths[:limit]...)
	sample = append(sample, fmt.Sprintf("... (+%d more)", len(paths)-limit))
	return sample
}

// CloudGameInfo はクラウドゲームメタ情報の API 公開型。
type CloudGameInfo struct {
	ID             string            `json:"id"`
	Title          string            `json:"title"`
	Publisher      string            `json:"publisher"`
	ImageHash      domain.BlobHash   `json:"imageHash,omitempty"`
	PlayStatus     domain.PlayStatus `json:"playStatus"`
	TotalPlayTime  int64             `json:"totalPlayTime"`
	LastPlayed     *time.Time        `json:"lastPlayed,omitempty"`
	ClearedAt      *time.Time        `json:"clearedAt,omitempty"`
	CurrentRouteID *string           `json:"currentRouteId,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

// cloudGame は game.json のクラウド保存フォーマット。
type cloudGame struct {
	ID             string            `json:"id"`
	Title          string            `json:"title"`
	Publisher      string            `json:"publisher"`
	ImageHash      domain.BlobHash   `json:"imageHash,omitempty"`
	PlayStatus     domain.PlayStatus `json:"playStatus"`
	TotalPlayTime  int64             `json:"totalPlayTime"`
	LastPlayed     *time.Time        `json:"lastPlayed,omitempty"`
	ClearedAt      *time.Time        `json:"clearedAt,omitempty"`
	CurrentRouteID *string           `json:"currentRouteId,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

// cloudSession は sessions.json のクラウド保存フォーマット。
type cloudSession struct {
	ID          string    `json:"id"`
	PlayedAt    time.Time `json:"playedAt"`
	Duration    int64     `json:"duration"`
	SessionName *string   `json:"sessionName,omitempty"`
	RouteID     *string   `json:"routeId,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// metaBuildResult は buildMetaSnapshot の戻り値。
type metaBuildResult struct {
	Snapshot      domain.MetaSnapshot
	SnapshotBytes []byte
	GameJSON      []byte
	SessionsJSON  []byte
}

// buildMetaSnapshot はゲーム情報・セッション・セーブハッシュから MetaSnapshot を構築する。
func buildMetaSnapshot(
	game domain.Game,
	sessions []domain.PlaySession,
	imageHash domain.BlobHash,
	savesHash domain.BlobHash,
	deviceName string,
) (metaBuildResult, error) {
	gameJSON, err := json.Marshal(cloudGame{
		ID:             game.ID,
		Title:          game.Title,
		Publisher:      game.Publisher,
		ImageHash:      imageHash,
		PlayStatus:     game.PlayStatus,
		TotalPlayTime:  game.TotalPlayTime,
		LastPlayed:     game.LastPlayed,
		ClearedAt:      game.ClearedAt,
		CurrentRouteID: game.CurrentRouteID,
		CreatedAt:      game.CreatedAt,
		UpdatedAt:      game.UpdatedAt,
	})
	if err != nil {
		return metaBuildResult{}, err
	}

	cs := make([]cloudSession, 0, len(sessions))
	for _, s := range sessions {
		cs = append(cs, cloudSession{
			ID:          s.ID,
			PlayedAt:    s.PlayedAt,
			Duration:    s.Duration,
			SessionName: s.SessionName,
			RouteID:     s.RouteID,
			UpdatedAt:   s.UpdatedAt,
		})
	}
	sessionsJSON, err := json.Marshal(cs)
	if err != nil {
		return metaBuildResult{}, err
	}

	meta := domain.MetaSnapshot{
		GameJSON:     hashBytes(gameJSON),
		SessionsJSON: hashBytes(sessionsJSON),
		Saves:        savesHash,
		DeviceName:   deviceName,
		CreatedAt:    time.Now().UTC(),
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return metaBuildResult{}, err
	}

	return metaBuildResult{
		Snapshot:      meta,
		SnapshotBytes: metaBytes,
		GameJSON:      gameJSON,
		SessionsJSON:  sessionsJSON,
	}, nil
}
