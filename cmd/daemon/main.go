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

	memapp "github.com/iWorld-y/owui-memory-daemon/internal/memoryops/application"
	"github.com/iWorld-y/owui-memory-daemon/internal/snapshotting/application"

	"github.com/iWorld-y/owui-memory-daemon/internal/infrastructure/config"
	"github.com/iWorld-y/owui-memory-daemon/internal/infrastructure/gitrepo"
	"github.com/iWorld-y/owui-memory-daemon/internal/infrastructure/llm"
	"github.com/iWorld-y/owui-memory-daemon/internal/infrastructure/logx"
	"github.com/iWorld-y/owui-memory-daemon/internal/infrastructure/owui"
	"github.com/iWorld-y/owui-memory-daemon/internal/infrastructure/retry"
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

	repo := &gitrepo.Repo{
		RepoPath:    cfg.Git.RepoPath,
		Remote:      cfg.Git.Remote,
		Branch:      cfg.Git.Branch,
		AuthorName:  cfg.Git.AuthorName,
		AuthorEmail: cfg.Git.AuthorEmail,
	}

	policy := retry.Policy{
		MaxAttempts: 3,
		Delays:      []time.Duration{10 * time.Second, 30 * time.Second},
	}
	retryAdapter := retry.Adapter{Policy: policy}

	snapshotter := &application.Snapshotter{Memories: owuiClient, Repo: repo}

	daily := &memapp.DailySummarizer{
		OWUI:     owuiClient,
		LLM:      llmClient,
		Retry:    retryAdapter,
		Logger:   memapp.SlogLogger{L: logger},
		Loc:      loc,
		Snapshot: snapshotter,
	}
	weekly := &memapp.WeeklyCompressor{
		OWUI:     owuiClient,
		LLM:      llmClient,
		Retry:    retryAdapter,
		Logger:   memapp.SlogLogger{L: logger},
		Loc:      loc,
		Snapshot: snapshotter,
	}
	monthly := &memapp.MonthlyCompressor{
		OWUI:     owuiClient,
		LLM:      llmClient,
		Retry:    retryAdapter,
		Logger:   memapp.SlogLogger{L: logger},
		Loc:      loc,
		Snapshot: snapshotter,
	}

	if runOnce != "" {
		ctx := context.Background()
		switch strings.ToLower(strings.TrimSpace(runOnce)) {
		case "daily":
			day := time.Now().In(loc).AddDate(0, 0, -1)
			if err := daily.Run(ctx, day); err != nil {
				slog.Error("task failed", "task", "daily", "err", err)
			}
		case "weekly":
			if err := weekly.Run(ctx, time.Now().In(loc)); err != nil {
				slog.Error("task failed", "task", "weekly", "err", err)
			}
		case "monthly":
			if err := monthly.Run(ctx, time.Now().In(loc)); err != nil {
				slog.Error("task failed", "task", "monthly", "err", err)
			}
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
		if err := daily.Run(context.Background(), time.Now().In(loc).AddDate(0, 0, -1)); err != nil {
			slog.Error("task failed", "task", "daily", "err", err)
		}
	}); err != nil {
		slog.Error("add daily cron failed", "err", err)
		os.Exit(1)
	}
	if _, err := c.AddFunc(cfg.Schedule.Weekly, func() {
		if err := weekly.Run(context.Background(), time.Now().In(loc)); err != nil {
			slog.Error("task failed", "task", "weekly", "err", err)
		}
	}); err != nil {
		slog.Error("add weekly cron failed", "err", err)
		os.Exit(1)
	}
	if _, err := c.AddFunc(cfg.Schedule.Monthly, func() {
		if err := monthly.Run(context.Background(), time.Now().In(loc)); err != nil {
			slog.Error("task failed", "task", "monthly", "err", err)
		}
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
