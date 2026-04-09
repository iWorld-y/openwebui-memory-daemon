package application

import (
	"context"
	"time"

	memapp "github.com/iWorld-y/owui-memory-daemon/internal/memoryops/application"
)

type MemorySourcePort interface {
	ListMemories(ctx context.Context) ([]Memory, error)
}

type FileWriterPort interface {
	WriteFile(path string, data []byte, perm uint32) error
}

type RepoPort interface {
	Ensure(ctx context.Context) error
	Add(ctx context.Context, paths ...string) error
	Commit(ctx context.Context, message string) error
	Push(ctx context.Context) error
	SnapshotMessage(now time.Time) string
	Path() string
}

type Memory = memapp.Memory
