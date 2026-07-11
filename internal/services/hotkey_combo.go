package services

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	vkF1     = 0x70
	vkInsert = 0x2D
	vkDelete = 0x2E
	vkHome   = 0x24
	vkEnd    = 0x23
	vkPrior  = 0x21 // PageUp
	vkNext   = 0x22 // PageDown
	vkScroll = 0x91
	vkPause  = 0x13
	vkSpace  = 0x20
)

// namedHotkeyKeys は表示名 → VK。エイリアスは parse 側で正規化する。
// PrintScreen / F12 は OS・WebView 都合で安定しないため非対応。
var namedHotkeyKeys = map[string]uint32{
	"INSERT":     vkInsert,
	"DELETE":     vkDelete,
	"HOME":       vkHome,
	"END":        vkEnd,
	"PAGEUP":     vkPrior,
	"PAGEDOWN":   vkNext,
	"SCROLLLOCK": vkScroll,
	"PAUSE":      vkPause,
	"SPACE":      vkSpace,
}

var namedHotkeyAliases = map[string]string{
	"INS":    "INSERT",
	"DEL":    "DELETE",
	"PGUP":   "PAGEUP",
	"PGDN":   "PAGEDOWN",
	"SCROLL": "SCROLLLOCK",
	"BREAK":  "PAUSE",
}

// ValidateHotkeyCombo はホットキー文字列がサポート形式かを検証する。
func ValidateHotkeyCombo(combo string) error {
	_, _, err := parseHotkeyCombo(combo)
	return err
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
		// F12 は DevTools / ホスト側に取られやすいため非対応（F1–F11 のみ）
		if err == nil && value >= 1 && value <= 11 {
			return uint32(vkF1 + value - 1), true
		}
	}
	if canonical, ok := namedHotkeyAliases[token]; ok {
		token = canonical
	}
	if vk, ok := namedHotkeyKeys[token]; ok {
		return vk, true
	}
	return 0, false
}

func hotkeyKeyName(key uint32) string {
	if key >= 'A' && key <= 'Z' {
		return string(rune(key))
	}
	if key >= '0' && key <= '9' {
		return string(rune(key))
	}
	if key >= vkF1 && key <= vkF1+10 {
		return "F" + strconv.Itoa(int(key-vkF1+1))
	}
	for name, vk := range namedHotkeyKeys {
		if vk == key {
			switch name {
			case "SCROLLLOCK":
				return "ScrollLock"
			case "PAGEUP":
				return "PageUp"
			case "PAGEDOWN":
				return "PageDown"
			default:
				return strings.ToUpper(name[:1]) + strings.ToLower(name[1:])
			}
		}
	}
	return fmt.Sprintf("0x%X", key)
}
