// Pull ステージング用のファイルシステム操作抽象（テスト差し替え用）。
package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"CloudLaunch_Go/internal/domain"
)

// pullFileOps は Pull の stage/backup 交換で使う FS 操作。
// Windows のファイルロック rename 失敗などをサービス層テストで決定的に再現するために差し替える。
type pullFileOps interface {
	MkdirAll(path string, perm os.FileMode) error
	RemoveAll(path string) error
	Rename(oldpath, newpath string) error
	Stat(name string) (os.FileInfo, error)
	Open(name string) (*os.File, error)
	Create(name string) (*os.File, error)
}

type osPullFileOps struct{}

func (osPullFileOps) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }
func (osPullFileOps) RemoveAll(path string) error                  { return os.RemoveAll(path) }
func (osPullFileOps) Rename(oldpath, newpath string) error         { return os.Rename(oldpath, newpath) }
func (osPullFileOps) Stat(name string) (os.FileInfo, error)        { return os.Stat(name) }
func (osPullFileOps) Open(name string) (*os.File, error)           { return os.Open(name) }
func (osPullFileOps) Create(name string) (*os.File, error)         { return os.Create(name) }

func pathExists(ops pullFileOps, path string) (bool, error) {
	_, err := ops.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func copyFileViaOps(ops pullFileOps, src, dst string) error {
	if err := ops.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	in, err := ops.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := ops.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func saveSnapshotsEqual(a, b domain.SaveSnapshot) bool {
	if len(a.Files) != len(b.Files) {
		return false
	}
	for k, v := range a.Files {
		if b.Files[k] != v {
			return false
		}
	}
	return true
}

// ensureEmptyDir は path が存在すれば中身ごと消し、空ディレクトリを作り直す。
func ensureEmptyDir(ops pullFileOps, path string) error {
	if err := ops.RemoveAll(path); err != nil {
		return fmt.Errorf("remove existing path: %w", err)
	}
	return ops.MkdirAll(path, 0o700)
}
