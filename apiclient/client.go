// Package apiclient is a thin, dependency-free Go client for the opentdm REST
// API. It is shared by the CLI and the Go SDK so the resolve contract lives in
// exactly one place.
package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to an opentdm server with a service token.
type Client struct {
	Host  string // e.g. https://tdm.example.com
	Token string // service token (otdm_...)
	HTTP  *http.Client
}

// New constructs a client. A zero timeout uses a sensible default.
func New(host, token string) *Client {
	return &Client{
		Host:  strings.TrimRight(host, "/"),
		Token: token,
		HTTP:  &http.Client{Timeout: 30 * time.Second},
	}
}

// APIError is a non-2xx response.
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("opentdm: HTTP %d: %s", e.Status, strings.TrimSpace(e.Body))
}

// Resolve fetches the merged variables for a project+environment in the given
// format ("dotenv", "json", "shell", "yaml", "properties"; "" => json),
// returning the raw body and its content type.
func (c *Client) Resolve(ctx context.Context, project, env, format string) ([]byte, string, error) {
	q := url.Values{}
	q.Set("env", env)
	if format != "" {
		q.Set("format", format)
	}
	endpoint := fmt.Sprintf("%s/api/v1/projects/%s/resolve?%s", c.Host, url.PathEscape(project), q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, "", &APIError{Status: resp.StatusCode, Body: string(body)}
	}
	return body, resp.Header.Get("Content-Type"), nil
}

// ResolveMap fetches the merged variables as a key/value map (format=json).
func (c *Client) ResolveMap(ctx context.Context, project, env string) (map[string]string, error) {
	body, _, err := c.Resolve(ctx, project, env, "json")
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("opentdm: decode resolve response: %w", err)
	}
	return out, nil
}
