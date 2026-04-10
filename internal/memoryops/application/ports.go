package application

import (
	"context"
	"log/slog"
	"time"
)

type OWUIPort interface {
	ListChats(ctx context.Context) ([]ChatListItem, error)
	GetChat(ctx context.Context, id string) (*Chat, error)

	ListMemories(ctx context.Context) ([]Memory, error)
	AddMemory(ctx context.Context, content string) error
	DeleteMemory(ctx context.Context, id string) error
}

type LLMPort interface {
	Summarize(ctx context.Context, prompt string) (string, error)
}

type SnapshotPort interface {
	SnapshotAndPush(ctx context.Context, now time.Time) error
}

type RetryPort interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type LoggerPort interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type SlogLogger struct{ L *slog.Logger }

func (l SlogLogger) Info(msg string, args ...any)  { l.logger().Info(msg, args...) }
func (l SlogLogger) Warn(msg string, args ...any)  { l.logger().Warn(msg, args...) }
func (l SlogLogger) Error(msg string, args ...any) { l.logger().Error(msg, args...) }

func (l SlogLogger) logger() *slog.Logger {
	if l.L == nil {
		return slog.Default()
	}
	return l.L
}

// Types mirrored from OWUI adapter, to keep ports stable.
type ChatListItem struct {
	ID        string
	UpdatedAt time.Time
}

type Chat struct {
	ID        string
	Title     string
	Messages  []ChatMessage
	UpdatedAt time.Time
}

type ChatMessage struct {
	Role    string
	Content string
}

type Memory struct {
	ID        string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}
