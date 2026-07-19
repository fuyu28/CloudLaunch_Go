// ゲームの実行プロセス監視と自動プレイ時間計測を提供する。
// プロセス列挙の OS 固有実装は process_provider_*.go 側に置く。
package services

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"CloudLaunch_Go/internal/domain"
	"CloudLaunch_Go/internal/logging"

	"golang.org/x/text/unicode/norm"
)

// MonitoringGame は監視対象のゲーム情報を保持する。
type MonitoringGame struct {
	GameID          string
	GameTitle       string
	ExePath         string
	ExeName         string
	LastDetected    *time.Time
	PlayStartTime   *time.Time
	AccumulatedTime int64
	LastNotFound    *time.Time
	IsPaused        bool
	PausedAt        *time.Time
	PendingEnd      bool
	PendingResume   bool
	SuppressResume  bool
}

// ProcessInfo はプロセス情報を保持する。
type ProcessInfo struct {
	Name string
	Pid  int
	Cmd  string
}

type normalizedProcess struct {
	info          ProcessInfo
	normalized    string
	normalizedCmd string
}

// normalizeProcessList は名前が空のプロセスを除外し、正規化済みプロセス一覧を構築する。
// ロックは保持しない（純粋関数）。
func normalizeProcessList(processes []ProcessInfo) []normalizedProcess {
	normalizedProcesses := make([]normalizedProcess, 0, len(processes))
	for _, proc := range processes {
		if proc.Name == "" {
			continue
		}
		normalizedProcesses = append(normalizedProcesses, normalizedProcess{
			info:          proc,
			normalized:    normalizeProcessToken(proc.Name),
			normalizedCmd: normalizeProcessPathToken(proc.Cmd),
		})
	}
	return normalizedProcesses
}

// afterPlaySyncer はプレイ終了後の自動 Push を抽象化するインターフェース。
type afterPlaySyncer interface {
	Push(ctx context.Context, gameID string, onProgress ProgressFunc) error
}

// ProcessMonitorService はゲームプロセス監視を提供する。
type ProcessMonitorService struct {
	repository         ProcessMonitorRepository
	logger             *slog.Logger
	cloudSync          afterPlaySyncer
	processProvider    func() ([]ProcessInfo, string)
	monitoredGames     map[string]*MonitoringGame
	autoTracking       bool
	monitoringInterval *time.Ticker
	monitoringStop     chan struct{}
	mu                 sync.Mutex
	interval           time.Duration
	sessionTimeout     time.Duration
	gameCleanupTimeout time.Duration
	// lastProcesses は直近の非空プロセス一覧スナップショット（service.mu で保護）。
	// 監視ループが定期更新するため、ホットキー撮影時の再列挙をほぼ不要にする。
	lastProcesses   []ProcessInfo
	lastProcessesAt time.Time
}

// NewProcessMonitorService は ProcessMonitorService を生成する。
func NewProcessMonitorService(repository ProcessMonitorRepository, logger *slog.Logger, cloudSync afterPlaySyncer) *ProcessMonitorService {
	service := &ProcessMonitorService{
		repository:         repository,
		logger:             logger,
		cloudSync:          cloudSync,
		monitoredGames:     make(map[string]*MonitoringGame),
		autoTracking:       true,
		interval:           2 * time.Second,
		sessionTimeout:     0,
		gameCleanupTimeout: 20 * time.Second,
	}
	// プラットフォーム既定の列挙。テストは processProvider を差し替えて注入する。
	service.processProvider = defaultProcessProvider(logger)
	return service
}

// StartMonitoring は監視を開始する。
func (service *ProcessMonitorService) StartMonitoring() {
	service.mu.Lock()
	if service.monitoringInterval != nil {
		service.mu.Unlock()
		return
	}
	service.monitoringStop = make(chan struct{})
	service.monitoringInterval = time.NewTicker(service.interval)
	service.mu.Unlock()

	service.logger.Info("プロセス監視を開始しました")

	go func() {
		// 1反復ごとに panic を回収する。1回のチェックで panic しても監視ループ自体は
		// 継続させ、エラーをログ（error.log）に残す。
		tick := func() {
			defer logging.Recover(service.logger, "process-monitor.checkProcesses")
			service.checkProcesses()
		}
		tick()
		for {
			select {
			case <-service.monitoringInterval.C:
				tick()
			case <-service.monitoringStop:
				return
			}
		}
	}()
}

