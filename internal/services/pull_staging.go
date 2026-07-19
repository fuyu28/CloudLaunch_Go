// Pull の同ボリューム stage/backup 交換とジャーナル回復。
package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/infrastructure/storage"
)

// ErrSaveDirChangedDuringPull は stage 構築後・swap 直前に live が外部変更されたときのセンチネル。
var ErrSaveDirChangedDuringPull = errors.New("セーブフォルダが Pull 準備中に変更されました。再実行してください")

func newPullOperationID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func pullSiblingPaths(liveDir, operationID string) (stagePath, backupPath string) {
	parent := filepath.Dir(liveDir)
	stagePath = filepath.Join(parent, ".cloudlaunch-stage-"+operationID)
	backupPath = filepath.Join(parent, ".cloudlaunch-backup-"+operationID)
	return stagePath, backupPath
}

// readLiveSaveTree は live が無ければ空スナップショットを返す。
func readLiveSaveTree(liveDir string) (domain.SaveSnapshot, error) {
	info, err := os.Stat(liveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.SaveSnapshot{Files: map[string]domain.BlobHash{}}, nil
		}
		return domain.SaveSnapshot{}, err
	}
	if !info.IsDir() {
		return domain.SaveSnapshot{}, fmt.Errorf("セーブフォルダのパスがディレクトリではありません: %s", liveDir)
	}
	return buildSaveTree(liveDir)
}

// pullApplySavesViaStaging は desired tree を stage に構築し、PREPARED ジャーナル後に
// live↔backup↔stage を rename で交換する。戻り値の OperationID は DB 反映時に APPLIED 化する。
//
// 承認済み untracked / tracked 削除は stage に載せないことで表現する（live を直接消さない）。
func (s *ContentSyncService) pullApplySavesViaStaging(
	ctx context.Context,
	bstore contentBlobStore,
	gameID string,
	onProgress ProgressFunc,
	saveDir string,
	saveSnap domain.SaveSnapshot,
	commitHash string,
) (op domain.PullOperation, err error) {
	ops := s.fileOps
	if ops == nil {
		ops = osPullFileOps{}
	}

	absLive, err := filepath.Abs(saveDir)
	if err != nil {
		return domain.PullOperation{}, err
	}
	saveDir = absLive

	preLive, err := readLiveSaveTree(saveDir)
	if err != nil {
		return domain.PullOperation{}, err
	}
	hadLive, err := pathExists(ops, saveDir)
	if err != nil {
		return domain.PullOperation{}, err
	}

	opID, err := newPullOperationID()
	if err != nil {
		return domain.PullOperation{}, fmt.Errorf("operation id: %w", err)
	}
	stagePath, backupPath := pullSiblingPaths(saveDir, opID)

	if err := s.buildPullStage(ctx, ops, bstore, gameID, onProgress, saveDir, stagePath, saveSnap, preLive); err != nil {
		_ = ops.RemoveAll(stagePath)
		return domain.PullOperation{}, err
	}

	// swap 直前の外部書き込み検出。ジャーナル前に判定し、失敗時は stage だけ捨てる。
	currentLive, err := readLiveSaveTree(saveDir)
	if err != nil {
		_ = ops.RemoveAll(stagePath)
		return domain.PullOperation{}, err
	}
	if !saveSnapshotsEqual(preLive, currentLive) {
		_ = ops.RemoveAll(stagePath)
		return domain.PullOperation{}, ErrSaveDirChangedDuringPull
	}

	op = domain.PullOperation{
		OperationID: opID,
		GameID:      gameID,
		LivePath:    saveDir,
		StagePath:   stagePath,
		BackupPath:  backupPath,
		CommitHash:  commitHash,
		Status:      domain.PullOperationPrepared,
		HadLive:     hadLive,
	}
	if err := s.repository.BeginPullOperation(ctx, op); err != nil {
		_ = ops.RemoveAll(stagePath)
		return domain.PullOperation{}, err
	}

	if err := s.swapLiveWithStage(ops, saveDir, stagePath, backupPath); err != nil {
		_ = s.rollbackPreparedPullDisk(ops, op)
		if clearErr := s.repository.ClearPullOperation(ctx, opID); clearErr != nil {
			s.logger.Warn("Pull ジャーナルのクリアに失敗", "operationId", opID, "error", clearErr)
		}
		return domain.PullOperation{}, err
	}

	return op, nil
}

