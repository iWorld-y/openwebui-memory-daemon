package main

import (
	"fmt"
	"strings"
	"time"
)

func validateRunOnceArgs(runOnce, dayArg string) error {
	dayArg = strings.TrimSpace(dayArg)
	if dayArg == "" {
		return nil
	}

	if strings.ToLower(strings.TrimSpace(runOnce)) != "daily" {
		return fmt.Errorf("flag -day can only be used with -run daily")
	}

	return nil
}

func resolveDailyRunDay(dayArg string, now time.Time, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.Local
	}

	dayArg = strings.TrimSpace(dayArg)
	if dayArg == "" {
		return now.In(loc), nil
	}

	day, err := time.ParseInLocation(time.DateOnly, dayArg, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day %q, expected YYYY-MM-DD: %w", dayArg, err)
	}

	return day, nil
}
