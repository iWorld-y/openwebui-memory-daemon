package owui

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

type Client struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL string, apiKey string, timeout time.Duration) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		baseURL: u,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

type chatListItemDTO struct {
	ID        string    `json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
}

type chatDTO struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	UpdatedAt time.Time `json:"updated_at"`
}

type memoryDTO struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

func (c *Client) ListChats(ctx context.Context) ([]memapp.ChatListItem, error) {
	var out []chatListItemDTO
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/chats/list", nil, &out); err != nil {
		return nil, err
	}
	res := make([]memapp.ChatListItem, 0, len(out))
	for _, it := range out {
		res = append(res, memapp.ChatListItem{ID: it.ID, UpdatedAt: it.UpdatedAt})
	}
	return res, nil
}

func (c *Client) GetChat(ctx context.Context, id string) (*memapp.Chat, error) {
	var out chatDTO
	p := "/api/v1/chats/" + url.PathEscape(id)
	if err := c.doJSON(ctx, http.MethodGet, p, nil, &out); err != nil {
		return nil, err
	}
	msgs := make([]memapp.ChatMessage, 0, len(out.Messages))
	for _, m := range out.Messages {
		msgs = append(msgs, memapp.ChatMessage{Role: m.Role, Content: m.Content})
	}
	return &memapp.Chat{
		ID:        out.ID,
		Title:     out.Title,
		Messages:  msgs,
		UpdatedAt: out.UpdatedAt,
	}, nil
}

func (c *Client) ListMemories(ctx context.Context) ([]memapp.Memory, error) {
	var out []memoryDTO
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/memories/", nil, &out); err != nil {
		return nil, err
	}
	res := make([]memapp.Memory, 0, len(out))
	for _, m := range out {
		res = append(res, memapp.Memory{ID: m.ID, Content: m.Content})
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
	u := *c.baseURL
	u.Path = path.Join(u.Path, pth)

	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(c.apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("owui http %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(b, out); err != nil {
		return err
	}
	return nil
}

var _ memapp.OWUIPort = (*Client)(nil)
