package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/iWorld-y/owui-memory-daemon/internal/config"
	"github.com/iWorld-y/owui-memory-daemon/internal/git"
	"github.com/iWorld-y/owui-memory-daemon/internal/llm"
	"github.com/iWorld-y/owui-memory-daemon/internal/logx"
	"github.com/iWorld-y/owui-memory-daemon/internal/owui"
	"github.com/iWorld-y/owui-memory-daemon/internal/retry"
	"github.com/iWorld-y/owui-memory-daemon/internal/snapshot"
	"github.com/iWorld-y/owui-memory-daemon/internal/task"
)

func main() {
	var cfgPath string
	var runOnce string
	flag.StringVar(&cfgPath, "config", "./config.yaml", "config yaml path")
	flag.StringVar(&runOnce, "run", "", "run once: daily|weekly|monthly")
	flag.Parse()

	cfg, err := config.Load(cfgPath)
	if err != nil {
		slog.Error("load config failed", "err", err)
		os.Exit(1)
	}

	logger, closeLog, err := logx.New(cfg.Log.Level, cfg.Log.Path)
	if err != nil {
		slog.Error("init logger failed", "err", err)
		os.Exit(1)
	}
	defer func() { _ = closeLog() }()
	slog.SetDefault(logger)

	loc := time.Local

	owuiClient, err := owui.NewClient(cfg.OpenWebUI.BaseURL, cfg.OpenWebUI.APIKey, 30*time.Second)
	if err != nil {
		slog.Error("init openwebui client failed", "err", err)
		os.Exit(1)
	}

	timeout := 60 * time.Second
	if cfg.LLM.TimeoutSec > 0 {
		timeout = time.Duration(cfg.LLM.TimeoutSec) * time.Second
	}
	llmClient, err := llm.NewClient(cfg.LLM.BaseURL, cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.MaxTokens, timeout)
	if err != nil {
		slog.Error("init llm client failed", "err", err)
		os.Exit(1)
	}

	repo := &git.Repo{
		Path:        cfg.Git.RepoPath,
		Remote:      cfg.Git.Remote,
		Branch:      cfg.Git.Branch,
		AuthorName:  cfg.Git.AuthorName,
		AuthorEmail: cfg.Git.AuthorEmail,
	}

	policy := retry.Policy{
		MaxAttempts: 3,
		Delays:      []time.Duration{10 * time.Second, 30 * time.Second},
	}

	daily := &task.DailySummarizer{OWUI: owuiClient, LLM: llmClient, Policy: policy, Logger: logger, Loc: loc}
	weekly := &task.WeeklyCompressor{OWUI: owuiClient, LLM: llmClient, Policy: policy, Logger: logger, Loc: loc}
	monthly := &task.MonthlyCompressor{OWUI: owuiClient, LLM: llmClient, Policy: policy, Logger: logger, Loc: loc}

	runAndSnapshot := func(ctx context.Context, name string, fn func(context.Context) error) {
		if err := fn(ctx); err != nil {
			slog.Error("task failed", "task", name, "err", err)
			return
		}
		if err := snapshot.SnapshotAndPush(ctx, owuiClient, repo, time.Now().In(loc)); err != nil {
			slog.Warn("snapshot push failed", "task", name, "err", err)
		} else {
			slog.Info("snapshot pushed", "task", name)
		}
	}

	if runOnce != "" {
		ctx := context.Background()
		switch strings.ToLower(strings.TrimSpace(runOnce)) {
		case "daily":
			day := time.Now().In(loc).AddDate(0, 0, -1)
			runAndSnapshot(ctx, "daily", func(ctx context.Context) error { return daily.Run(ctx, day) })
		case "weekly":
			runAndSnapshot(ctx, "weekly", func(ctx context.Context) error { return weekly.Run(ctx, time.Now().In(loc)) })
		case "monthly":
			runAndSnapshot(ctx, "monthly", func(ctx context.Context) error { return monthly.Run(ctx, time.Now().In(loc)) })
		default:
			slog.Error("unknown run value", "run", runOnce)
			os.Exit(2)
		}
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	c := cron.New(cron.WithParser(parser))

	if _, err := c.AddFunc(cfg.Schedule.Daily, func() {
		runAndSnapshot(context.Background(), "daily", func(ctx context.Context) error {
			return daily.Run(ctx, time.Now().In(loc).AddDate(0, 0, -1))
		})
	}); err != nil {
		slog.Error("add daily cron failed", "err", err)
		os.Exit(1)
	}
	if _, err := c.AddFunc(cfg.Schedule.Weekly, func() {
		runAndSnapshot(context.Background(), "weekly", func(ctx context.Context) error {
			return weekly.Run(ctx, time.Now().In(loc))
		})
	}); err != nil {
		slog.Error("add weekly cron failed", "err", err)
		os.Exit(1)
	}
	if _, err := c.AddFunc(cfg.Schedule.Monthly, func() {
		runAndSnapshot(context.Background(), "monthly", func(ctx context.Context) error {
			return monthly.Run(ctx, time.Now().In(loc))
		})
	}); err != nil {
		slog.Error("add monthly cron failed", "err", err)
		os.Exit(1)
	}

	c.Start()
	defer c.Stop()

	slog.Info("daemon started", "daily", cfg.Schedule.Daily, "weekly", cfg.Schedule.Weekly, "monthly", cfg.Schedule.Monthly)
	<-ctx.Done()
	slog.Info("daemon stopped", "at", time.Now().Format(time.RFC3339))
}

