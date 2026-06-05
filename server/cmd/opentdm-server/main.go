// Command opentdm-server is the opentdm API server. Subcommands:
//
//	serve        run the HTTP server (default)
//	gen-key      print a fresh base64 32-byte key (for OPENTDM_MASTER_KEY etc.)
//	healthcheck  GET /readyz and exit 0/1 (for container HEALTHCHECK; no shell needed)
//	version      print version and exit
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/opentdm/opentdm/server/internal/app"
	"github.com/opentdm/opentdm/server/internal/config"
	"github.com/opentdm/opentdm/server/internal/crypto"
	"github.com/opentdm/opentdm/server/internal/httpapi"
	"github.com/opentdm/opentdm/server/internal/logging"
	"github.com/opentdm/opentdm/server/internal/server"
	"github.com/opentdm/opentdm/server/internal/store"
	"github.com/opentdm/opentdm/server/internal/webui"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "serve":
		os.Exit(runServe())
	case "gen-key":
		os.Exit(runGenKey())
	case "healthcheck":
		os.Exit(runHealthcheck())
	case "version", "-v", "--version":
		fmt.Println("opentdm-server", version)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\nusage: opentdm-server [serve|gen-key|healthcheck|version]\n", cmd)
		os.Exit(2)
	}
}

func runServe() int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		return 1
	}
	if err := cfg.RequireServe(); err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		return 1
	}
	logger := logging.New(cfg.LogLevel, cfg.LogJSON)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Database + migrations.
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("db_connect", "err", err)
		return 1
	}
	defer st.Close()
	if cfg.MigrateOnStart {
		if err := st.Migrate(ctx); err != nil {
			logger.Error("migrate", "err", err)
			return 1
		}
		logger.Info("migrations_applied")
	}

	// Encryption key provider (env master key; KMS later).
	keys, err := crypto.NewEnvKeyProvider("env:v1", cfg.MasterKey, nil)
	if err != nil {
		logger.Error("key_provider", "err", err)
		return 1
	}

	// First-run setup token: if no users exist, mint a one-time token and print
	// it so the operator can create the first admin.
	setupToken := ""
	if n, err := st.Q().UserCount(ctx); err == nil && n == 0 {
		b, _ := crypto.RandomBytes(24)
		setupToken = base64.RawURLEncoding.EncodeToString(b)
		fmt.Println("──────────────────────────────────────────────────────────────")
		fmt.Println(" opentdm first-run: create the first admin with this setup token")
		fmt.Println("   OPENTDM setup token:", setupToken)
		fmt.Println("──────────────────────────────────────────────────────────────")
		logger.Warn("first_run_setup_token_printed")
	}

	svc := app.NewService(st, keys, cfg.TokenPepper, setupToken)
	secureCookies := strings.HasPrefix(cfg.Host, "https")

	// Web UI: from disk when OPENTDM_WEB_DIR is set, otherwise the embedded build.
	var web http.Handler
	if cfg.WebDir != "" {
		web = webui.DirHandler(cfg.WebDir)
		logger.Info("web_ui_from_dir", "dir", cfg.WebDir)
	} else if h, err := webui.Handler(); err != nil {
		logger.Warn("web_ui_unavailable", "err", err)
	} else {
		web = h
	}

	dbCheck := httpapi.ReadyCheck{Name: "database", Check: st.Ping}
	srv := server.New(cfg, logger, svc, secureCookies, web, dbCheck)

	if err := srv.Run(ctx); err != nil {
		logger.Error("server_error", "err", err)
		return 1
	}
	return 0
}

func runGenKey() int {
	b, err := crypto.RandomBytes(32)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen-key error:", err)
		return 1
	}
	fmt.Println(base64.StdEncoding.EncodeToString(b))
	return 0
}

func runHealthcheck() int {
	host := os.Getenv("OPENTDM_HOST")
	if host == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		host = "http://127.0.0.1:" + port
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(host + "/readyz")
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck: status", resp.StatusCode)
		return 1
	}
	return 0
}