// StopMonitoring は監視を停止する。
func (service *ProcessMonitorService) StopMonitoring() {
	service.mu.Lock()
	if service.monitoringInterval == nil {
		service.mu.Unlock()
		return
	}
	service.monitoringInterval.Stop()
	close(service.monitoringStop)
	service.monitoringInterval = nil
	service.monitoringStop = nil
	service.mu.Unlock()

	service.saveAllActiveSessions()
	service.logger.Info("プロセス監視を停止しました")
}

// IsMonitoring は監視中かどうかを返す。
func (service *ProcessMonitorService) IsMonitoring() bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	return service.monitoringInterval != nil
}

// UpdateAutoTracking は自動ゲーム検出設定を更新する。
func (service *ProcessMonitorService) UpdateAutoTracking(enabled bool) {
	service.mu.Lock()
	service.autoTracking = enabled
	service.mu.Unlock()

	if enabled && service.IsMonitoring() {
		service.checkProcesses()
	}
}

// GetMonitoringStatus は監視状態を返す。
func (service *ProcessMonitorService) GetMonitoringStatus() []domain.MonitoringGameStatus {
	service.mu.Lock()
	defer service.mu.Unlock()

	status := make([]domain.MonitoringGameStatus, 0, len(service.monitoredGames))
	now := time.Now()
	for _, game := range service.monitoredGames {
		playTime := game.AccumulatedTime
		if game.PlayStartTime != nil && !game.IsPaused && !game.PendingEnd {
			playTime += int64(now.Sub(*game.PlayStartTime).Seconds())
		}
		status = append(status, domain.MonitoringGameStatus{
			GameID:            game.GameID,
			GameTitle:         game.GameTitle,
			ExeName:           game.ExeName,
			IsPlaying:         game.PlayStartTime != nil && !game.IsPaused && !game.PendingEnd,
			PlayTime:          playTime,
			IsPaused:          game.IsPaused,
			NeedsConfirmation: game.PendingEnd,
			NeedsResume:       game.PendingResume,
		})
	}
	return status
}

// GetHotkeyTargetGameID はホットキー撮影時に保存先とするゲームIDを返す。
// 監視中ゲームのうち「現在プレイ中」を優先し、なければ「中断中」を返す。
func (service *ProcessMonitorService) GetHotkeyTargetGameID() string {
	service.mu.Lock()
	defer service.mu.Unlock()

	var bestPlaying *MonitoringGame
	var bestPaused *MonitoringGame
	for _, game := range service.monitoredGames {
		if game == nil {
			continue
		}
		if game.PendingEnd {
			continue
		}

		isPlaying := game.PlayStartTime != nil && !game.IsPaused
		if isPlaying {
			if bestPlaying == nil || isLaterGameActivity(game, bestPlaying) {
				bestPlaying = game
			}
			continue
		}

		if game.IsPaused {
			if bestPaused == nil || isLaterGameActivity(game, bestPaused) {
				bestPaused = game
			}
		}
	}

	if bestPlaying != nil {
		return bestPlaying.GameID
	}
	if bestPaused != nil {
		return bestPaused.GameID
	}
	return ""
}

func isLaterGameActivity(left *MonitoringGame, right *MonitoringGame) bool {
	leftTime := latestGameActivityAt(left)
	rightTime := latestGameActivityAt(right)
	if leftTime == nil {
		return false
	}
	if rightTime == nil {
		return true
	}
	return leftTime.After(*rightTime)
}

func latestGameActivityAt(game *MonitoringGame) *time.Time {
	if game == nil {
		return nil
	}
	if game.LastDetected != nil {
		return game.LastDetected
	}
	if game.PlayStartTime != nil {
		return game.PlayStartTime
	}
	if game.PausedAt != nil {
		return game.PausedAt
	}
	return nil
}

