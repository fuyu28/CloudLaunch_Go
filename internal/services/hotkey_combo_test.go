package services

import (
	"strings"
	"testing"
)

func TestParseHotkeyCombo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		combo      string
		wantKey    uint32
		wantMod    uint32 // without MOD_NOREPEAT
		wantErrSub string
	}{
		{name: "default", combo: "Ctrl+Alt+S", wantKey: 'S', wantMod: modControl | modAlt},
		{name: "aliases", combo: "control+windows+f8", wantKey: vkF1 + 7, wantMod: modControl | modWin},
		{name: "bare F8", combo: "F8", wantKey: vkF1 + 7, wantMod: 0},
		{name: "shift insert", combo: "Shift+Insert", wantKey: vkInsert, wantMod: modShift},
		{name: "page aliases", combo: "Alt+PgDn", wantKey: vkNext, wantMod: modAlt},
		{name: "space", combo: "Ctrl+Space", wantKey: vkSpace, wantMod: modControl},
		{name: "empty", combo: "  ", wantErrSub: "empty"},
		{name: "unknown", combo: "Ctrl+Foo", wantErrSub: "unknown key"},
		{name: "modifier only", combo: "Ctrl+Alt", wantErrSub: "key is missing"},
		{name: "multiple keys", combo: "Ctrl+A+B", wantErrSub: "multiple keys"},
		{name: "f12 rejected", combo: "F12", wantErrSub: "unknown key"},
		{name: "ctrl f12 rejected", combo: "Ctrl+F12", wantErrSub: "unknown key"},
		{name: "printscreen rejected", combo: "PrintScreen", wantErrSub: "unknown key"},
		{name: "f13 rejected", combo: "F13", wantErrSub: "unknown key"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mods, key, err := parseHotkeyCombo(tt.combo)
			if tt.wantErrSub != "" {
				if err == nil {
					t.Fatalf("expected error containing %q", tt.wantErrSub)
				}
				if !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if key != tt.wantKey {
				t.Fatalf("key: got 0x%X want 0x%X", key, tt.wantKey)
			}
			gotUserMods := mods &^ modNoRepeat
			if gotUserMods != tt.wantMod {
				t.Fatalf("modifiers: got 0x%X want 0x%X", gotUserMods, tt.wantMod)
			}
			if mods&modNoRepeat == 0 {
				t.Fatal("expected MOD_NOREPEAT to be set")
			}
		})
	}
}

func TestHotkeyKeyName(t *testing.T) {
	t.Parallel()
	cases := map[uint32]string{
		'S':      "S",
		vkF1 + 7: "F8",
		vkInsert: "Insert",
		vkSpace:  "Space",
		vkPrior:  "PageUp",
		vkScroll: "ScrollLock",
	}
	for key, want := range cases {
		if got := hotkeyKeyName(key); got != want {
			t.Fatalf("hotkeyKeyName(0x%X)=%q want %q", key, got, want)
		}
	}
}

func TestValidateHotkeyCombo(t *testing.T) {
	t.Parallel()
	if err := ValidateHotkeyCombo("F8"); err != nil {
		t.Fatalf("ValidateHotkeyCombo(F8): %v", err)
	}
	if err := ValidateHotkeyCombo("PrintScreen"); err == nil {
		t.Fatal("expected error for PrintScreen")
	}
	if err := ValidateHotkeyCombo("F12"); err == nil {
		t.Fatal("expected error for F12")
	}
	if err := ValidateHotkeyCombo("Nope"); err == nil {
		t.Fatal("expected error for Nope")
	}
}
