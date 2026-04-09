package domain

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

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
	t, err := time.ParseInLocation(time.DateOnly, m[1], loc)
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

func ISOWeekStart(loc *time.Location, year int, week int) time.Time {
	if loc == nil {
		loc = time.Local
	}
	if week < 1 || week > 53 || year < 1 {
		return time.Time{}
	}
	// ISO week 1 is the week with Jan 4th.
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, loc)
	wd := int(jan4.Weekday())
	if wd == 0 {
		wd = 7 // Sunday -> 7
	}
	mondayWeek1 := jan4.AddDate(0, 0, -(wd - 1))
	return mondayWeek1.AddDate(0, 0, (week-1)*7)
}
