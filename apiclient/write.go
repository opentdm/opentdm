package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ConfigInfo is config metadata returned by the management API.
type ConfigInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Format string `json:"format"`
}

// ItemKV is a variable item (editor/write shape).
type ItemKV struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"is_secret"`
	Deleted  bool   `json:"deleted"`
}

// requestJSON performs an authenticated JSON request and unwraps the {data}
// envelope into out (out may be nil for no response body).
func (c *Client) requestJSON(ctx context.Context, method, path string, body, out any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Host+path, r)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if resp.StatusCode/100 != 2 {
		return &APIError{Status: resp.StatusCode, Body: string(data)}
	}
	if out != nil {
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(data, &env); err != nil {
			return err
		}
		return json.Unmarshal(env.Data, out)
	}
	return nil
}

// ListConfigs returns a project's configs.
func (c *Client) ListConfigs(ctx context.Context, project string) ([]ConfigInfo, error) {
	var out []ConfigInfo
	err := c.requestJSON(ctx, http.MethodGet, "/api/v1/projects/"+url.PathEscape(project)+"/configs", nil, &out)
	return out, err
}

// FindConfig resolves a config name to its metadata.
func (c *Client) FindConfig(ctx context.Context, project, name string) (ConfigInfo, error) {
	configs, err := c.ListConfigs(ctx, project)
	if err != nil {
		return ConfigInfo{}, err
	}
	for _, cfg := range configs {
		if cfg.Name == name {
			return cfg, nil
		}
	}
	return ConfigInfo{}, fmt.Errorf("config %q not found in project %q", name, project)
}

func itemsPath(project, configID, env string) string {
	return fmt.Sprintf("/api/v1/projects/%s/configs/%s/items?env=%s",
		url.PathEscape(project), url.PathEscape(configID), url.QueryEscape(env))
}

// GetItems returns the (decrypted) variable items at a (config, layer).
func (c *Client) GetItems(ctx context.Context, project, configID, env string) ([]ItemKV, error) {
	var out []ItemKV
	err := c.requestJSON(ctx, http.MethodGet, itemsPath(project, configID, env), nil, &out)
	return out, err
}

// SetItems replaces all variable items at a (config, layer).
func (c *Client) SetItems(ctx context.Context, project, configID, env string, items []ItemKV) error {
	return c.requestJSON(ctx, http.MethodPut, itemsPath(project, configID, env), map[string]any{"items": items}, nil)
}

// PutBlob uploads file content to a (config, layer).
func (c *Client) PutBlob(ctx context.Context, project, configID, env, contentType string, content []byte) error {
	endpoint := fmt.Sprintf("%s/api/v1/projects/%s/configs/%s/blob?env=%s",
		c.Host, url.PathEscape(project), url.PathEscape(configID), url.QueryEscape(env))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode/100 != 2 {
		return &APIError{Status: resp.StatusCode, Body: string(data)}
	}
	return nil
}
