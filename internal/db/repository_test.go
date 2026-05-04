package db

import (
	"testing"
	"time"

	"CloudLaunch_Go/internal/models"
)

func TestNormalizeProgressPlayStatus(t *testing.T) {
	t.Parallel()

	lastPlayed := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	clearedAt := time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name          string
		lastPlayed    *time.Time
		clearedAt     *time.Time
		totalPlayTime int64
		expected      models.PlayStatus
	}{
		{
			name:          "clearedAt wins over progress",
			clearedAt:     &clearedAt,
			totalPlayTime: 0,
			expected:      models.PlayStatusCleared,
		},
		{
			name:          "history promotes to playing by lastPlayed",
			lastPlayed:    &lastPlayed,
			totalPlayTime: 0,
			expected:      models.PlayStatusPlaying,
		},
		{
			name:          "history promotes to playing by total play time",
			totalPlayTime: 30,
			expected:      models.PlayStatusPlaying,
		},
		{
			name:          "no history stays unplayed",
			totalPlayTime: 0,
			expected:      models.PlayStatusUnplayed,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			actual := normalizeProgressPlayStatus(
				testCase.lastPlayed,
				testCase.clearedAt,
				testCase.totalPlayTime,
			)

			if actual != testCase.expected {
				t.Fatalf("expected %q, got %q", testCase.expected, actual)
			}
		})
	}
}
