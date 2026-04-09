package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Repo struct {
	Path        string
	Remote      string
	Branch      string
	AuthorName  string
	AuthorEmail string
}

func (r *Repo) Ensure(ctx context.Context) error {
	if strings.TrimSpace(r.Path) == "" {
		return fmt.Errorf("git repo path is empty")
	}
	if err := os.MkdirAll(r.Path, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(r.Path, ".git")); err == nil {
		return nil
	}
	if err := r.run(ctx, "git", "init"); err != nil {
		return err
	}
	readme := filepath.Join(r.Path, "README.md")
	if _, err := os.Stat(readme); err != nil {
		_ = os.WriteFile(readme, []byte("# owui memories backup\n\nThis repo is managed by owui-memory-daemon.\n"), 0o644)
	}
	return nil
}

func (r *Repo) Add(ctx context.Context, paths ...string) error {
	args := append([]string{"add", "--"}, paths...)
	return r.run(ctx, "git", args...)
}

func (r *Repo) Commit(ctx context.Context, message string) error {
	if strings.TrimSpace(message) == "" {
		message = "snapshot"
	}
	return r.run(ctx, "git", "commit", "-m", message)
}

func (r *Repo) Push(ctx context.Context) error {
	remote := strings.TrimSpace(r.Remote)
	branch := strings.TrimSpace(r.Branch)
	if remote == "" || branch == "" {
		return nil
	}
	return r.run(ctx, "git", "push", remote, "HEAD:"+branch)
}

func (r *Repo) SnapshotMessage(now time.Time) string {
	return fmt.Sprintf("snapshot: %s", now.Format("2006-01-02_15:04:05"))
}

func (r *Repo) run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = r.Path
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME="+r.AuthorName,
		"GIT_AUTHOR_EMAIL="+r.AuthorEmail,
		"GIT_COMMITTER_NAME="+r.AuthorName,
		"GIT_COMMITTER_EMAIL="+r.AuthorEmail,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s failed: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

