package sqlite

import (
	"strings"
	"time"
)

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// parsing time "2025-08-05T10:15:43Z" as "2006-01-02 15:04:05": cannot parse "T10:15:43Z" as " "
func tryParseTime(s string) (time.Time, error) {
	if strings.Contains(s, "T") {
		return tryParseStdTime(s)
	}
	return time.Parse("2006-01-02 15:04:05", s)
}

func tryParseStdTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05Z", s)
}
