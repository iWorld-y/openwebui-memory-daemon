package task

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/iWorld-y/owui-memory-daemon/internal/llm"
	"github.com/iWorld-y/owui-memory-daemon/internal/memory"
	"github.com/iWorld-y/owui-memory-daemon/internal/owui"
	"github.com/iWorld-y/owui-memory-daemon/internal/retry"
)

type MonthlyCompressor struct {
	OWUI   *owui.Client
	LLM    *llm.Client
	Policy retry.Policy
	Logger *slog.Logger
	Loc    *time.Location
}

func (t *MonthlyCompressor) Run(ctx context.Context, ref time.Time) error {
	log := t.Logger
	if log == nil {
		log = slog.Default()
	}
	loc := t.Loc
	if loc == nil {
		loc = time.Local
	}
	ref = ref.In(loc)

	// target is previous month per design
	firstOfMonth := time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, loc)
	targetEnd := firstOfMonth
	targetStart := firstOfMonth.AddDate(0, -1, 0)
	monthKey := targetStart.Format("2006-01")

	var mems []owui.Memory
	if err := retry.Do(ctx, t.Policy, func(ctx context.Context) error {
		var err error
		mems, err = t.OWUI.ListMemories(ctx)
		return err
	}); err != nil {
		return err
	}

	type picked struct {
		id      string
		content string
		start   time.Time
	}
	var ws []picked
	for _, m := range mems {
		if memory.KindFromContent(m.Content) != memory.KindWeekly {
			continue
		}
		y, w, ok := memory.ParseWeeklyKey(m.Content)
		if !ok {
			continue
		}
		start := isoWeekStart(loc, y, w)
		if start.IsZero() {
			continue
		}
		if start.Before(targetStart) || !start.Before(targetEnd) {
			continue
		}

		ws = append(ws, picked{id: m.ID, content: m.Content, start: start})
	}

	if len(ws) == 0 {
		log.Info("monthly: no weekly summaries", "month", monthKey, "range_start", targetStart.Format("2006-01-02"), "range_end", targetEnd.Format("2006-01-02"))
		return nil
	}

	sort.Slice(ws, func(i, j int) bool { return ws[i].start.Before(ws[j].start) })

	var b strings.Builder
	for _, w := range ws {
		b.WriteString(w.content)
		b.WriteString("\n")
	}
	prompt := MonthlyPrompt("📅 "+monthKey, b.String())

	var summary string
	if err := retry.Do(ctx, t.Policy, func(ctx context.Context) error {
		var err error
		summary, err = t.LLM.Summarize(ctx, prompt)
		return err
	}); err != nil {
		return err
	}

	content := fmt.Sprintf("📅 %s 月总结：%s", monthKey, strings.TrimSpace(summary))
	if err := retry.Do(ctx, t.Policy, func(ctx context.Context) error {
		return t.OWUI.AddMemory(ctx, content)
	}); err != nil {
		return err
	}

	for _, w := range ws {
		if err := retry.Do(ctx, t.Policy, func(ctx context.Context) error {
			return t.OWUI.DeleteMemory(ctx, w.id)
		}); err != nil {
			log.Warn("monthly: delete weekly summary failed", "id", w.id, "err", err)
		}
	}

	log.Info("monthly: memory added", "month", monthKey, "merged", len(ws))
	return nil
}

func isoWeekStart(loc *time.Location, year int, week int) time.Time {
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

