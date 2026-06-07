// Package config loads and validates opentdm server configuration from the
// environment. All variables are prefixed OPENTDM_ except the standard
// DATABASE_URL and PORT. Validation happens once, at boot, fail-fast.
package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// maxRateLimit bounds the auth rate-limit knobs — a sanity ceiling that also
// guarantees the int64→int narrowing in Load can't overflow on 32-bit builds.
const maxRateLimit = 1_000_000

// Config is the fully-parsed, validated server configuration.
type Config struct {
	Bind     string // network interface to bind, e.g. "0.0.0.0"
	Port     string // TCP port, e.g. "8080"
	Host     string // externally-visible base URL, e.g. "http://localhost:8080"
	LogLevel string // debug|info|warn|error
	LogJSON  bool   // true => JSON logs, false => text

	DatabaseURL string // postgres connection string

	// Secret material (decoded). Empty unless the corresponding env var is set.
	MasterKey     []byte // 32 bytes, from OPENTDM_MASTER_KEY (base64)
	TokenPepper   []byte // from OPENTDM_TOKEN_PEPPER (base64)
	SessionSecret []byte // from OPENTDM_SESSION_SECRET (base64)

	MigrateOnStart bool
	MaxBlobBytes   int64
	WebDir         string // optional: serve UI from disk instead of embed

	// Per-IP rate limiting for the unauthenticated auth endpoints (login,
	// bootstrap, invitation accept). RPM <= 0 disables it.
	AuthRateLimitRPM   int
	AuthRateLimitBurst int

	// How often buffered token/PAT last-used timestamps are flushed to the DB.
	TokenTouchInterval time.Duration

	// SMTP for invitation emails (optional; when unset, invite links are logged).
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	SMTPTLS      string // starttls (default) | implicit | none
}

// Load reads configuration from the environment, applying defaults. It returns
// an error only when a present value is malformed (bad base64, non-numeric).
// It does NOT enforce that required-for-serving values are present — call
// RequireServe for that, so subcommands like gen-key can run without secrets.
func Load() (*Config, error) {
	c := &Config{
		Bind:           envOr("OPENTDM_BIND", "0.0.0.0"),
		Port:           envOr("PORT", "8080"),
		Host:           envOr("OPENTDM_HOST", "http://localhost:8080"),
		LogLevel:       strings.ToLower(envOr("OPENTDM_LOG_LEVEL", "info")),
		LogJSON:        strings.ToLower(envOr("OPENTDM_LOG_FORMAT", "json")) != "text",
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		MigrateOnStart: envBool("OPENTDM_MIGRATE_ON_START", true),
		WebDir:         os.Getenv("OPENTDM_WEB_DIR"),
		SMTPHost:       os.Getenv("OPENTDM_SMTP_HOST"),
		SMTPUsername:   os.Getenv("OPENTDM_SMTP_USERNAME"),
		SMTPPassword:   os.Getenv("OPENTDM_SMTP_PASSWORD"),
		SMTPFrom:       os.Getenv("OPENTDM_SMTP_FROM"),
		SMTPTLS:        strings.ToLower(os.Getenv("OPENTDM_SMTP_TLS")),
	}
	smtpPort, err := strconv.Atoi(envOr("OPENTDM_SMTP_PORT", "0"))
	if err != nil {
		return nil, fmt.Errorf("OPENTDM_SMTP_PORT: %w", err)
	}
	c.SMTPPort = smtpPort

	maxBlob, err := envInt64("OPENTDM_MAX_BLOB_BYTES", 10*1024*1024)
	if err != nil {
		return nil, err
	}
	c.MaxBlobBytes = maxBlob

	rlRPM, err := envInt64("OPENTDM_AUTH_RATELIMIT_RPM", 10)
	if err != nil {
		return nil, err
	}
	if rlRPM < 0 || rlRPM > maxRateLimit {
		return nil, fmt.Errorf("OPENTDM_AUTH_RATELIMIT_RPM: must be between 0 and %d", maxRateLimit)
	}
	c.AuthRateLimitRPM = int(rlRPM)
	rlBurst, err := envInt64("OPENTDM_AUTH_RATELIMIT_BURST", 5)
	if err != nil {
		return nil, err
	}
	if rlBurst < 0 || rlBurst > maxRateLimit {
		return nil, fmt.Errorf("OPENTDM_AUTH_RATELIMIT_BURST: must be between 0 and %d", maxRateLimit)
	}
	c.AuthRateLimitBurst = int(rlBurst)

	touchSecs, err := envInt64("OPENTDM_TOKEN_TOUCH_INTERVAL", 30)
	if err != nil {
		return nil, err
	}
	if touchSecs <= 0 {
		touchSecs = 30
	}
	c.TokenTouchInterval = time.Duration(touchSecs) * time.Second

	if c.MasterKey, err = decodeKey("OPENTDM_MASTER_KEY", 32, true); err != nil {
		return nil, err
	}
	if c.TokenPepper, err = decodeKey("OPENTDM_TOKEN_PEPPER", 0, false); err != nil {
		return nil, err
	}
	if c.SessionSecret, err = decodeKey("OPENTDM_SESSION_SECRET", 0, false); err != nil {
		return nil, err
	}
	return c, nil
}

// RequireServe verifies all values required to actually serve traffic are set.
func (c *Config) RequireServe() error {
	var missing []string
	if len(c.MasterKey) == 0 {
		missing = append(missing, "OPENTDM_MASTER_KEY")
	}
	if len(c.TokenPepper) == 0 {
		missing = append(missing, "OPENTDM_TOKEN_PEPPER")
	}
	if len(c.SessionSecret) == 0 {
		missing = append(missing, "OPENTDM_SESSION_SECRET")
	}
	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}
	return nil
}

// Addr is the host:port the server listens on.
func (c *Config) Addr() string { return c.Bind + ":" + c.Port }

// decodeKey base64-decodes an env var. If exactLen > 0 the decoded length must
// match exactly. If the var is unset it returns nil (RequireServe enforces
// presence later); if exactLen>0 and unset and required, it is still nil here.
func decodeKey(name string, exactLen int, _ bool) ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return nil, nil
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		// Tolerate URL-safe / unpadded encodings as a convenience.
		if k2, e2 := base64.RawStdEncoding.DecodeString(raw); e2 == nil {
			key = k2
		} else {
			return nil, fmt.Errorf("%s: not valid base64: %w", name, err)
		}
	}
	if exactLen > 0 && len(key) != exactLen {
		return nil, fmt.Errorf("%s: decoded length is %d bytes, want %d", name, len(key), exactLen)
	}
	return key, nil
}

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func envBool(name string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	switch v {
	case "":
		return def
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func envInt64(name string, def int64) (int64, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: not an integer: %w", name, err)
	}
	return n, nil
}
