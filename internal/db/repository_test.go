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
		current       models.PlayStatus
		lastPlayed    *time.Time
		clearedAt     *time.Time
		totalPlayTime int64
		expected      models.PlayStatus
	}{
		{
			name:          "clearedAt wins over stale status",
			current:       models.PlayStatusUnplayed,
			clearedAt:     &clearedAt,
			totalPlayTime: 0,
			expected:      models.PlayStatusPlayed,
		},
		{
			name:          "played status is preserved",
			current:       models.PlayStatusPlayed,
			totalPlayTime: 0,
			expected:      models.PlayStatusPlayed,
		},
		{
			name:          "history promotes to playing by lastPlayed",
			current:       models.PlayStatusUnplayed,
			lastPlayed:    &lastPlayed,
			totalPlayTime: 0,
			expected:      models.PlayStatusPlaying,
		},
		{
			name:          "history promotes to playing by total play time",
			current:       models.PlayStatusUnplayed,
			totalPlayTime: 30,
			expected:      models.PlayStatusPlaying,
		},
		{
			name:          "no history stays unplayed",
			current:       models.PlayStatusPlaying,
			totalPlayTime: 0,
			expected:      models.PlayStatusUnplayed,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			actual := normalizeProgressPlayStatus(
				testCase.current,
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
