package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	memapp "github.com/iWorld-y/owui-memory-daemon/internal/memoryops/application"
)

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type chatCompletionChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
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

func parseChatCompletionChunkContent(body []byte) (string, bool) {
	var r chatCompletionChunk
	if err := json.Unmarshal(body, &r); err != nil {
		return "", false
	}
	if len(r.Choices) == 0 {
		return "", false
	}
	if content := r.Choices[0].Delta.Content; content != "" {
		return content, true
	}
	if content := r.Choices[0].Message.Content; content != "" {
		return content, true
	}
	return "", false
}

func readChatCompletionStream(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var out strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			break
		}

		chunk, ok := parseChatCompletionChunkContent([]byte(data))
		if ok {
			out.WriteString(chunk)
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if out.Len() == 0 {
		return "", fmt.Errorf("llm: stream returned no content")
	}
	return out.String(), nil
}

type Client struct {
	baseURL    *url.URL
	apiKey     string
	model      string
	maxTokens  int
	httpClient *resty.Client
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
		httpClient: resty.New().
			SetBaseURL(u.String()).
			SetTimeout(timeout).
			SetHeader("Content-Type", "application/json"),
	}, nil
}

func (c *Client) Summarize(ctx context.Context, prompt string) (string, error) {
	u := *c.baseURL
	u.Path = path.Join(u.Path, "/chat/completions")

	reqBody := map[string]any{
		"model":      c.model,
		"max_tokens": c.maxTokens,
		"stream":     true,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	req := c.httpClient.R().
		SetContext(ctx).
		SetHeader("Accept", "text/event-stream").
		SetBody(reqBody).
		SetDoNotParseResponse(true)
	if strings.TrimSpace(c.apiKey) != "" {
		req.SetAuthToken(c.apiKey)
	}

	resp, err := req.Post(u.String())
	if err != nil {
		return "", err
	}
	if resp.RawBody() != nil {
		defer resp.RawBody().Close()
	}

	if resp.IsError() {
		body, readErr := io.ReadAll(resp.RawBody())
		if readErr != nil {
			return "", readErr
		}
		return "", fmt.Errorf("llm http %d: %s", resp.StatusCode(), strings.TrimSpace(string(body)))
	}

	contentType := strings.ToLower(resp.Header().Get("Content-Type"))
	if strings.Contains(contentType, "text/event-stream") {
		return readChatCompletionStream(resp.RawBody())
	}

	body, err := io.ReadAll(resp.RawBody())
	if err != nil {
		return "", err
	}

	out, ok := parseChatCompletionContent(body)
	if !ok {
		return "", fmt.Errorf("llm: unexpected response shape: %s", strings.TrimSpace(string(body)))
	}
	return out, nil
}

var _ memapp.LLMPort = (*Client)(nil)
