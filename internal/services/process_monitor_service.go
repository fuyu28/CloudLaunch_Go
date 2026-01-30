// @fileoverview ゲームの実行プロセス監視と自動プレイ時間計測を提供する。
package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"CloudLaunch_Go/internal/db"
	"CloudLaunch_Go/internal/models"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
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

// ProcessMonitorService はゲームプロセス監視を提供する。
type ProcessMonitorService struct {
	repository         *db.Repository
	logger             *slog.Logger
	cloudSync          *CloudSyncService
	monitoredGames     map[string]*MonitoringGame
	autoTracking       bool
	monitoringInterval *time.Ticker
	monitoringStop     chan struct{}
	mu                 sync.Mutex
	interval           time.Duration
	sessionTimeout     time.Duration
	gameCleanupTimeout time.Duration
}

// NewProcessMonitorService は ProcessMonitorService を生成する。
func NewProcessMonitorService(repository *db.Repository, logger *slog.Logger, cloudSync *CloudSyncService) *ProcessMonitorService {
	return &ProcessMonitorService{
		repository:         repository,
		logger:             logger,
		cloudSync:          cloudSync,
		monitoredGames:     make(map[string]*MonitoringGame),
		autoTracking:       true,
		interval:           2 * time.Second,
		sessionTimeout:     0,
		gameCleanupTimeout: 20 * time.Second,
	}
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
		// 起動時に即時チェック
		service.checkProcesses()
		for {
			select {
			case <-service.monitoringInterval.C:
				service.checkProcesses()
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
func (service *ProcessMonitorService) GetMonitoringStatus() []models.MonitoringGameStatus {
	service.mu.Lock()
	defer service.mu.Unlock()

	status := make([]models.MonitoringGameStatus, 0, len(service.monitoredGames))
	now := time.Now()
	for _, game := range service.monitoredGames {
		playTime := game.AccumulatedTime
		if game.PlayStartTime != nil && !game.IsPaused && !game.PendingEnd {
			playTime += int64(now.Sub(*game.PlayStartTime).Seconds())
		}
		status = append(status, models.MonitoringGameStatus{
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

// GetProcessSnapshot は現在のプロセス一覧と正規化後の値を取得する。
func (service *ProcessMonitorService) GetProcessSnapshot() models.ProcessSnapshot {
	processes, source := service.getProcesses()

	items := make([]models.ProcessSnapshotItem, 0, len(processes))
	for _, proc := range processes {
		items = append(items, models.ProcessSnapshotItem{
			Name:           proc.Name,
			Pid:            proc.Pid,
			Cmd:            proc.Cmd,
			NormalizedName: normalizeProcessToken(proc.Name),
			NormalizedCmd:  normalizeProcessToken(proc.Cmd),
		})
	}

	return models.ProcessSnapshot{
		Source: source,
		Items:  items,
	}
}

func (service *ProcessMonitorService) addMonitoredGame(gameID string, title string, exePath string) {
	exeName := filepath.Base(exePath)
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
	normalizedProcesses := make([]normalizedProcess, 0, len(processes))
	for _, proc := range processes {
		if proc.Name == "" {
			continue
		}
		normalizedProcesses = append(normalizedProcesses, normalizedProcess{
			info:          proc,
			normalized:    normalizeProcessToken(proc.Name),
			normalizedCmd: normalizeProcessToken(proc.Cmd),
		})
	}

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
	service.mu.Unlock()

	if accumulated > 0 {
		game.AccumulatedTime = accumulated
		service.saveSession(*game, now)
		service.mu.Lock()
		if current, ok := service.monitoredGames[gameID]; ok {
			current.AccumulatedTime = 0
		}
		service.mu.Unlock()
	}
	return true
}

func (service *ProcessMonitorService) checkProcesses() {
	processes, _ := service.getProcesses()

	normalizedProcesses := make([]normalizedProcess, 0, len(processes))
	for _, proc := range processes {
		if proc.Name == "" {
			continue
		}
		normalizedProcesses = append(normalizedProcesses, normalizedProcess{
			info:          proc,
			normalized:    normalizeProcessToken(proc.Name),
			normalizedCmd: normalizeProcessToken(proc.Cmd),
		})
	}

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
				continue
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
	for gameID, game := range service.monitoredGames {
		if game.PlayStartTime == nil && game.LastNotFound != nil && !game.IsPaused && !game.PendingEnd {
			if now.Sub(*game.LastNotFound) > service.gameCleanupTimeout {
				gameIDsToCleanup = append(gameIDsToCleanup, gameID)
			}
		}
	}
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

func (service *ProcessMonitorService) saveSession(game MonitoringGame, endedAt time.Time) {
	sessionName := "自動記録 - " + game.ExeName
	ctx := context.Background()
	_, err := service.repository.CreatePlaySession(ctx, models.PlaySession{
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

	if _, err := service.repository.UpdateGame(ctx, *current); err != nil {
		service.logger.Error("プレイ時間更新に失敗", "error", err)
		return
	}

	service.logger.Info("プレイセッションを保存", "exeName", game.ExeName, "duration", game.AccumulatedTime)
	if service.cloudSync != nil {
		go func(gameID string) {
			result := service.cloudSync.SyncGame(context.Background(), "default", gameID)
			if !result.Success {
				service.logger.Warn("クラウド同期に失敗", "gameId", gameID, "detail", result.Error)
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
	games, err := service.repository.ListGames(ctx, "", models.PlayStatus(""), "title", "asc")
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
		exeName := filepath.Base(game.ExePath)
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
	normalizedExeName := normalizeProcessToken(gameExeName)
	normalizedExePath := normalizeProcessToken(gameExePath)
	normalizedExeDir := normalizeProcessToken(filepath.Dir(gameExePath))

	for _, proc := range processes {
		if proc.info.Name == "" || proc.info.Cmd == "" {
			continue
		}
		if proc.normalized != normalizedExeName {
			continue
		}

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
	}
	return false
}

func (service *ProcessMonitorService) getProcessesNative() ([]ProcessInfo, error) {
	return service.getProcessesPowerShell()
}

func (service *ProcessMonitorService) getProcessesPowerShell() ([]ProcessInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(
		ctx,
		"powershell",
		"-Command",
		`$OutputEncoding=[System.Text.Encoding]::UTF8; Get-Process | Select-Object ProcessName, Id, Path | ConvertTo-Csv -NoTypeInformation`,
	)
	output, err := command.Output()
	if err != nil {
		return nil, err
	}

	records, err := parseCSVBytes(output)
	if err != nil {
		return nil, err
	}

	processes := make([]ProcessInfo, 0, len(records))
	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) < 3 {
			continue
		}
		name := strings.TrimSpace(record[0])
		pidStr := strings.TrimSpace(record[1])
		fullPath := strings.TrimSpace(record[2])
		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid <= 0 || name == "" {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(name), ".exe") {
			name += ".exe"
		}
		if fullPath == "" {
			fullPath = name
		}
		processes = append(processes, ProcessInfo{Name: name, Pid: pid, Cmd: fullPath})
	}
	return processes, nil
}

func (service *ProcessMonitorService) getProcessesFallback() ([]ProcessInfo, error) {
	return service.getProcessesWmic()
}

func (service *ProcessMonitorService) getProcessesWmic() ([]ProcessInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(
		ctx,
		"wmic",
		"process",
		"get",
		"Name,ProcessId,ExecutablePath",
		"/FORMAT:CSV",
	)
	output, err := command.Output()
	if err != nil {
		return nil, err
	}

	records, err := parseCSVBytes(output)
	if err != nil {
		return nil, err
	}

	processes := make([]ProcessInfo, 0, len(records))
	for _, record := range records {
		if len(record) < 4 {
			continue
		}
		name := strings.TrimSpace(record[1])
		pidStr := strings.TrimSpace(record[2])
		fullPath := strings.TrimSpace(record[3])
		if name == "" || pidStr == "" {
			continue
		}
		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid <= 0 {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(name), ".exe") {
			name += ".exe"
		}
		if fullPath == "" {
			fullPath = name
		}
		processes = append(processes, ProcessInfo{Name: name, Pid: pid, Cmd: fullPath})
	}
	return processes, nil
}

func decodeProcessOutput(output []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(output), japanese.ShiftJIS.NewDecoder())
	return io.ReadAll(reader)
}

func decodeUTF16LE(output []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(output), unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder())
	return io.ReadAll(reader)
}

func parseCSVBytes(output []byte) ([][]string, error) {
	parse := func(data []byte) ([][]string, error) {
		reader := csv.NewReader(bytes.NewReader(data))
		reader.LazyQuotes = true
		reader.TrimLeadingSpace = true
		return reader.ReadAll()
	}

	if bytes.Contains(output, []byte{0x00}) {
		if decoded, err := decodeUTF16LE(output); err == nil {
			if records, err := parse(decoded); err == nil {
				return records, nil
			}
		}
	}

	if records, err := parse(output); err == nil {
		return records, nil
	}

	if decoded, err := decodeProcessOutput(output); err == nil {
		if records, err := parse(decoded); err == nil {
			return records, nil
		}
	}

	return parse(output)
}

func (service *ProcessMonitorService) getProcesses() ([]ProcessInfo, string) {
	processes, err := service.getProcessesNative()
	if err == nil {
		return processes, "native"
	}

	service.logger.Warn("ネイティブコマンドが失敗しました。フォールバックを使用します", "error", err)
	processes, err = service.getProcessesFallback()
	if err != nil {
		service.logger.Error("フォールバックも失敗しました", "error", err)
		return []ProcessInfo{}, "fallback"
	}
	return processes, "fallback"
}

func normalizeProcessToken(value string) string {
	if value == "" {
		return ""
	}
	return norm.NFC.String(strings.ToLower(value))
}