// GetProcessSnapshot は現在のプロセス一覧と正規化後の値を取得する。
func (service *ProcessMonitorService) GetProcessSnapshot() domain.ProcessSnapshot {
	processes, source := service.getProcesses()

	items := make([]domain.ProcessSnapshotItem, 0, len(processes))
	for _, proc := range processes {
		items = append(items, domain.ProcessSnapshotItem{
			Name:           proc.Name,
			Pid:            proc.Pid,
			Cmd:            proc.Cmd,
			NormalizedName: normalizeProcessToken(proc.Name),
			NormalizedCmd:  normalizeProcessPathToken(proc.Cmd),
		})
	}

	return domain.ProcessSnapshot{
		Source: source,
		Items:  items,
	}
}

func (service *ProcessMonitorService) addMonitoredGame(gameID string, title string, exePath string) {
	exeName := windowsPathBase(exePath)
	service.monitoredGames[gameID] = &MonitoringGame{
		GameID:          gameID,
		GameTitle:       title,
		ExePath:         exePath,
		ExeName:         exeName,
		AccumulatedTime: 0,
	}
	service.logger.Info("ゲーム監視を追加", "title", title, "exeName", exeName, "gameId", gameID)
}

func (service *ProcessMonitorService) removeMonitoredGame(gameID string) {
	game, exists := service.monitoredGames[gameID]
	if !exists {
		return
	}
	if game.PlayStartTime != nil {
		now := time.Now()
		duration := int64(now.Sub(*game.PlayStartTime).Seconds())
		if duration > 0 {
			game.AccumulatedTime += duration
			service.saveSession(*game, now)
		}
	}
	delete(service.monitoredGames, gameID)
	service.logger.Info("ゲーム監視を削除", "exeName", game.ExeName, "gameId", gameID)
}

// PauseSession はセッションを中断状態にする。
func (service *ProcessMonitorService) PauseSession(gameID string) bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	game, exists := service.monitoredGames[gameID]
	if !exists {
		return false
	}
	now := time.Now()
	if game.PlayStartTime != nil && !game.IsPaused {
		game.AccumulatedTime += int64(now.Sub(*game.PlayStartTime).Seconds())
	}
	game.PlayStartTime = nil
	game.IsPaused = true
	game.PendingEnd = false
	game.PendingResume = false
	game.SuppressResume = true
	game.PausedAt = &now
	return true
}

// ResumeSession は中断状態のセッションを再開する。
func (service *ProcessMonitorService) ResumeSession(gameID string) bool {
	processes, _ := service.getProcesses()
	normalizedProcesses := normalizeProcessList(processes)

	service.mu.Lock()
	defer service.mu.Unlock()
	game, exists := service.monitoredGames[gameID]
	if !exists {
		return false
	}
	if !service.isGameProcessRunning(game.ExeName, game.ExePath, normalizedProcesses) {
		return false
	}
	now := time.Now()
	game.IsPaused = false
	game.PendingEnd = false
	game.PendingResume = false
	game.SuppressResume = false
	game.PausedAt = nil
	game.PlayStartTime = &now
	game.LastDetected = &now
	return true
}

// EndSession は現在のセッションを終了して保存する。
func (service *ProcessMonitorService) EndSession(gameID string) bool {
	service.mu.Lock()
	game, exists := service.monitoredGames[gameID]
	if !exists {
		service.mu.Unlock()
		return false
	}
	now := time.Now()
	if game.PlayStartTime != nil {
		game.AccumulatedTime += int64(now.Sub(*game.PlayStartTime).Seconds())
	}
	game.PlayStartTime = nil
	game.IsPaused = false
	game.PendingEnd = false
	game.PendingResume = false
	game.SuppressResume = false
	game.PausedAt = nil
	accumulated := game.AccumulatedTime
	game.AccumulatedTime = 0
	game.LastNotFound = &now
	// 値コピーをロック内で確定させ、saveSession に渡す。共有 *MonitoringGame を
	// ロック外で書き換えると checkProcesses 側との data race になる（accumulated を
	// 一時的に書き戻してから再 Lock で 0 戻し、というかつての二重書きが原因だった）。
	snapshot := *game
	snapshot.AccumulatedTime = accumulated
	service.mu.Unlock()

	if accumulated > 0 {
		service.saveSession(snapshot, now)
	}
	return true
}

