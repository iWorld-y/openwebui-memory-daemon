package logx

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func New(level string, path string) (*slog.Logger, func() error, error) {
	lvl := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}

	var w io.Writer = os.Stdout
	closeFn := func() error { return nil }
	if strings.TrimSpace(path) != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, nil, err
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, err
		}
		w = io.MultiWriter(os.Stdout, f)
		closeFn = f.Close
	}

	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl})
	return slog.New(h), closeFn, nil
}
