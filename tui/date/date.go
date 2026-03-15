package date

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	daysRegex   = regexp.MustCompile(`^(\d+)d$`)
	weeksRegex  = regexp.MustCompile(`^(\d+)w$`)
	monthsRegex = regexp.MustCompile(`^(\d+)m$`)
	yearsRegex  = regexp.MustCompile(`^(\d+)y$`)
)

// ParseRelative converts relative dates (like "today", "tomorrow", "2d", "1w") to YYYY-MM-DD
// Uses UTC time for consistent syncing, same as Lua os.date("!%Y-%m-%d")
func ParseRelative(val string) string {
	val = strings.ToLower(val)
	now := time.Now().UTC()

	switch val {
	case "today":
		return now.Format("2006-01-02")
	case "tomorrow":
		return now.AddDate(0, 0, 1).Format("2006-01-02")
	}

	// X days
	if match := daysRegex.FindStringSubmatch(val); match != nil {
		days, _ := strconv.Atoi(match[1])
		return now.AddDate(0, 0, days).Format("2006-01-02")
	}

	// X weeks
	if match := weeksRegex.FindStringSubmatch(val); match != nil {
		weeks, _ := strconv.Atoi(match[1])
		return now.AddDate(0, 0, weeks*7).Format("2006-01-02")
	}

	// X months
	if match := monthsRegex.FindStringSubmatch(val); match != nil {
		months, _ := strconv.Atoi(match[1])
		return now.AddDate(0, months, 0).Format("2006-01-02")
	}

	// X years
	if match := yearsRegex.FindStringSubmatch(val); match != nil {
		years, _ := strconv.Atoi(match[1])
		return now.AddDate(years, 0, 0).Format("2006-01-02")
	}

	// If it doesn't match a relative format, return it as-is (might already be YYYY-MM-DD or invalid)
	return val
}
