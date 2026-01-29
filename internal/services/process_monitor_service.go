// @fileoverview ゲームの実行プロセス監視と自動プレイ時間計測を提供する。
package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"runtime"
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
}

// ProcessInfo はプロセス情報を保持する。
type ProcessInfo struct {
	Name string
	Pid  int
	Cmd  string
}

// ProcessMonitorService はゲームプロセス監視を提供する。
type ProcessMonitorService struct {
	repository         *db.Repository
	logger             *slog.Logger
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
func NewProcessMonitorService(repository *db.Repository, logger *slog.Logger) *ProcessMonitorService {
	return &ProcessMonitorService{
		repository:         repository,
		logger:             logger,
		monitoredGames:     make(map[string]*MonitoringGame),
		autoTracking:       true,
		interval:           2 * time.Second,
		sessionTimeout:     4 * time.Second,
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
		if game.PlayStartTime != nil {
			playTime += int64(now.Sub(*game.PlayStartTime).Seconds())
		}
		status = append(status, models.MonitoringGameStatus{
			GameID:    game.GameID,
			GameTitle: game.GameTitle,
			ExeName:   game.ExeName,
			IsPlaying: game.PlayStartTime != nil,
			PlayTime:  playTime,
		})
	}
	return status
}

// GetProcessSnapshot は現在のプロセス一覧と正規化後の値を取得する。
func (service *ProcessMonitorService) GetProcessSnapshot() models.ProcessSnapshot {
	processes, err := service.getProcessesNative()
	source := "native"
	if err != nil {
		service.logger.Warn("ネイティブコマンドが失敗しました。フォールバックを使用します", "error", err)
		processes, err = service.getProcessesFallback()
		source = "fallback"
		if err != nil {
			service.logger.Error("フォールバックも失敗しました", "error", err)
			processes = []ProcessInfo{}
		}
	}

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
			service.saveSession(*game, duration, now)
		}
	}
	delete(service.monitoredGames, gameID)
	service.logger.Info("ゲーム監視を削除", "exeName", game.ExeName, "gameId", gameID)
}

func (service *ProcessMonitorService) checkProcesses() {
	processes, err := service.getProcessesNative()
	if err != nil {
		service.logger.Warn("ネイティブコマンドが失敗しました。フォールバックを使用します", "error", err)
		processes, err = service.getProcessesFallback()
		if err != nil {
			service.logger.Error("フォールバックも失敗しました", "error", err)
			processes = []ProcessInfo{}
		}
	}

	service.autoAddGamesFromDatabase(processes)

	processMap := make(map[string][]ProcessInfo)
	for _, proc := range processes {
		if proc.Name == "" {
			continue
		}
		name := normalizeProcessToken(proc.Name)
		processMap[name] = append(processMap[name], proc)
	}

	now := time.Now()
	type pendingSession struct {
		Game     MonitoringGame
		Duration int64
		EndedAt  time.Time
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
			game.LastDetected = &now
			game.LastNotFound = nil
			if game.PlayStartTime == nil {
				game.PlayStartTime = &now
				game.AccumulatedTime = 0
				service.logger.Info("ゲーム開始を検知", "title", game.GameTitle, "exeName", game.ExeName)
			}
		} else {
			if game.LastNotFound == nil {
				game.LastNotFound = &now
			}
			if game.PlayStartTime != nil && game.LastDetected != nil {
				if now.Sub(*game.LastDetected) > service.sessionTimeout {
					duration := int64(now.Sub(*game.PlayStartTime).Seconds())
					if duration > 0 {
						game.AccumulatedTime += duration
						sessionsToSave = append(sessionsToSave, pendingSession{
							Game:     *game,
							Duration: duration,
							EndedAt:  now,
						})
					}
					game.PlayStartTime = nil
					game.LastDetected = nil
					service.logger.Info("ゲーム終了を検知", "title", game.GameTitle, "exeName", game.ExeName)
				}
			}
		}
	}
	for gameID, game := range service.monitoredGames {
		if game.PlayStartTime == nil && game.LastNotFound != nil {
			if now.Sub(*game.LastNotFound) > service.gameCleanupTimeout {
				gameIDsToCleanup = append(gameIDsToCleanup, gameID)
			}
		}
	}
	service.mu.Unlock()

	for _, session := range sessionsToSave {
		service.saveSession(session.Game, session.Duration, session.EndedAt)
	}
	for _, gameID := range gameIDsToCleanup {
		service.mu.Lock()
		service.removeMonitoredGame(gameID)
		service.mu.Unlock()
	}
}

func (service *ProcessMonitorService) saveSession(game MonitoringGame, duration int64, endedAt time.Time) {
	if duration <= 0 {
		return
	}
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
}

func (service *ProcessMonitorService) saveAllActiveSessions() {
	service.mu.Lock()
	type pendingSession struct {
		Game     MonitoringGame
		Duration int64
		EndedAt  time.Time
	}
	sessions := make([]pendingSession, 0, len(service.monitoredGames))
	now := time.Now()
	for _, game := range service.monitoredGames {
		if game.PlayStartTime != nil {
			duration := int64(now.Sub(*game.PlayStartTime).Seconds())
			if duration > 0 {
				game.AccumulatedTime += duration
				sessions = append(sessions, pendingSession{
					Game:     *game,
					Duration: duration,
					EndedAt:  now,
				})
			}
		}
	}
	service.mu.Unlock()

	for _, session := range sessions {
		service.saveSession(session.Game, session.Duration, session.EndedAt)
	}
}

func (service *ProcessMonitorService) autoAddGamesFromDatabase(processes []ProcessInfo) {
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

	processNames := make(map[string]struct{}, len(processes))
	for _, proc := range processes {
		if proc.Name == "" {
			continue
		}
		processNames[normalizeProcessToken(proc.Name)] = struct{}{}
	}

	for _, game := range games {
		if game.ExePath == "" {
			continue
		}
		exeName := filepath.Base(game.ExePath)
		normalizedExe := normalizeProcessToken(exeName)
		if _, ok := processNames[normalizedExe]; !ok {
			continue
		}
		if !service.isGameProcessRunning(exeName, game.ExePath, processes) {
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
	processes []ProcessInfo,
) bool {
	normalizedExeName := normalizeProcessToken(gameExeName)
	normalizedExePath := normalizeProcessToken(gameExePath)
	normalizedExeDir := normalizeProcessToken(filepath.Dir(gameExePath))

	for _, proc := range processes {
		if proc.Name == "" || proc.Cmd == "" {
			continue
		}
		procName := normalizeProcessToken(proc.Name)
		if procName != normalizedExeName {
			continue
		}

		procCmd := normalizeProcessToken(proc.Cmd)
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
	if runtime.GOOS != "windows" {
		return nil, errors.New("process monitoring is supported on Windows only")
	}
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
	if runtime.GOOS != "windows" {
		return nil, errors.New("process monitoring is supported on Windows only")
	}
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

	if decoded, err := decodeUTF16LE(output); err == nil {
		return parse(decoded)
	}

	return parse(output)
}

func normalizeProcessToken(value string) string {
	if value == "" {
		return ""
	}
	return norm.NFC.String(strings.ToLower(value))
}
