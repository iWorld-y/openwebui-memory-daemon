package memory

import (
	"testing"
	"time"
)

func TestKindFromContent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		content string
		want    Kind
	}{
		{"daily", "📋 2026-04-09 对话总结：xxx", KindDaily},
		{"weekly", "📦 2026-W14 周总结：xxx", KindWeekly},
		{"monthly", "📅 2026-04 月总结：xxx", KindMonthly},
		{"manual", "用户手动记忆：xxx", KindNone},
		{"empty", "", KindNone},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := KindFromContent(tc.content); got != tc.want {
				t.Fatalf("KindFromContent(%q)=%v want %v", tc.content, got, tc.want)
			}
		})
	}
}

func TestParseDailyDate(t *testing.T) {
	t.Parallel()

	got, ok := ParseDailyDate("📋 2026-04-09 对话总结：xxx", time.UTC)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	want := time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %s want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
	}

	if _, ok := ParseDailyDate("📋 2026/04/09 对话总结：xxx", time.UTC); ok {
		t.Fatalf("expected ok=false for unsupported date format")
	}
}

func TestParseWeeklyKey(t *testing.T) {
	t.Parallel()

	y, w, ok := ParseWeeklyKey("📦 2026-W14 周总结：xxx")
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if y != 2026 || w != 14 {
		t.Fatalf("got year=%d week=%d want 2026/14", y, w)
	}
}

func TestParseMonthlyKey(t *testing.T) {
	t.Parallel()

	y, m, ok := ParseMonthlyKey("📅 2026-04 月总结：xxx")
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if y != 2026 || m != 4 {
		t.Fatalf("got year=%d month=%d want 2026/4", y, m)
	}
}

