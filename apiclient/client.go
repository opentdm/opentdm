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
	"strconv"
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

// Collision reports that a key was defined by more than one config; the
// higher-precedence config (WinningConfig) supplied the resolved value.
type Collision struct {
	Key           string `json:"key"`
	WinningConfig string `json:"winning_config"`
	LosingConfig  string `json:"losing_config"`
}

// ResolveResult is a rendered resolve response plus the cross-config collision
// count (from the X-OpenTDM-Collisions header).
type ResolveResult struct {
	Body        []byte
	ContentType string
	Collisions  int
}

// projectResolvePath is the whole-project (cross-config merge) resolve path.
func projectResolvePath(project string) string {
	return fmt.Sprintf("/api/v1/projects/%s/resolve", url.PathEscape(project))
}

// configResolvePath is the per-file (single config) resolve path.
func configResolvePath(project, config string) string {
	return fmt.Sprintf("/api/v1/projects/%s/configs/%s/resolve", url.PathEscape(project), url.PathEscape(config))
}

// resolveGET issues an authenticated GET to a resolve endpoint path and returns
// the raw body, headers, and status.
func (c *Client) resolveGET(ctx context.Context, path string, q url.Values) ([]byte, http.Header, int, error) {
	endpoint := fmt.Sprintf("%s%s?%s", c.Host, path, q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, nil, 0, err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	return body, resp.Header, resp.StatusCode, nil
}

// Resolve fetches the merged variables for a project+environment in the given
// format ("dotenv", "json", "shell", "yaml", "properties"; "" => json). The
// result carries the raw body, its content type, and the collision count.
func (c *Client) Resolve(ctx context.Context, project, env, format string) (ResolveResult, error) {
	q := url.Values{}
	q.Set("env", env)
	if format != "" {
		q.Set("format", format)
	}
	body, header, status, err := c.resolveGET(ctx, projectResolvePath(project), q)
	if err != nil {
		return ResolveResult{}, err
	}
	if status != http.StatusOK {
		return ResolveResult{}, &APIError{Status: status, Body: string(body)}
	}
	collisions, _ := strconv.Atoi(header.Get("X-OpenTDM-Collisions"))
	return ResolveResult{Body: body, ContentType: header.Get("Content-Type"), Collisions: collisions}, nil
}

// ResolveMap fetches the merged variables as a key/value map (format=json),
// also returning the collision count.
func (c *Client) ResolveMap(ctx context.Context, project, env string) (map[string]string, int, error) {
	res, err := c.Resolve(ctx, project, env, "json")
	if err != nil {
		return nil, 0, err
	}
	out := map[string]string{}
	if err := json.Unmarshal(res.Body, &out); err != nil {
		return nil, 0, fmt.Errorf("opentdm: decode resolve response: %w", err)
	}
	return out, res.Collisions, nil
}

// ResolveWithMeta fetches the merged variables (as a map) together with the full
// cross-config collision detail, using the server's meta=true JSON envelope.
func (c *Client) ResolveWithMeta(ctx context.Context, project, env string) (map[string]string, []Collision, error) {
	q := url.Values{}
	q.Set("env", env)
	q.Set("meta", "true")
	body, _, status, err := c.resolveGET(ctx, projectResolvePath(project), q)
	if err != nil {
		return nil, nil, err
	}
	if status != http.StatusOK {
		return nil, nil, &APIError{Status: status, Body: string(body)}
	}
	var envelope struct {
		Data map[string]string `json:"data"`
		Meta struct {
			Collisions []Collision `json:"collisions"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, nil, fmt.Errorf("opentdm: decode resolve response: %w", err)
	}
	return envelope.Data, envelope.Meta.Collisions, nil
}

// ResolveConfig fetches a SINGLE config's resolved variables (base → env
// override, tombstones) for a project+environment, rendered in the given format
// ("dotenv", "json", "shell", "yaml", "properties"; "" => json). This is the
// per-file retrieval path: no cross-config merge, so no collisions are reported.
func (c *Client) ResolveConfig(ctx context.Context, project, config, env, format string) (ResolveResult, error) {
	q := url.Values{}
	q.Set("env", env)
	if format != "" {
		q.Set("format", format)
	}
	body, header, status, err := c.resolveGET(ctx, configResolvePath(project, config), q)
	if err != nil {
		return ResolveResult{}, err
	}
	if status != http.StatusOK {
		return ResolveResult{}, &APIError{Status: status, Body: string(body)}
	}
	return ResolveResult{Body: body, ContentType: header.Get("Content-Type")}, nil
}

// ResolveConfigMap fetches a single config's resolved variables as a key/value
// map (format=json).
func (c *Client) ResolveConfigMap(ctx context.Context, project, config, env string) (map[string]string, error) {
	res, err := c.ResolveConfig(ctx, project, config, env, "json")
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	if err := json.Unmarshal(res.Body, &out); err != nil {
		return nil, fmt.Errorf("opentdm: decode resolve response: %w", err)
	}
	return out, nil
}