func (s *ContentSyncService) buildPullStage(
	ctx context.Context,
	ops pullFileOps,
	bstore contentBlobStore,
	gameID string,
	onProgress ProgressFunc,
	liveDir, stagePath string,
	saveSnap domain.SaveSnapshot,
	preLive domain.SaveSnapshot,
) error {
	if err := ensureEmptyDir(ops, stagePath); err != nil {
		return err
	}

	total := len(saveSnap.Files)
	needsDownload := make(map[string]string, total)
	copied := 0

	for relPath, wantHash := range saveSnap.Files {
		stageFile, err := storage.ResolveSafeRelativePath(stagePath, relPath)
		if err != nil {
			return err
		}
		// live 側も安全パスとして解決できること（相対パス検証）。存在しなくてもよい。
		liveFile, err := storage.ResolveSafeRelativePath(liveDir, relPath)
		if err != nil {
			return err
		}

		if localHash, ok := preLive.Files[relPath]; ok && localHash == wantHash {
			if err := copyFileViaOps(ops, liveFile, stageFile); err != nil {
				return fmt.Errorf("copy unchanged %s: %w", relPath, err)
			}
			got, herr := hashFileStream(stageFile)
			if herr != nil {
				return herr
			}
			if got != wantHash {
				return fmt.Errorf("stage hash mismatch after copy: %s", relPath)
			}
			copied++
			continue
		}
		needsDownload[relPath] = wantHash
	}

	var wrappedProgress func(int, int)
	if onProgress != nil {
		alreadyDone := copied
		onProgress(alreadyDone, total)
		wrappedProgress = func(downloaded, _ int) {
			onProgress(alreadyDone+downloaded, total)
		}
	}
	if err := bstore.downloadBlobs(ctx, gameID, stagePath, needsDownload, s.config.S3UploadConcurrency, wrappedProgress); err != nil {
		return err
	}

	return validateStageMatchesSnapshot(stagePath, saveSnap)
}

func validateStageMatchesSnapshot(stagePath string, saveSnap domain.SaveSnapshot) error {
	for relPath, wantHash := range saveSnap.Files {
		target, err := storage.ResolveSafeRelativePath(stagePath, relPath)
		if err != nil {
			return err
		}
		got, err := hashFileStream(target)
		if err != nil {
			return fmt.Errorf("stage missing or unreadable %s: %w", relPath, err)
		}
		if got != wantHash {
			return fmt.Errorf("stage hash mismatch: %s", relPath)
		}
	}
	// stage に余分なファイルが無いことも確認（desired tree の完全一致）。
	built, err := buildSaveTree(stagePath)
	if err != nil {
		return err
	}
	if !saveSnapshotsEqual(built, saveSnap) {
		return fmt.Errorf("stage tree does not match remote snapshot")
	}
	return nil
}

func (s *ContentSyncService) swapLiveWithStage(ops pullFileOps, liveDir, stagePath, backupPath string) error {
	liveExists, err := pathExists(ops, liveDir)
	if err != nil {
		return err
	}
	if liveExists {
		if err := ops.Rename(liveDir, backupPath); err != nil {
			return fmt.Errorf("live→backup rename: %w", err)
		}
	}
	if err := ops.Rename(stagePath, liveDir); err != nil {
		// stage→live 失敗時は backup があれば live を復元してから返す。
		if liveExists {
			if renErr := ops.Rename(backupPath, liveDir); renErr != nil {
				return fmt.Errorf("stage→live rename: %v (restore also failed: %w)", err, renErr)
			}
		}
		return fmt.Errorf("stage→live rename: %w", err)
	}
	return nil
}

