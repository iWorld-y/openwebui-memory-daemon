package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	memapp "github.com/iWorld-y/owui-memory-daemon/internal/memoryops/application"
)

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func parseChatCompletionContent(body []byte) (string, bool) {
	var r chatCompletionResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return "", false
	}
	if len(r.Choices) == 0 {
		return "", false
	}
	return r.Choices[0].Message.Content, true
}

type Client struct {
	baseURL    *url.URL
	apiKey     string
	model      string
	maxTokens  int
	httpClient *http.Client
}

func NewClient(baseURL string, apiKey string, model string, maxTokens int, timeout time.Duration) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	return &Client{
		baseURL:   u,
		apiKey:    apiKey,
		model:     model,
		maxTokens: maxTokens,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *Client) Summarize(ctx context.Context, prompt string) (string, error) {
	u := *c.baseURL
	u.Path = path.Join(u.Path, "/chat/completions")

	reqBody := map[string]any{
		"model":      c.model,
		"max_tokens": c.maxTokens,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(c.apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	out, ok := parseChatCompletionContent(body)
	if !ok {
		return "", fmt.Errorf("llm: unexpected response shape: %s", strings.TrimSpace(string(body)))
	}
	return out, nil
}

var _ memapp.LLMPort = (*Client)(nil)
