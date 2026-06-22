package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"CloudLaunch_Go/internal/domain"
)

func hashBytes(data []byte) domain.BlobHash {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hashFile(path string) (domain.BlobHash, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	return hashBytes(data), data, nil
}

// buildSaveSnapshot はセーブディレクトリを走査し SaveSnapshot とブロブマップを返す。
// saveFolderPath が未設定またはディレクトリが存在しない場合はエラー。
func buildSaveSnapshot(saveDir string) (domain.SaveSnapshot, map[domain.BlobHash][]byte, error) {
	if saveDir == "" {
		return domain.SaveSnapshot{}, nil, fmt.Errorf("セーブフォルダのパスが未設定です")
	}
	info, err := os.Stat(saveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.SaveSnapshot{}, nil, fmt.Errorf("セーブフォルダが見つかりません: %s", saveDir)
		}
		return domain.SaveSnapshot{}, nil, err
	}
	if !info.IsDir() {
		return domain.SaveSnapshot{}, nil, fmt.Errorf("セーブフォルダのパスがディレクトリではありません: %s", saveDir)
	}

	files := make(map[string]domain.BlobHash)
	blobs := make(map[domain.BlobHash][]byte)

	err = filepath.Walk(saveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		hash, data, err := hashFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(saveDir, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = hash
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
	walkErr := filepath.Walk(saveDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(saveDir, path)
		if relErr != nil {
			return relErr
		}
		slashRel := filepath.ToSlash(rel)
		if _, ok := snapshot.Files[slashRel]; ok {
			return nil // 新スナップショットに含まれる → 残す
		}
		if _, ok := baseTree[slashRel]; ok {
			tracked = append(tracked, slashRel)
		} else {
			untracked = append(untracked, slashRel)
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
// その後 saveDir 配下の空ディレクトリを除去する。
func applyDeletions(saveDir string, relPaths []string) error {
	for _, rel := range relPaths {
		// relPaths は planDeletions が saveDir 配下を走査して得たパスなので安全。
		target := filepath.Join(saveDir, filepath.FromSlash(rel))
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return pruneEmptyDirs(saveDir)
}

// pruneEmptyDirs は saveDir 配下の空ディレクトリを削除する（saveDir 自身は残す）。
func pruneEmptyDirs(saveDir string) error {
	var dirs []string
	err := filepath.Walk(saveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != saveDir {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	// 深い階層から削除するため長いパス順に並べる。
	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})
	for _, dir := range dirs {
		if err := os.Remove(dir); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			entries, readErr := os.ReadDir(dir)
			if readErr == nil && len(entries) > 0 {
				continue
			}
			return err
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
