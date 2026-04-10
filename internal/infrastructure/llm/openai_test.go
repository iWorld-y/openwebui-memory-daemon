package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseChatCompletionContent(t *testing.T) {
	t.Parallel()

	body := []byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`)
	got, ok := parseChatCompletionContent(body)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if got != "hello" {
		t.Fatalf("got %q want %q", got, "hello")
	}
}

func TestClientSummarizeConsumesStreamingChatCompletions(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/chat/completions"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Authorization"), "Bearer secret"; got != want {
			t.Fatalf("authorization = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Accept"), "text/event-stream"; got != want {
			t.Fatalf("accept = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Content-Type"), "application/json"; got != want {
			t.Fatalf("content-type = %q, want %q", got, want)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll(request body) error = %v", err)
		}
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("Unmarshal(request body) error = %v", err)
		}
		if got, ok := reqBody["stream"].(bool); !ok || !got {
			t.Fatalf("request stream = %#v, want true", reqBody["stream"])
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, ": keep-alive\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret", "gpt-test", 64, time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	got, err := client.Summarize(context.Background(), "summarize me")
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}
	if got != "hello world" {
		t.Fatalf("got %q, want %q", got, "hello world")
	}
}