func (service *ProcessMonitorService) checkProcesses() {
	processes, _ := service.getProcesses()

	normalizedProcesses := normalizeProcessList(processes)

	service.autoAddGamesFromDatabase(processes, normalizedProcesses)

	processMap := make(map[string][]normalizedProcess)
	for _, proc := range normalizedProcesses {
		processMap[proc.normalized] = append(processMap[proc.normalized], proc)
	}

	now := time.Now()
	type pendingSession struct {
		Game    MonitoringGame
		EndedAt time.Time
	}
	sessionsToSave := make([]pendingSession, 0)
	gameIDsToCleanup := make([]string, 0)

	service.mu.Lock()
	for _, game := range service.monitoredGames {
		service.updateMonitoredGameState(game, processMap, now)
	}
	gameIDsToCleanup = service.collectGameIDsToCleanup(now, gameIDsToCleanup)
	service.mu.Unlock()

	for _, session := range sessionsToSave {
		service.saveSession(session.Game, session.EndedAt)
	}
	for _, gameID := range gameIDsToCleanup {
		service.mu.Lock()
		service.removeMonitoredGame(gameID)
		service.mu.Unlock()
	}
}

// updateMonitoredGameState は 1 ゲーム分の検知状態を更新する。
// service.mu を保持した状態で呼ばれる前提（ロックの取得/解放は呼び出し側）。
// 元コードの paused-running 分岐における continue は、本メソッドでは早期 return に対応する。
func (service *ProcessMonitorService) updateMonitoredGameState(
	game *MonitoringGame,
	processMap map[string][]normalizedProcess,
	now time.Time,
) {
	normalizedExeName := normalizeProcessToken(game.ExeName)
	matching := processMap[normalizedExeName]
	isRunning := false
	if len(matching) > 0 {
		isRunning = service.isGameProcessRunning(game.ExeName, game.ExePath, matching)
	}

	if isRunning {
		if game.IsPaused {
			if !game.SuppressResume {
				game.PendingResume = true
			}
			game.LastDetected = &now
			game.LastNotFound = nil
			return
		}
		game.LastDetected = &now
		game.LastNotFound = nil
		if game.PlayStartTime == nil && !game.IsPaused && !game.PendingEnd {
			game.PlayStartTime = &now
			game.AccumulatedTime = 0
			service.logger.Info("ゲーム開始を検知", "title", game.GameTitle, "exeName", game.ExeName)
		}
	} else {
		if game.PendingResume {
			game.PendingResume = false
		}
		if game.SuppressResume {
			game.SuppressResume = false
		}
		if game.LastNotFound == nil {
			game.LastNotFound = &now
		}
		if game.PlayStartTime != nil && !game.IsPaused && !game.PendingEnd {
			if now.Sub(*game.LastDetected) > service.sessionTimeout {
				duration := int64(now.Sub(*game.PlayStartTime).Seconds())
				if duration > 0 {
					game.AccumulatedTime += duration
				}
				game.PlayStartTime = nil
				game.PendingEnd = true
				game.LastDetected = nil
				service.logger.Info("ゲーム終了確認待ち", "title", game.GameTitle, "exeName", game.ExeName)
			}
		}
	}
}

// collectGameIDsToCleanup はクリーンアップ対象（一定時間未検出のゲーム）の ID を抽出して append する。
// service.mu を保持した状態で呼ばれる前提（ロックの取得/解放は呼び出し側）。
func (service *ProcessMonitorService) collectGameIDsToCleanup(now time.Time, gameIDsToCleanup []string) []string {
	for gameID, game := range service.monitoredGames {
		if game.PlayStartTime == nil && game.LastNotFound != nil && !game.IsPaused && !game.PendingEnd {
			if now.Sub(*game.LastNotFound) > service.gameCleanupTimeout {
				gameIDsToCleanup = append(gameIDsToCleanup, gameID)
			}
		}
	}
	return gameIDsToCleanup
}

