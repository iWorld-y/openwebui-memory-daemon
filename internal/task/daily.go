package task

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/iWorld-y/owui-memory-daemon/internal/llm"
	"github.com/iWorld-y/owui-memory-daemon/internal/owui"
	"github.com/iWorld-y/owui-memory-daemon/internal/retry"
)

type DailySummarizer struct {
	OWUI   *owui.Client
	LLM    *llm.Client
	Policy retry.Policy
	Logger *slog.Logger
	Loc    *time.Location
}

func (t *DailySummarizer) Run(ctx context.Context, day time.Time) error {
	log := t.Logger
	if log == nil {
		log = slog.Default()
	}
	loc := t.Loc
	if loc == nil {
		loc = time.Local
	}
	day = day.In(loc)
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)

	var chats []owui.ChatListItem
	if err := retry.Do(ctx, t.Policy, func(ctx context.Context) error {
		var err error
		chats, err = t.OWUI.ListChats(ctx)
		return err
	}); err != nil {
		return err
	}

	var ids []string
	for _, c := range chats {
		u := c.UpdatedAt.In(loc)
		if !u.Before(start) && u.Before(end) {
			ids = append(ids, c.ID)
		}
	}
	if len(ids) == 0 {
		log.Info("daily: no chats", "date", start.Format("2006-01-02"))
		return nil
	}

	var b strings.Builder
	for _, id := range ids {
		var chat *owui.Chat
		if err := retry.Do(ctx, t.Policy, func(ctx context.Context) error {
			var err error
			chat, err = t.OWUI.GetChat(ctx, id)
			return err
		}); err != nil {
			return err
		}

		b.WriteString("\n---\n")
		b.WriteString("对话ID: ")
		b.WriteString(chat.ID)
		if chat.Title != "" {
			b.WriteString(" 标题: ")
			b.WriteString(chat.Title)
		}
		b.WriteString("\n")
		for _, m := range chat.Messages {
			b.WriteString(m.Role)
			b.WriteString(": ")
			b.WriteString(m.Content)
			b.WriteString("\n")
		}
	}

	chatsText := b.String()
	// Best-effort truncation: keep 80% of bytes to leave room for prompt overhead.
	chatsText = TruncateKeepHeadTail(chatsText, int(float64(len(chatsText))*0.8))
	dateKey := start.Format("2006-01-02")
	prompt := DailyPrompt(dateKey, chatsText)

	var summary string
	if err := retry.Do(ctx, t.Policy, func(ctx context.Context) error {
		var err error
		summary, err = t.LLM.Summarize(ctx, prompt)
		return err
	}); err != nil {
		return err
	}

	content := fmt.Sprintf("📋 %s 对话总结：%s", dateKey, strings.TrimSpace(summary))
	if err := retry.Do(ctx, t.Policy, func(ctx context.Context) error {
		return t.OWUI.AddMemory(ctx, content)
	}); err != nil {
		return err
	}

	log.Info("daily: memory added", "date", dateKey)
	return nil
}

