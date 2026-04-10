package owui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	memapp "github.com/iWorld-y/owui-memory-daemon/internal/memoryops/application"
)

type Client struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *resty.Client
}

type unixTime struct {
	time.Time
}

func (t *unixTime) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		t.Time = time.Time{}
		return nil
	}

	// number (unix seconds)
	if s[0] != '"' {
		var sec int64
		if err := json.Unmarshal(b, &sec); err != nil {
			return err
		}
		t.Time = time.Unix(sec, 0).UTC()
		return nil
	}

	// string (RFC3339 or numeric)
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	str = strings.TrimSpace(str)
	if str == "" {
		t.Time = time.Time{}
		return nil
	}

	if ts, err := time.Parse(time.RFC3339, str); err == nil {
		t.Time = ts
		return nil
	}
	var sec int64
	if err := json.Unmarshal([]byte(str), &sec); err == nil {
		t.Time = time.Unix(sec, 0).UTC()
		return nil
	}
	return fmt.Errorf("invalid time value: %s", s)
}

func NewClient(baseURL string, apiKey string, timeout time.Duration, logger *slog.Logger) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	rc := resty.New().
		SetBaseURL(u.String()).
		SetTimeout(timeout).
		SetHeader("Accept", "application/json").
		SetDebug(true).
		SetLogger(slogRestyLogger{logger: logger})
	if strings.TrimSpace(apiKey) != "" {
		rc.SetAuthToken(apiKey)
	}
	return &Client{
		baseURL:    u,
		apiKey:     apiKey,
		httpClient: rc,
	}, nil
}

type chatListItemDTO struct {
	ID        string   `json:"id"`
	UpdatedAt unixTime `json:"updated_at"`
}

type chatDTO struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Chat  struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	} `json:"chat"`
	Timestamp unixTime `json:"timestamp"`
}

type memoryDTO struct {
	ID        string   `json:"id"`
	Content   string   `json:"content"`
	CreatedAt unixTime `json:"created_at"`
	UpdatedAt unixTime `json:"updated_at"`
}

func (c *Client) ListChats(ctx context.Context) ([]memapp.ChatListItem, error) {
	var out []chatListItemDTO
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/chats/list", nil, &out); err != nil {
		return nil, err
	}
	res := make([]memapp.ChatListItem, 0, len(out))
	for _, it := range out {
		res = append(res, memapp.ChatListItem{ID: it.ID, UpdatedAt: it.UpdatedAt.Time})
	}
	return res, nil
}

func (c *Client) GetChat(ctx context.Context, id string) (*memapp.Chat, error) {
	var out chatDTO
	p := "/api/v1/chats/" + url.PathEscape(id)
	if err := c.doJSON(ctx, http.MethodGet, p, nil, &out); err != nil {
		return nil, err
	}
	msgs := make([]memapp.ChatMessage, 0, len(out.Chat.Messages))
	for _, m := range out.Chat.Messages {
		msgs = append(msgs, memapp.ChatMessage{Role: m.Role, Content: m.Content})
	}
	return &memapp.Chat{
		ID:        out.ID,
		Title:     out.Title,
		Messages:  msgs,
		UpdatedAt: out.Timestamp.Time,
	}, nil
}

func (c *Client) ListMemories(ctx context.Context) ([]memapp.Memory, error) {
	var out []memoryDTO
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/memories/", nil, &out); err != nil {
		return nil, err
	}
	res := make([]memapp.Memory, 0, len(out))
	for _, m := range out {
		res = append(res, memapp.Memory{
			ID:        m.ID,
			Content:   m.Content,
			CreatedAt: m.CreatedAt.Time,
			UpdatedAt: m.UpdatedAt.Time,
		})
	}
	return res, nil
}

func (c *Client) AddMemory(ctx context.Context, content string) error {
	body := map[string]string{"content": content}
	return c.doJSON(ctx, http.MethodPost, "/api/v1/memories/add", body, nil)
}

func (c *Client) DeleteMemory(ctx context.Context, id string) error {
	p := "/api/v1/memories/" + url.PathEscape(id)
	return c.doJSON(ctx, http.MethodDelete, p, nil, nil)
}

func (c *Client) doJSON(ctx context.Context, method string, pth string, in any, out any) error {
	req := c.httpClient.R().SetContext(ctx)
	if in != nil {
		req.SetBody(in)
	}
	if out != nil {
		req.SetResult(out)
	}

	resp, err := req.Execute(method, pth)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("owui http %d: %s", resp.StatusCode(), strings.TrimSpace(resp.String()))
	}
	return nil
}

type slogRestyLogger struct {
	logger *slog.Logger
}

func (l slogRestyLogger) Errorf(format string, args ...any) {
	l.loggerOrDefault().Error(fmt.Sprintf(format, args...))
}

func (l slogRestyLogger) Warnf(format string, args ...any) {
	l.loggerOrDefault().Warn(fmt.Sprintf(format, args...))
}

func (l slogRestyLogger) Debugf(format string, args ...any) {
	l.loggerOrDefault().Debug(fmt.Sprintf(format, args...))
}

func (l slogRestyLogger) loggerOrDefault() *slog.Logger {
	if l.logger == nil {
		return slog.Default()
	}
	return l.logger
}

var _ memapp.OWUIPort = (*Client)(nil)