func (service *ProcessMonitorService) saveSession(game MonitoringGame, endedAt time.Time) {
	sessionName := "自動記録 - " + game.ExeName
	ctx := context.Background()
	_, err := service.repository.CreatePlaySession(ctx, domain.PlaySession{
		GameID:      game.GameID,
		PlayedAt:    endedAt,
		Duration:    game.AccumulatedTime,
		SessionName: &sessionName,
	})
	if err != nil {
		service.logger.Error("プレイセッション保存に失敗", "error", err)
		return
	}

	current, err := service.repository.GetGameByID(ctx, game.GameID)
	if err != nil || current == nil {
		service.logger.Error("ゲーム取得に失敗", "error", err)
		return
	}
	current.TotalPlayTime += game.AccumulatedTime
	current.LastPlayed = &endedAt
	if current.SaveFolderPath != nil {
		saveFolderPath := strings.TrimSpace(*current.SaveFolderPath)
		if saveFolderPath != "" {
			if snap, hashErr := buildSaveTree(saveFolderPath); hashErr != nil {
				service.logger.Warn("ローカルセーブハッシュの計算に失敗", "error", hashErr)
			} else if snapJSON, merr := json.Marshal(snap); merr == nil {
				h := hashBytes(snapJSON)
				current.LocalSaveHash = &h
				current.LocalSaveHashUpdatedAt = &endedAt
			}
		}
	}

	if _, err := service.repository.UpdateGame(ctx, *current); err != nil {
		service.logger.Error("プレイ時間更新に失敗", "error", err)
		return
	}

	service.logger.Info("プレイセッションを保存", "exeName", game.ExeName, "duration", game.AccumulatedTime)
	if service.cloudSync != nil {
		go func(gameID string) {
			defer logging.Recover(service.logger, "process-monitor.afterPlayPush")
			if err := service.cloudSync.Push(context.Background(), gameID, nil); err != nil {
				// オフラインモードはユーザーが明示的に同期を抑止しているので warn 級にしない。
				if errors.Is(err, ErrOffline) {
					service.logger.Debug("オフラインモードのためクラウド同期をスキップ", "gameId", gameID)
					return
				}
				service.logger.Warn("クラウド同期に失敗", "gameId", gameID, "detail", err)
			}
		}(game.GameID)
	}
}

func (service *ProcessMonitorService) saveAllActiveSessions() {
	service.mu.Lock()
	type pendingSession struct {
		Game    MonitoringGame
		EndedAt time.Time
	}
	sessions := make([]pendingSession, 0, len(service.monitoredGames))
	now := time.Now()
	for _, game := range service.monitoredGames {
		if game.PlayStartTime != nil {
			duration := int64(now.Sub(*game.PlayStartTime).Seconds())
			if duration > 0 {
				game.AccumulatedTime += duration
			}
		}
		if game.AccumulatedTime > 0 {
			sessions = append(sessions, pendingSession{
				Game:    *game,
				EndedAt: now,
			})
		}
	}
	service.mu.Unlock()

	for _, session := range sessions {
		service.saveSession(session.Game, session.EndedAt)
	}
}

func (service *ProcessMonitorService) autoAddGamesFromDatabase(processes []ProcessInfo, normalized []normalizedProcess) {
	service.mu.Lock()
	autoTracking := service.autoTracking
	service.mu.Unlock()
	if !autoTracking {
		return
	}

	ctx := context.Background()
	games, err := service.repository.ListGames(ctx, "", domain.PlayStatus(""), "title", "asc")
	if err != nil || len(games) == 0 {
		return
	}

	processNames := make(map[string]struct{}, len(normalized))
	for _, proc := range normalized {
		if proc.normalized == "" {
			continue
		}
		processNames[proc.normalized] = struct{}{}
	}

	for _, game := range games {
		if game.ExePath == "" || game.ExePath == UnconfiguredExePath {
			continue
		}
		exeName := windowsPathBase(game.ExePath)
		normalizedExe := normalizeProcessToken(exeName)
		if _, ok := processNames[normalizedExe]; !ok {
			continue
		}
		if !service.isGameProcessRunning(exeName, game.ExePath, normalized) {
			continue
		}

		service.mu.Lock()
		if _, exists := service.monitoredGames[game.ID]; !exists {
			service.addMonitoredGame(game.ID, game.Title, game.ExePath)
		}
		service.mu.Unlock()
	}
}

