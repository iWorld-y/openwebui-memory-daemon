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

type WeeklyCompressor struct {
	OWUI   *owui.Client
	LLM    *llm.Client
	Policy retry.Policy
	Logger *slog.Logger
	Loc    *time.Location
}

func (t *WeeklyCompressor) Run(ctx context.Context, ref time.Time) error {
	log := t.Logger
	if log == nil {
		log = slog.Default()
	}
	loc := t.Loc
	if loc == nil {
		loc = time.Local
	}
	ref = ref.In(loc)
	year, week := ref.ISOWeek()
	weekKey := fmt.Sprintf("%04d-W%02d", year, week)

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
	prompt := WeeklyPrompt("📦 "+weekKey, b.String())

	var summary string
	if err := retry.Do(ctx, t.Policy, func(ctx context.Context) error {
		var err error
		summary, err = t.LLM.Summarize(ctx, prompt)
		return err
	}); err != nil {
		return err
	}

	content := fmt.Sprintf("📦 %s 周总结：%s", weekKey, strings.TrimSpace(summary))
	if err := retry.Do(ctx, t.Policy, func(ctx context.Context) error {
		return t.OWUI.AddMemory(ctx, content)
	}); err != nil {
		return err
	}

	for _, d := range ds {
		if err := retry.Do(ctx, t.Policy, func(ctx context.Context) error {
			return t.OWUI.DeleteMemory(ctx, d.id)
		}); err != nil {
			log.Warn("weekly: delete daily summary failed", "id", d.id, "err", err)
		}
	}

	log.Info("weekly: memory added", "week", weekKey, "merged", len(ds))
	return nil
}

