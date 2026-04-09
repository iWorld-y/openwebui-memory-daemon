package application

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Snapshotter struct {
	Memories MemorySourcePort
	Repo     RepoPort
}

func (s *Snapshotter) SnapshotAndPush(ctx context.Context, now time.Time) error {
	if err := s.Repo.Ensure(ctx); err != nil {
		return err
	}
	mems, err := s.Memories.ListMemories(ctx)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(mems, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(s.Repo.Path(), "memories.json"), b, 0o644); err != nil {
		return err
	}
	if err := s.Repo.Add(ctx, "memories.json", "README.md"); err != nil {
		return err
	}
	// Commit may fail if nothing changed; treat as success.
	if err := s.Repo.Commit(ctx, s.Repo.SnapshotMessage(now)); err != nil {
		// git returns non-zero when there's nothing to commit; ignore.
		if !containsNothingToCommit(err.Error()) {
			return err
		}
	}
	return s.Repo.Push(ctx)
}

func containsNothingToCommit(s string) bool {
	s = strings.ToLower(s)
	return strings.Contains(s, "nothing to commit") || strings.Contains(s, "no changes added to commit")
}