func (service *ProcessMonitorService) isGameProcessRunning(
	gameExeName string,
	gameExePath string,
	processes []normalizedProcess,
) bool {
	for _, proc := range processes {
		if service.matchGameProcess(gameExeName, gameExePath, proc) {
			return true
		}
	}
	return false
}

func (service *ProcessMonitorService) matchGameProcess(
	gameExeName string,
	gameExePath string,
	proc normalizedProcess,
) bool {
	if proc.info.Name == "" || proc.info.Cmd == "" {
		return false
	}
	normalizedExeName := normalizeProcessToken(gameExeName)
	if proc.normalized != normalizedExeName {
		return false
	}

	normalizedExePath := normalizeProcessToken(normalizeWindowsPathSeparators(gameExePath))
	normalizedExeDir := normalizeProcessToken(windowsPathDir(gameExePath))
	procCmd := proc.normalizedCmd
	if procCmd == normalizedExePath {
		return true
	}
	if strings.Contains(procCmd, normalizedExePath) || strings.Contains(normalizedExePath, procCmd) {
		return true
	}
	if strings.Contains(procCmd, normalizedExeDir) {
		return true
	}
	return false
}

// FindProcessIDsByExe は実行ファイルパスに一致するプロセスIDを返す。
func (service *ProcessMonitorService) FindProcessIDsByExe(exePath string) ([]int, error) {
	trimmed := strings.TrimSpace(exePath)
	if trimmed == "" {
		return nil, errors.New("exePath is empty")
	}

	// 監視ループが更新した新しいスナップショットがあればそれを使い、無ければ再列挙する。
	processes := service.recentProcesses()
	if processes == nil {
		processes, _ = service.getProcesses()
	}
	// 稼働中の Windows でプロセス一覧が空になるのは列挙失敗時のみ。
	if len(processes) == 0 {
		return nil, errors.New("プロセス一覧を取得できませんでした")
	}

	exeName := windowsPathBase(trimmed)
	if !strings.HasSuffix(strings.ToLower(exeName), ".exe") {
		exeName += ".exe"
	}

	normalizedProcesses := normalizeProcessList(processes)

	ids := make([]int, 0, 2)
	for _, proc := range normalizedProcesses {
		if service.matchGameProcess(exeName, trimmed, proc) {
			ids = append(ids, proc.info.Pid)
		}
	}
	return ids, nil
}

func (service *ProcessMonitorService) getProcesses() ([]ProcessInfo, string) {
	if service.processProvider == nil {
		return []ProcessInfo{}, "unsupported"
	}
	processes, source := service.processProvider()
	service.cacheProcesses(processes)
	return processes, source
}

// cacheProcesses は非空のプロセス一覧をスナップショットとして保存する。
// getProcesses は service.mu を保持せずに呼ばれる前提のため、ここで短くロックを取得する。
func (service *ProcessMonitorService) cacheProcesses(processes []ProcessInfo) {
	if len(processes) == 0 {
		return
	}
	service.mu.Lock()
	service.lastProcesses = processes
	service.lastProcessesAt = time.Now()
	service.mu.Unlock()
}

// recentProcesses は監視ループの間隔内に更新された新しいスナップショットを返す。
// 古い（または未取得の）場合は nil を返し、呼び出し側で再列挙させる。
func (service *ProcessMonitorService) recentProcesses() []ProcessInfo {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.lastProcesses != nil && time.Since(service.lastProcessesAt) <= service.interval+time.Second {
		return service.lastProcesses
	}
	return nil
}

func normalizeProcessToken(value string) string {
	if value == "" {
		return ""
	}
	return norm.NFC.String(strings.ToLower(value))
}

func normalizeProcessPathToken(value string) string {
	return normalizeProcessToken(normalizeWindowsPathSeparators(value))
}
