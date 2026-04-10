package application

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	memory "github.com/iWorld-y/owui-memory-daemon/internal/memoryops/domain"
)

type MonthlyCompressor struct {
	OWUI     OWUIPort
	LLM      LLMPort
	Retry    RetryPort
	Logger   LoggerPort
	Loc      *time.Location
	Snapshot SnapshotPort
}

func (u *MonthlyCompressor) Run(ctx context.Context, ref time.Time) error {
	log := u.logger()
	loc := u.loc()
	ref = ref.In(loc)

	// target is previous month per design
	firstOfMonth := time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, loc)
	targetEnd := firstOfMonth
	targetStart := firstOfMonth.AddDate(0, -1, 0)
	monthKey := targetStart.Format("2006-01")

	var mems []Memory
	if err := u.do(ctx, func(ctx context.Context) error {
		var err error
		mems, err = u.OWUI.ListMemories(ctx)
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
		if memory.KindFromContent(m.Content) == memory.KindMonthly {
			continue
		}
		start := memoryTime(m, loc)
		if start.IsZero() {
			continue
		}
		if start.Before(targetStart) || !start.Before(targetEnd) {
			continue
		}

		ws = append(ws, picked{id: m.ID, content: m.Content, start: start})
	}

	if len(ws) == 0 {
		log.Info(
			"monthly: no weekly summaries",
			"month", monthKey,
			"range_start", targetStart.Format(time.DateOnly),
			"range_end", targetEnd.Format(time.DateOnly),
		)
		return nil
	}

	sort.Slice(ws, func(i, j int) bool { return ws[i].start.Before(ws[j].start) })

	var b strings.Builder
	for _, w := range ws {
		b.WriteString(w.content)
		b.WriteString("\n")
	}
	prompt := MonthlyPrompt(monthKey, b.String())

	var summary string
	if err := u.do(ctx, func(ctx context.Context) error {
		var err error
		summary, err = u.LLM.Summarize(ctx, prompt)
		return err
	}); err != nil {
		return err
	}

	content := fmt.Sprintf("📅 %s 月总结：%s", monthKey, strings.TrimSpace(summary))
	if err := u.do(ctx, func(ctx context.Context) error {
		return u.OWUI.AddMemory(ctx, content)
	}); err != nil {
		return err
	}

	for _, w := range ws {
		if err := u.do(ctx, func(ctx context.Context) error {
			return u.OWUI.DeleteMemory(ctx, w.id)
		}); err != nil {
			log.Warn("monthly: delete weekly summary failed", "id", w.id, "err", err)
		}
	}

	log.Info("monthly: memory added", "month", monthKey, "merged", len(ws))

	if u.Snapshot != nil {
		if err := u.Snapshot.SnapshotAndPush(ctx, time.Now().In(loc)); err != nil {
			log.Warn("snapshot push failed", "task", "monthly", "err", err)
		} else {
			log.Info("snapshot pushed", "task", "monthly")
		}
	}

	return nil
}

func (u *MonthlyCompressor) loc() *time.Location {
	if u.Loc == nil {
		return time.Local
	}
	return u.Loc
}

func (u *MonthlyCompressor) logger() LoggerPort {
	if u.Logger == nil {
		return SlogLogger{}
	}
	return u.Logger
}

func (u *MonthlyCompressor) do(ctx context.Context, fn func(context.Context) error) error {
	if u.Retry == nil {
		return fn(ctx)
	}
	return u.Retry.Do(ctx, fn)
}
