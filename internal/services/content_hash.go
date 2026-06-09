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

// removeFilesNotInSnapshot removes local save files that are absent from snapshot.
// It only deletes paths discovered by walking saveDir, then removes empty directories.
func removeFilesNotInSnapshot(saveDir string, snapshot domain.SaveSnapshot) error {
	var dirs []string
	err := filepath.Walk(saveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == saveDir {
			return nil
		}
		if info.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		rel, err := filepath.Rel(saveDir, path)
		if err != nil {
			return err
		}
		if _, ok := snapshot.Files[filepath.ToSlash(rel)]; ok {
			return nil
		}
		return os.Remove(path)
	})
	if err != nil {
		return err
	}

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
