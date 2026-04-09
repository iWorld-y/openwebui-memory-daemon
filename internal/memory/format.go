package memory

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Kind int

const (
	KindNone Kind = iota
	KindDaily
	KindWeekly
	KindMonthly
)

func KindFromContent(content string) Kind {
	s := strings.TrimSpace(content)
	switch {
	case strings.HasPrefix(s, "📋 "):
		return KindDaily
	case strings.HasPrefix(s, "📦 "):
		return KindWeekly
	case strings.HasPrefix(s, "📅 "):
		return KindMonthly
	default:
		return KindNone
	}
}

var (
	reDaily   = regexp.MustCompile(`^📋\s+(\d{4}-\d{2}-\d{2})\b`)
	reWeekly  = regexp.MustCompile(`^📦\s+(\d{4})-W(\d{1,2})\b`)
	reMonthly = regexp.MustCompile(`^📅\s+(\d{4})-(\d{2})\b`)
)

func ParseDailyDate(content string, loc *time.Location) (time.Time, bool) {
	if loc == nil {
		loc = time.Local
	}
	m := reDaily.FindStringSubmatch(strings.TrimSpace(content))
	if len(m) != 2 {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", m[1], loc)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func ParseWeeklyKey(content string) (year int, week int, ok bool) {
	m := reWeekly.FindStringSubmatch(strings.TrimSpace(content))
	if len(m) != 3 {
		return 0, 0, false
	}
	y, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, 0, false
	}
	w, err := strconv.Atoi(m[2])
	if err != nil || w < 1 || w > 53 {
		return 0, 0, false
	}
	return y, w, true
}

func ParseMonthlyKey(content string) (year int, month int, ok bool) {
	m := reMonthly.FindStringSubmatch(strings.TrimSpace(content))
	if len(m) != 3 {
		return 0, 0, false
	}
	y, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, 0, false
	}
	mo, err := strconv.Atoi(m[2])
	if err != nil || mo < 1 || mo > 12 {
		return 0, 0, false
	}
	return y, mo, true
}

