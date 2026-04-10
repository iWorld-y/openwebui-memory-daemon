package application

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type DailySummarizer struct {
	OWUI     OWUIPort
	LLM      LLMPort
	Retry    RetryPort
	Logger   LoggerPort
	Loc      *time.Location
	Snapshot SnapshotPort
}

func (u *DailySummarizer) Run(ctx context.Context, day time.Time) error {
	log := u.logger()
	loc := u.loc()
	day = day.In(loc)

	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)

	var chats []ChatListItem
	if err := u.do(ctx, func(ctx context.Context) error {
		var err error
		chats, err = u.OWUI.ListChats(ctx)
		return err
	}); err != nil {
		return err
	}

	var ids []string
	for _, c := range chats {
		updated := c.UpdatedAt.In(loc)
		if !updated.Before(start) && updated.Before(end) {
			ids = append(ids, c.ID)
		}
	}

	if len(ids) == 0 {
		log.Info("daily: no chats", "date", start.Format(time.DateOnly))
		return nil
	}

	var b strings.Builder
	for _, id := range ids {
		var chat *Chat
		if err := u.do(ctx, func(ctx context.Context) error {
			var err error
			chat, err = u.OWUI.GetChat(ctx, id)
			return err
		}); err != nil {
			return err
		}

		log.Info("chat.ID: %s", chat.ID)
		log.Info("chat.Title: %s", chat.Title)

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

	dateKey := start.Format(time.DateOnly)
	prompt := DailyPrompt(dateKey, chatsText)

	var summary string
	if err := u.do(ctx, func(ctx context.Context) error {
		var err error
		summary, err = u.LLM.Summarize(ctx, prompt)
		return err
	}); err != nil {
		return err
	}

	content := fmt.Sprintf("📋 %s 对话总结：%s", dateKey, strings.TrimSpace(summary))
	if err := u.do(ctx, func(ctx context.Context) error {
		return u.OWUI.AddMemory(ctx, content)
	}); err != nil {
		return err
	}

	log.Info("daily: memory added", "date", dateKey)

	if u.Snapshot != nil {
		if err := u.Snapshot.SnapshotAndPush(ctx, time.Now().In(loc)); err != nil {
			log.Warn("snapshot push failed", "task", "daily", "err", err)
		} else {
			log.Info("snapshot pushed", "task", "daily")
		}
	}

	return nil
}

func (u *DailySummarizer) loc() *time.Location {
	if u.Loc == nil {
		return time.Local
	}
	return u.Loc
}

func (u *DailySummarizer) logger() LoggerPort {
	if u.Logger == nil {
		return SlogLogger{}
	}
	return u.Logger
}

func (u *DailySummarizer) do(ctx context.Context, fn func(context.Context) error) error {
	if u.Retry == nil {
		return fn(ctx)
	}
	return u.Retry.Do(ctx, fn)
}