// rollbackPreparedPullDisk は PREPARED（DB 未反映）状態のディスクを旧 live に戻す。
//
// rename 途中の存在組み合わせ:
//   - backup あり → 少なくとも live→backup 済み。backup を live に戻し stage/新 live を捨てる
//   - backup なし・stage あり → 未 swap。stage だけ捨てる
//   - backup なし・stage なし・HadLive=false → stage→live 済みで旧 live 無し。新 live を捨てる
func (s *ContentSyncService) rollbackPreparedPullDisk(ops pullFileOps, op domain.PullOperation) error {
	if ops == nil {
		ops = osPullFileOps{}
	}
	backupExists, err := pathExists(ops, op.BackupPath)
	if err != nil {
		return err
	}
	liveExists, err := pathExists(ops, op.LivePath)
	if err != nil {
		return err
	}
	stageExists, err := pathExists(ops, op.StagePath)
	if err != nil {
		return err
	}

	if backupExists {
		if liveExists {
			if err := ops.RemoveAll(op.LivePath); err != nil {
				return fmt.Errorf("remove new live during rollback: %w", err)
			}
		}
		if err := ops.Rename(op.BackupPath, op.LivePath); err != nil {
			return fmt.Errorf("backup→live rename during rollback: %w", err)
		}
		if stageExists {
			if err := ops.RemoveAll(op.StagePath); err != nil {
				return fmt.Errorf("remove stage during rollback: %w", err)
			}
		}
		return nil
	}

	if stageExists {
		if err := ops.RemoveAll(op.StagePath); err != nil {
			return fmt.Errorf("remove stage during rollback: %w", err)
		}
		return nil
	}

	if !op.HadLive && liveExists {
		if err := ops.RemoveAll(op.LivePath); err != nil {
			return fmt.Errorf("remove uncommitted live during rollback: %w", err)
		}
	}
	return nil
}

func (s *ContentSyncService) finishAppliedPullDisk(ops pullFileOps, op domain.PullOperation) error {
	if ops == nil {
		ops = osPullFileOps{}
	}
	if err := ops.RemoveAll(op.BackupPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := ops.RemoveAll(op.StagePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// RecoverPullOperations は未完了 Pull ジャーナルを決定的に回復する。
// API / ランタイム開始前に呼ぶ想定。同一ゲームの同期と直列化する。
func (s *ContentSyncService) RecoverPullOperations(ctx context.Context) error {
	ops, err := s.repository.ListPullOperations(ctx)
	if err != nil {
		return err
	}
	var firstErr error
	for _, op := range ops {
		if err := s.recoverOnePullOperation(ctx, op); err != nil {
			s.logger.Warn("Pull ジャーナルの回復に失敗",
				"operationId", op.OperationID, "gameId", op.GameID, "status", op.Status, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (s *ContentSyncService) recoverOnePullOperation(ctx context.Context, op domain.PullOperation) error {
	defer s.lockGame(op.GameID)()
	return s.recoverOnePullOperationUnlocked(ctx, op)
}

// recoverPullOperationsForGame は指定ゲームの未完了ジャーナルだけを回復する。
// 呼び出し側が既に game lock を保持している前提（Push/Pull/Status 入口）。
func (s *ContentSyncService) recoverPullOperationsForGame(ctx context.Context, gameID string) error {
	ops, err := s.repository.ListPullOperations(ctx)
	if err != nil {
		return err
	}
	for _, op := range ops {
		if op.GameID != gameID {
			continue
		}
		if err := s.recoverOnePullOperationUnlocked(ctx, op); err != nil {
			return err
		}
	}
	return nil
}

func (s *ContentSyncService) recoverOnePullOperationUnlocked(ctx context.Context, op domain.PullOperation) error {
	ops := s.fileOps
	if ops == nil {
		ops = osPullFileOps{}
	}

	switch op.Status {
	case domain.PullOperationApplied:
		if err := s.finishAppliedPullDisk(ops, op); err != nil {
			return err
		}
		return s.repository.ClearPullOperation(ctx, op.OperationID)
	case domain.PullOperationPrepared, "":
		if err := s.rollbackPreparedPullDisk(ops, op); err != nil {
			return err
		}
		return s.repository.ClearPullOperation(ctx, op.OperationID)
	default:
		return fmt.Errorf("unknown pull operation status: %s", op.Status)
	}
}

// restoreLiveAfterDBFailure は swap 済み・DB 失敗時に backup を即時復元しジャーナルを消す。
func (s *ContentSyncService) restoreLiveAfterDBFailure(ctx context.Context, op domain.PullOperation) error {
	ops := s.fileOps
	if ops == nil {
		ops = osPullFileOps{}
	}
	diskErr := s.rollbackPreparedPullDisk(ops, op)
	clearErr := s.repository.ClearPullOperation(ctx, op.OperationID)
	if diskErr != nil {
		return diskErr
	}
	return clearErr
}
