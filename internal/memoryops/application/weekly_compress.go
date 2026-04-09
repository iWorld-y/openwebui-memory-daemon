package application

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	memory "github.com/iWorld-y/owui-memory-daemon/internal/memoryops/domain"
)

type WeeklyCompressor struct {
	OWUI     OWUIPort
	LLM      LLMPort
	Retry    RetryPort
	Logger   LoggerPort
	Loc      *time.Location
	Snapshot SnapshotPort
}

func (u *WeeklyCompressor) Run(ctx context.Context, ref time.Time) error {
	log := u.logger()
	loc := u.loc()
	ref = ref.In(loc)

	year, week := ref.ISOWeek()
	weekKey := fmt.Sprintf("%04d-W%02d", year, week)
	weekStart := memory.ISOWeekStart(loc, year, week)
	weekEnd := weekStart.AddDate(0, 0, 6)
	weekStartKey := weekStart.Format(time.DateOnly)
	weekEndKey := weekEnd.Format(time.DateOnly)

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
		date    time.Time
	}
	var ds []picked
	for _, m := range mems {
		if memory.KindFromContent(m.Content) != memory.KindDaily {
			continue
		}
		d, ok := memory.ParseDailyDate(m.Content, loc)
		if !ok {
			continue
		}
		y, w := d.ISOWeek()
		if y == year && w == week {
			ds = append(ds, picked{id: m.ID, content: m.Content, date: d})
		}
	}

	if len(ds) == 0 {
		log.Info("weekly: no daily summaries", "week", weekKey)
		return nil
	}

	sort.Slice(ds, func(i, j int) bool { return ds[i].date.Before(ds[j].date) })

	var b strings.Builder
	for _, d := range ds {
		b.WriteString(d.content)
		b.WriteString("\n")
	}
	prompt := WeeklyPrompt(weekStartKey, weekEndKey, b.String())

	var summary string
	if err := u.do(ctx, func(ctx context.Context) error {
		var err error
		summary, err = u.LLM.Summarize(ctx, prompt)
		return err
	}); err != nil {
		return err
	}

	content := fmt.Sprintf("📦 %s 周总结：%s", weekKey, strings.TrimSpace(summary))
	if err := u.do(ctx, func(ctx context.Context) error {
		return u.OWUI.AddMemory(ctx, content)
	}); err != nil {
		return err
	}

	for _, d := range ds {
		if err := u.do(ctx, func(ctx context.Context) error {
			return u.OWUI.DeleteMemory(ctx, d.id)
		}); err != nil {
			log.Warn("weekly: delete daily summary failed", "id", d.id, "err", err)
		}
	}

	log.Info("weekly: memory added", "week", weekKey, "merged", len(ds))

	if u.Snapshot != nil {
		if err := u.Snapshot.SnapshotAndPush(ctx, time.Now().In(loc)); err != nil {
			log.Warn("snapshot push failed", "task", "weekly", "err", err)
		} else {
			log.Info("snapshot pushed", "task", "weekly")
		}
	}

	return nil
}

func (u *WeeklyCompressor) loc() *time.Location {
	if u.Loc == nil {
		return time.Local
	}
	return u.Loc
}

func (u *WeeklyCompressor) logger() LoggerPort {
	if u.Logger == nil {
		return SlogLogger{}
	}
	return u.Logger
}

func (u *WeeklyCompressor) do(ctx context.Context, fn func(context.Context) error) error {
	if u.Retry == nil {
		return fn(ctx)
	}
	return u.Retry.Do(ctx, fn)
}
