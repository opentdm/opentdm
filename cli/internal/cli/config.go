package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the persisted CLI context.
type Config struct {
	Host    string `json:"host"`
	Token   string `json:"token"`
	Project string `json:"project,omitempty"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".opentdm", "config.json"), nil
}

// loadConfig reads the config file (ignoring a missing file).
func loadConfig() Config {
	var c Config
	path, err := configPath()
	if err != nil {
		return c
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return c
	}
	_ = json.Unmarshal(data, &c)
	return c
}

// saveConfig writes the config file with restrictive permissions.
func saveConfig(c Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// effective merges file config with environment and explicit flag overrides
// (flag > env > file).
func effective(flagHost, flagToken, flagProject string) Config {
	c := loadConfig()
	if v := os.Getenv("OPENTDM_HOST"); v != "" {
		c.Host = v
	}
	if v := os.Getenv("OPENTDM_TOKEN"); v != "" {
		c.Token = v
	}
	if v := os.Getenv("OPENTDM_PROJECT"); v != "" {
		c.Project = v
	}
	if flagHost != "" {
		c.Host = flagHost
	}
	if flagToken != "" {
		c.Token = flagToken
	}
	if flagProject != "" {
		c.Project = flagProject
	}
	return c
}
