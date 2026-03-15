package date

import (
	"testing"
	"time"
)

func TestParseRelative(t *testing.T) {
	now := time.Now().UTC()
	todayStr := now.Format("2006-01-02")
	tomorrowStr := now.AddDate(0, 0, 1).Format("2006-01-02")
	threeDaysStr := now.AddDate(0, 0, 3).Format("2006-01-02")
	twoWeeksStr := now.AddDate(0, 0, 14).Format("2006-01-02")
	oneMonthStr := now.AddDate(0, 1, 0).Format("2006-01-02")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"today", "today", todayStr},
		{"tomorrow", "tomorrow", tomorrowStr},
		{"2d", "2d", now.AddDate(0, 0, 2).Format("2006-01-02")},
		{"3d", "3d", threeDaysStr},
		{"1w", "1w", now.AddDate(0, 0, 7).Format("2006-01-02")},
		{"2w", "2w", twoWeeksStr},
		{"1m", "1m", oneMonthStr},
		{"already formatted", "2023-12-25", "2023-12-25"},
		{"invalid format", "invalid", "invalid"},
		{"capitalized", "TODAY", todayStr},
		{"1y", "1y", now.AddDate(1, 0, 0).Format("2006-01-02")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := ParseRelative(tc.input)
			if actual != tc.expected {
				t.Errorf("ParseRelative(%q) = %q, expected %q", tc.input, actual, tc.expected)
			}
		})
	}
}
