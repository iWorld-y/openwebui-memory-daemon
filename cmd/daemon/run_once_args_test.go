package main

import (
	"testing"
	"time"
)

func TestResolveDailyRunDayDefaultToNow(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, time.April, 10, 15, 4, 5, 0, time.UTC)

	got, err := resolveDailyRunDay("", now, loc)
	if err != nil {
		t.Fatalf("resolveDailyRunDay() error = %v", err)
	}

	want := now.In(loc)
	if !got.Equal(want) {
		t.Fatalf("resolveDailyRunDay() = %v, want %v", got, want)
	}
}

func TestResolveDailyRunDayParseDate(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, time.April, 10, 10, 0, 0, 0, loc)

	got, err := resolveDailyRunDay(" 2026-04-09 ", now, loc)
	if err != nil {
		t.Fatalf("resolveDailyRunDay() error = %v", err)
	}

	want := time.Date(2026, time.April, 9, 0, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("resolveDailyRunDay() = %v, want %v", got, want)
	}
}

func TestResolveDailyRunDayRejectInvalidDate(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, time.April, 10, 10, 0, 0, 0, loc)

	if _, err := resolveDailyRunDay("2026/04/09", now, loc); err == nil {
		t.Fatal("resolveDailyRunDay() expected error, got nil")
	}
}

func TestValidateRunOnceArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		runOnce string
		dayArg  string
		wantErr bool
	}{
		{name: "daily allows day", runOnce: "daily", dayArg: "2026-04-09", wantErr: false},
		{name: "weekly forbids day", runOnce: "weekly", dayArg: "2026-04-09", wantErr: true},
		{name: "monthly forbids day", runOnce: "monthly", dayArg: "2026-04-09", wantErr: true},
		{name: "weekly without day", runOnce: "weekly", dayArg: "", wantErr: false},
		{name: "daemon mode forbids day", runOnce: "", dayArg: "2026-04-09", wantErr: true},
		{name: "trimmed daily", runOnce: " Daily ", dayArg: "2026-04-09", wantErr: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateRunOnceArgs(tc.runOnce, tc.dayArg)
			if tc.wantErr && err == nil {
				t.Fatal("validateRunOnceArgs() expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("validateRunOnceArgs() unexpected error = %v", err)
			}
		})
	}
}
