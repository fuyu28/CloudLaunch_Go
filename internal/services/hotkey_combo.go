package services

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	vkF1 = 0x70
)

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
		if err == nil && value >= 1 && value <= 12 {
			return uint32(vkF1 + value - 1), true
		}
	}
	return 0, false
}
