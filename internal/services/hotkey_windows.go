//go:build windows

package services

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	hotkeyID    = 0x201
	wmHotkey    = 0x0312
	wmQuit      = 0x0012
	modAlt      = 0x0001
	modControl  = 0x0002
	modShift    = 0x0004
	modWin      = 0x0008
	modNoRepeat = 0x4000
	vkF1        = 0x70
)

var (
	kernel32dll           = windows.NewLazySystemDLL("kernel32.dll")
	procRegisterHotKey    = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey  = user32.NewProc("UnregisterHotKey")
	procGetMessage        = user32.NewProc("GetMessageW")
	procTranslateMessage  = user32.NewProc("TranslateMessage")
	procDispatchMessage   = user32.NewProc("DispatchMessageW")
	procPostThreadMessage = user32.NewProc("PostThreadMessageW")
	procGetThreadID       = kernel32dll.NewProc("GetCurrentThreadId")
)

type hotkeyServiceWindows struct {
	logger     *slog.Logger
	handler    HotkeyHandler
	modifiers  uint32
	key        uint32
	appPID     uint32
	started    atomic.Bool
	threadID   uint32
	stopCh     chan struct{}
	stoppedCh  chan struct{}
	lastNonApp atomic.Uintptr
	mu         sync.Mutex
}

func newHotkeyService(logger *slog.Logger, config HotkeyConfig, handler HotkeyHandler) HotkeyService {
	modifiers, key, err := parseHotkeyCombo(config.Combo)
	if err != nil {
		if logger != nil {
			logger.Warn("ホットキー設定の解析に失敗しました", "combo", config.Combo, "error", err)
		}
		return &hotkeyServiceWindows{logger: logger}
	}
	return &hotkeyServiceWindows{
		logger:    logger,
		handler:   handler,
		modifiers: modifiers,
		key:       key,
		appPID:    uint32(os.Getpid()),
	}
}

func (service *hotkeyServiceWindows) Start() error {
	if service == nil {
		return errors.New("hotkey service is nil")
	}
	if service.modifiers == 0 || service.key == 0 {
		return errors.New("hotkey is not configured")
	}
	if service.handler == nil {
		return errors.New("hotkey handler is nil")
	}
	if service.started.Swap(true) {
		return nil
	}

	service.stopCh = make(chan struct{})
	service.stoppedCh = make(chan struct{})

	go service.run()
	return nil
}

func (service *hotkeyServiceWindows) Stop() {
	if service == nil || !service.started.Load() {
		return
	}
	service.mu.Lock()
	threadID := service.threadID
	service.mu.Unlock()
	if threadID != 0 {
		procPostThreadMessage.Call(uintptr(threadID), wmQuit, 0, 0)
	}
	if service.stopCh != nil {
		close(service.stopCh)
	}
	if service.stoppedCh != nil {
		<-service.stoppedCh
	}
	service.started.Store(false)
}

func (service *hotkeyServiceWindows) run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	threadID, _, _ := procGetThreadID.Call()
	service.mu.Lock()
	service.threadID = uint32(threadID)
	service.mu.Unlock()

	ok, _, err := procRegisterHotKey.Call(0, hotkeyID, uintptr(service.modifiers), uintptr(service.key))
	if ok == 0 {
		if service.logger != nil {
			service.logger.Error("ホットキー登録に失敗しました", "error", err)
		}
		close(service.stoppedCh)
		return
	}
	if service.logger != nil {
		service.logger.Info("ホットキーを登録しました", "combo", formatHotkey(service.modifiers, service.key))
	}
	defer func() {
		procUnregisterHotKey.Call(0, hotkeyID)
		close(service.stoppedCh)
	}()

	go service.trackForeground()

	var msg windows.MSG
	for {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			return
		}
		if msg.Message == wmHotkey {
			target := service.resolveCaptureTarget()
			if target.HWND != 0 {
				go service.handler(target)
			} else if service.logger != nil {
				service.logger.Warn("撮影対象ウィンドウが見つかりませんでした")
			}
			continue
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func (service *hotkeyServiceWindows) trackForeground() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-service.stopCh:
			return
		case <-ticker.C:
			hwnd := foregroundWindowAny()
			if hwnd == 0 {
				continue
			}
			pid := windowProcessID(hwnd)
			if pid != 0 && pid != service.appPID {
				service.lastNonApp.Store(uintptr(hwnd))
			}
		}
	}
}

func (service *hotkeyServiceWindows) resolveCaptureTarget() CaptureTarget {
	hwnd := foregroundWindowAny()
	if hwnd == 0 {
		return CaptureTarget{}
	}
	pid := windowProcessID(hwnd)
	if pid != 0 && pid != service.appPID {
		service.lastNonApp.Store(uintptr(hwnd))
		return CaptureTarget{HWND: uintptr(hwnd)}
	}
	previous := service.lastNonApp.Load()
	if previous == 0 {
		return CaptureTarget{}
	}
	return CaptureTarget{HWND: previous, FromFallback: true}
}

func parseHotkeyCombo(combo string) (uint32, uint32, error) {
	trimmed := strings.TrimSpace(combo)
	if trimmed == "" {
		return 0, 0, errors.New("combo is empty")
	}
	parts := strings.Split(trimmed, "+")
	var modifiers uint32
	var key uint32
	for _, part := range parts {
		token := strings.ToUpper(strings.TrimSpace(part))
		if token == "" {
			continue
		}
		switch token {
		case "CTRL", "CONTROL":
			modifiers |= modControl
		case "ALT":
			modifiers |= modAlt
		case "SHIFT":
			modifiers |= modShift
		case "WIN", "WINDOWS":
			modifiers |= modWin
		default:
			if key != 0 {
				return 0, 0, fmt.Errorf("multiple keys: %s", combo)
			}
			parsed, ok := parseHotkeyKey(token)
			if !ok {
				return 0, 0, fmt.Errorf("unknown key: %s", token)
			}
			key = parsed
		}
	}
	if key == 0 {
		return 0, 0, errors.New("key is missing")
	}
	modifiers |= modNoRepeat
	return modifiers, key, nil
}

func parseHotkeyKey(token string) (uint32, bool) {
	if len(token) == 1 {
		ch := token[0]
		if ch >= 'A' && ch <= 'Z' {
			return uint32(ch), true
		}
		if ch >= '0' && ch <= '9' {
			return uint32(ch), true
		}
	}
	if strings.HasPrefix(token, "F") && len(token) > 1 {
		value, err := strconv.Atoi(token[1:])
		if err == nil && value >= 1 && value <= 12 {
			return uint32(vkF1 + value - 1), true
		}
	}
	return 0, false
}

func formatHotkey(modifiers uint32, key uint32) string {
	parts := make([]string, 0, 4)
	if modifiers&modControl != 0 {
		parts = append(parts, "Ctrl")
	}
	if modifiers&modAlt != 0 {
		parts = append(parts, "Alt")
	}
	if modifiers&modShift != 0 {
		parts = append(parts, "Shift")
	}
	if modifiers&modWin != 0 {
		parts = append(parts, "Win")
	}
	parts = append(parts, hotkeyKeyName(key))
	return strings.Join(parts, "+")
}

func hotkeyKeyName(key uint32) string {
	if key >= 'A' && key <= 'Z' {
		return string(rune(key))
	}
	if key >= '0' && key <= '9' {
		return string(rune(key))
	}
	if key >= vkF1 && key <= vkF1+11 {
		return "F" + strconv.Itoa(int(key-vkF1+1))
	}
	return fmt.Sprintf("0x%X", key)
}
