package snapshot

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/iWorld-y/owui-memory-daemon/internal/git"
	"github.com/iWorld-y/owui-memory-daemon/internal/owui"
)

func SnapshotAndPush(ctx context.Context, owuiClient *owui.Client, repo *git.Repo, now time.Time) error {
	if err := repo.Ensure(ctx); err != nil {
		return err
	}
	mems, err := owuiClient.ListMemories(ctx)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(mems, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(repo.Path, "memories.json"), b, 0o644); err != nil {
		return err
	}
	if err := repo.Add(ctx, "memories.json", "README.md"); err != nil {
		return err
	}
	// Commit may fail if nothing changed; treat as success.
	if err := repo.Commit(ctx, repo.SnapshotMessage(now)); err != nil {
		// git returns non-zero when there's nothing to commit; ignore.
		// We detect by message to avoid importing extra plumbing.
		if !containsNothingToCommit(err.Error()) {
			return err
		}
	}
	// Push may fail per design; caller decides whether to log and continue.
	return repo.Push(ctx)
}

func containsNothingToCommit(s string) bool {
	s = strings.ToLower(s)
	return strings.Contains(s, "nothing to commit") || strings.Contains(s, "no changes added to commit")
}

