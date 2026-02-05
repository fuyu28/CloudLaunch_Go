package services

import (
	"testing"
)

func TestTrimQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "path with spaces and quotes",
			input:    `"C:\Program Files\My Game\game.exe"`,
			expected: `C:\Program Files\My Game\game.exe`,
		},
		{
			name:     "path with spaces without quotes",
			input:    `C:\Program Files\My Game\game.exe`,
			expected: `C:\Program Files\My Game\game.exe`,
		},
		{
			name:     "path with Japanese characters and quotes",
			input:    `"C:\Games\テストゲーム\game.exe"`,
			expected: `C:\Games\テストゲーム\game.exe`,
		},
		{
			name:     "path with Japanese characters without quotes",
			input:    `C:\Games\テストゲーム\game.exe`,
			expected: `C:\Games\テストゲーム\game.exe`,
		},
		{
			name:     "empty quotes",
			input:    `""`,
			expected: ``,
		},
		{
			name:     "single quote",
			input:    `"`,
			expected: `"`,
		},
		{
			name:     "empty string",
			input:    ``,
			expected: ``,
		},
		{
			name:     "only opening quote",
			input:    `"test`,
			expected: `"test`,
		},
		{
			name:     "only closing quote",
			input:    `test"`,
			expected: `test"`,
		},
		{
			name:     "quotes in the middle",
			input:    `C:\Program "Files"\game.exe`,
			expected: `C:\Program "Files"\game.exe`,
		},
		{
			name:     "single character with quotes",
			input:    `"a"`,
			expected: `a`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimQuotes(tt.input)
			if result != tt.expected {
				t.Errorf("trimQuotes(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
