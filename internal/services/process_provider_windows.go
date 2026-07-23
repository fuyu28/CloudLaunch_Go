//go:build windows

// Windows 向けのプロセス列挙（PowerShell / WMIC）と CSV・文字コード解釈を提供する。
package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"io"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// defaultProcessProvider は PowerShell を優先し、失敗時のみ WMIC へフォールバックする。
func defaultProcessProvider(logger *slog.Logger) func() ([]ProcessInfo, string) {
	return func() ([]ProcessInfo, string) {
		return listWindowsProcesses(logger, getProcessesPowerShell, getProcessesWmic)
	}
}

// listWindowsProcesses は native → fallback の解決順序を固定する。
// テストでは各取得関数を差し替えてフォールバック分岐だけを検証する。
func listWindowsProcesses(
	logger *slog.Logger,
	native func() ([]ProcessInfo, error),
	fallback func() ([]ProcessInfo, error),
) ([]ProcessInfo, string) {
	processes, err := native()
	if err == nil {
		return processes, "native"
	}

	if logger != nil {
		logger.Warn("ネイティブコマンドが失敗しました。フォールバックを使用します", "error", err)
	}
	processes, err = fallback()
	if err != nil {
		if logger != nil {
			logger.Error("フォールバックも失敗しました", "error", err)
		}
		return []ProcessInfo{}, "fallback"
	}
	return processes, "fallback"
}

func getProcessesPowerShell() ([]ProcessInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := execCommandHidden(
		ctx,
		"powershell",
		"-Command",
		`$OutputEncoding=[System.Text.Encoding]::UTF8; Get-Process | Select-Object ProcessName, Id, Path | ConvertTo-Csv -NoTypeInformation`,
	)
	output, err := command.Output()
	if err != nil {
		return nil, err
	}
	return processesFromPowerShellCSV(output)
}

func getProcessesWmic() ([]ProcessInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := execCommandHidden(
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
	return processesFromWmicCSV(output)
}

func processesFromPowerShellCSV(output []byte) ([]ProcessInfo, error) {
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
		if ext := strings.ToLower(filepath.Ext(fullPath)); ext != ".exe" {
			continue
		}
		processes = append(processes, ProcessInfo{Name: name, Pid: pid, Cmd: fullPath})
	}
	return processes, nil
}

func processesFromWmicCSV(output []byte) ([]ProcessInfo, error) {
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
		if ext := strings.ToLower(filepath.Ext(fullPath)); ext != ".exe" {
			continue
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

	// まずUTF-8の生データを優先して解釈し、失敗時のみShift-JISへフォールバックする。
	if utf8.Valid(output) {
		if records, err := parse(output); err == nil {
			return records, nil
		}
	}

	if decoded, err := decodeProcessOutput(output); err == nil {
		if records, err := parse(decoded); err == nil {
			return records, nil
		}
	}

	return parse(output)
}
