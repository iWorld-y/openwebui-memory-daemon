package llm

import "testing"

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
