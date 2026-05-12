package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Client struct {
	baseURL  string
	email    string
	password string
	hc       *http.Client
	log      *zap.Logger

	mu    sync.RWMutex
	token string
}

func New(baseURL, email, password string, log *zap.Logger) *Client {
	return &Client{
		baseURL:  baseURL,
		email:    email,
		password: password,
		hc:       &http.Client{Timeout: 20 * time.Second},
		log:      log,
	}
}

func (c *Client) Get(ctx context.Context, path string, query url.Values, out any) error {
	return c.do(ctx, http.MethodGet, path, query, nil, out)
}

func (c *Client) Post(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPost, path, nil, body, out)
}

func (c *Client) Put(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPut, path, nil, body, out)
}

func (c *Client) Patch(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPatch, path, nil, body, out)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	if err := c.ensureToken(ctx, false); err != nil {
		return err
	}

	resp, err := c.send(ctx, method, path, query, body)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		_ = resp.Body.Close()
		if err := c.ensureToken(ctx, true); err != nil {
			return err
		}
		resp, err = c.send(ctx, method, path, query, body)
		if err != nil {
			return err
		}
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, truncate(string(raw), 200))
	}

	if out == nil || len(raw) == 0 {
		return nil
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode %s %s: %w", method, path, err)
	}
	return nil
}

func (c *Client) send(ctx context.Context, method, path string, query url.Values, body any) (*http.Response, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}

	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		rdr = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), rdr)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if tok := c.tokenCopy(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	return c.hc.Do(req)
}

func (c *Client) tokenCopy() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

func (c *Client) ensureToken(ctx context.Context, force bool) error {
	if !force {
		if c.tokenCopy() != "" {
			return nil
		}
	}
	if c.email == "" || c.password == "" {
		return errors.New("service credentials not configured")
	}
	body := map[string]string{"email": c.email, "password": c.password}
	resp, err := c.send(ctx, http.MethodPost, "/api/v1/auth/login", nil, body)
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed: %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("login read: %w", err)
	}
	var env struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("login decode: %w", err)
	}
	if env.Data.AccessToken == "" {
		return errors.New("login: empty token")
	}
	c.mu.Lock()
	c.token = env.Data.AccessToken
	c.mu.Unlock()
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
