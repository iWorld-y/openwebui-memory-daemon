package application

import (
	"context"
	"reflect"
	"testing"
	"time"
)

type fakeOWUI struct {
	memories []Memory
	added    []string
	deleted  []string
}

func (f *fakeOWUI) ListChats(context.Context) ([]ChatListItem, error) { return nil, nil }

func (f *fakeOWUI) GetChat(context.Context, string) (*Chat, error) { return nil, nil }

func (f *fakeOWUI) ListMemories(context.Context) ([]Memory, error) { return f.memories, nil }

func (f *fakeOWUI) AddMemory(_ context.Context, content string) error {
	f.added = append(f.added, content)
	return nil
}

func (f *fakeOWUI) DeleteMemory(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	return nil
}

type fakeLLM struct {
	summary string
	prompts []string
}

func (f *fakeLLM) Summarize(_ context.Context, prompt string) (string, error) {
	f.prompts = append(f.prompts, prompt)
	return f.summary, nil
}

type noopLogger struct{}

func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

func TestWeeklyCompressorUsesMemoryTimestamps(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("UTC+8", 8*60*60)
	owui := &fakeOWUI{
		memories: []Memory{
			{
				ID:        "daily-1",
				Content:   "【想法类记忆 - AI与职业发展】第一条",
				CreatedAt: time.Date(2026, time.March, 31, 23, 30, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, time.March, 31, 23, 30, 0, 0, time.UTC),
			},
			{
				ID:        "daily-2",
				Content:   "【项目记忆】第二条",
				CreatedAt: time.Date(2026, time.April, 4, 8, 0, 0, 0, loc),
				UpdatedAt: time.Date(2026, time.April, 4, 8, 0, 0, 0, loc),
			},
			{
				ID:        "other-week",
				Content:   "【不在本周】",
				CreatedAt: time.Date(2026, time.April, 6, 9, 0, 0, 0, loc),
				UpdatedAt: time.Date(2026, time.April, 6, 9, 0, 0, 0, loc),
			},
		},
	}
	llm := &fakeLLM{summary: "本周总结"}
	u := &WeeklyCompressor{
		OWUI:   owui,
		LLM:    llm,
		Logger: noopLogger{},
		Loc:    loc,
	}

	ref := time.Date(2026, time.April, 1, 12, 0, 0, 0, loc)
	if err := u.Run(context.Background(), ref); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got, want := owui.added, []string{"📦 2026-W14 周总结：本周总结"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("added memories = %#v, want %#v", got, want)
	}
	if got, want := owui.deleted, []string{"daily-1", "daily-2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("deleted memories = %#v, want %#v", got, want)
	}
}

func TestMonthlyCompressorUsesMemoryTimestamps(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("UTC+8", 8*60*60)
	owui := &fakeOWUI{
		memories: []Memory{
			{
				ID:        "weekly-1",
				Content:   "任意周总结一",
				CreatedAt: time.Date(2026, time.March, 3, 9, 0, 0, 0, loc),
				UpdatedAt: time.Date(2026, time.March, 3, 9, 0, 0, 0, loc),
			},
			{
				ID:        "weekly-2",
				Content:   "任意周总结二",
				CreatedAt: time.Date(2026, time.March, 28, 20, 0, 0, 0, loc),
				UpdatedAt: time.Date(2026, time.March, 28, 20, 0, 0, 0, loc),
			},
			{
				ID:        "other-month",
				Content:   "不在目标月份",
				CreatedAt: time.Date(2026, time.April, 2, 8, 0, 0, 0, loc),
				UpdatedAt: time.Date(2026, time.April, 2, 8, 0, 0, 0, loc),
			},
		},
	}
	llm := &fakeLLM{summary: "本月总结"}
	u := &MonthlyCompressor{
		OWUI:   owui,
		LLM:    llm,
		Logger: noopLogger{},
		Loc:    loc,
	}

	ref := time.Date(2026, time.April, 10, 12, 0, 0, 0, loc)
	if err := u.Run(context.Background(), ref); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got, want := owui.added, []string{"📅 2026-03 月总结：本月总结"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("added memories = %#v, want %#v", got, want)
	}
	if got, want := owui.deleted, []string{"weekly-1", "weekly-2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("deleted memories = %#v, want %#v", got, want)
	}
}
