package owui

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientListChatsLogsDebugOutputViaLogger(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("authorization header = %q, want %q", got, "Bearer secret")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"chat-1","updated_at":"2026-04-10T08:00:00Z"}]`))
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret", time.Second, logger)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	chats, err := client.ListChats(context.Background())
	if err != nil {
		t.Fatalf("ListChats() error = %v", err)
	}
	if len(chats) != 1 {
		t.Fatalf("len(chats) = %d, want %d", len(chats), 1)
	}

	gotLogs := logs.String()
	if !strings.Contains(gotLogs, "level=DEBUG") {
		t.Fatalf("expected debug log level in logs, got %q", gotLogs)
	}
	if !strings.Contains(gotLogs, "/api/v1/chats/list") {
		t.Fatalf("expected request path in logs, got %q", gotLogs)
	}
}

func TestClientListChatsReturnsStatusAndBodyOnHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream exploded", http.StatusBadGateway)
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, "", time.Second, nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.ListChats(context.Background())
	if err == nil {
		t.Fatalf("ListChats() error = nil, want non-nil")
	}
	if got := err.Error(); !strings.Contains(got, "owui http 502: upstream exploded") {
		t.Fatalf("error = %q, want status and body", got)
	}
}

func TestClientListMemoriesParsesTimestamps(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/api/v1/memories/"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"mem-1","content":"plain memory","created_at":1772632763,"updated_at":"2026-03-03T12:00:00Z"}]`))
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, "", time.Second, nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	memories, err := client.ListMemories(context.Background())
	if err != nil {
		t.Fatalf("ListMemories() error = %v", err)
	}
	if len(memories) != 1 {
		t.Fatalf("len(memories) = %d, want %d", len(memories), 1)
	}

	got := memories[0]
	if got.ID != "mem-1" {
		t.Fatalf("memory id = %q, want %q", got.ID, "mem-1")
	}
	if got.Content != "plain memory" {
		t.Fatalf("memory content = %q, want %q", got.Content, "plain memory")
	}

	wantCreatedAt := time.Unix(1772632763, 0).UTC()
	if !got.CreatedAt.Equal(wantCreatedAt) {
		t.Fatalf("created_at = %s, want %s", got.CreatedAt.Format(time.RFC3339), wantCreatedAt.Format(time.RFC3339))
	}

	wantUpdatedAt := time.Date(2026, time.March, 3, 12, 0, 0, 0, time.UTC)
	if !got.UpdatedAt.Equal(wantUpdatedAt) {
		t.Fatalf("updated_at = %s, want %s", got.UpdatedAt.Format(time.RFC3339), wantUpdatedAt.Format(time.RFC3339))
	}
}
