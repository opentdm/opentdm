// Package cli implements the opentdm command-line interface.
package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/opentdm/opentdm/apiclient"
)

const usage = `opentdm — manage project/environment config and pull it into CI and tests

Usage:
  opentdm login --host URL --token TOKEN [--project SLUG]
  opentdm pull  --env ENV [--project SLUG] [--format dotenv|json|shell|yaml|properties] [-o FILE] [--collisions]
  opentdm run   --env ENV [--project SLUG] -- <command> [args...]
  opentdm list  [--project SLUG] [--json]                                (needs a user PAT)
  opentdm configs set --env ENV [--secret] CONFIG KEY=VAL [KEY=VAL...]   (needs a user PAT)
  opentdm push-file   --env ENV --file PATH CONFIG                       (needs a user PAT)
  opentdm version

Tokens: a service token (otdm_...) is read-only (pull/run); a user PAT (otdmu_...) can also write.
Auth precedence: flags > OPENTDM_HOST/OPENTDM_TOKEN/OPENTDM_PROJECT env > ~/.opentdm/config.json
`

// Main dispatches a subcommand and returns a process exit code.
func Main(version string, args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	switch args[0] {
	case "login":
		return cmdLogin(args[1:])
	case "pull":
		return cmdPull(args[1:])
	case "run":
		return cmdRun(args[1:])
	case "list":
		return cmdList(args[1:])
	case "configs":
		return cmdConfigs(args[1:])
	case "push-file":
		return cmdPushFile(args[1:])
	case "version", "-v", "--version":
		fmt.Println("opentdm", version)
		return 0
	case "help", "-h", "--help":
		fmt.Print(usage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

func cmdLogin(args []string) int {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	host := fs.String("host", "", "server base URL")
	token := fs.String("token", "", "service token (otdm_...) or user PAT (otdmu_...)")
	project := fs.String("project", "", "default project slug")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *host == "" || *token == "" {
		fmt.Fprintln(os.Stderr, "login requires --host and --token")
		return 2
	}
	if err := saveConfig(Config{Host: *host, Token: *token, Project: *project}); err != nil {
		fmt.Fprintln(os.Stderr, "save config:", err)
		return 1
	}
	path, _ := configPath()
	fmt.Printf("Saved credentials to %s\n", path)
	return 0
}

func cmdPull(args []string) int {
	fs := flag.NewFlagSet("pull", flag.ContinueOnError)
	host := fs.String("host", "", "server base URL")
	token := fs.String("token", "", "service token")
	project := fs.String("project", "", "project slug")
	env := fs.String("env", "", "environment slug (required)")
	format := fs.String("format", "dotenv", "output format: dotenv|json|shell|yaml|properties")
	output := fs.String("o", "", "output file (default stdout)")
	showCollisions := fs.Bool("collisions", false, "list cross-config key collisions on stderr")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg := effective(*host, *token, *project)
	if code := requireResolveArgs(cfg, *env); code != 0 {
		return code
	}
	client := apiclient.New(cfg.Host, cfg.Token)
	res, err := client.Resolve(context.Background(), cfg.Project, *env, *format)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	warnCollisions(client, cfg, *env, res.Collisions, *showCollisions)
	if *output == "" || *output == "-" {
		_, _ = os.Stdout.Write(res.Body)
		return 0
	}
	if err := writeFileAtomic(*output, res.Body); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", *output)
	return 0
}

// warnCollisions reports cross-config key collisions on stderr (never stdout, so
// it can't corrupt piped/redirected output). With detail it fetches and lists
// each shadowed key via the meta=true envelope.
func warnCollisions(client *apiclient.Client, cfg Config, env string, count int, detail bool) {
	if count <= 0 {
		return
	}
	if !detail {
		fmt.Fprintf(os.Stderr, "warning: %d cross-config key collision(s); rerun with --collisions for detail\n", count)
		return
	}
	_, collisions, err := client.ResolveWithMeta(context.Background(), cfg.Project, env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %d cross-config key collision(s) (detail unavailable: %v)\n", count, err)
		return
	}
	fmt.Fprintf(os.Stderr, "warning: %d cross-config key collision(s):\n", len(collisions))
	for _, c := range collisions {
		fmt.Fprintf(os.Stderr, "  %s — kept %q, shadowed %q\n", c.Key, c.WinningConfig, c.LosingConfig)
	}
}

func cmdRun(args []string) int {
	// Split flags (before "--") from the command (after "--").
	flagArgs, cmdArgs := splitDashDash(args)
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	host := fs.String("host", "", "server base URL")
	token := fs.String("token", "", "service token")
	project := fs.String("project", "", "project slug")
	env := fs.String("env", "", "environment slug (required)")
	if err := fs.Parse(flagArgs); err != nil {
		return 2
	}
	if len(cmdArgs) == 0 {
		fmt.Fprintln(os.Stderr, "run requires a command after '--', e.g. opentdm run --env staging -- npm test")
		return 2
	}
	cfg := effective(*host, *token, *project)
	if code := requireResolveArgs(cfg, *env); code != 0 {
		return code
	}
	client := apiclient.New(cfg.Host, cfg.Token)
	vars, collisions, err := client.ResolveMap(context.Background(), cfg.Project, *env)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if collisions > 0 {
		fmt.Fprintf(os.Stderr, "warning: %d cross-config key collision(s) in %s/%s\n", collisions, cfg.Project, *env)
	}
	return runProcess(cmdArgs, mergeEnv(os.Environ(), vars))
}

func cmdList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	host := fs.String("host", "", "server base URL")
	token := fs.String("token", "", "user PAT (otdmu_...)")
	project := fs.String("project", "", "project slug")
	asJSON := fs.Bool("json", false, "output as JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg := effective(*host, *token, *project)
	if cfg.Host == "" || cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "not logged in: set --host/--token, OPENTDM_HOST/OPENTDM_TOKEN, or run 'opentdm login'")
		return 2
	}
	if cfg.Project == "" {
		fmt.Fprintln(os.Stderr, "missing project: pass --project or set a default with 'opentdm login --project'")
		return 2
	}
	client := apiclient.New(cfg.Host, cfg.Token)
	configs, err := client.ListConfigs(context.Background(), cfg.Project)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if *asJSON {
		b, err := json.MarshalIndent(configs, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(string(b))
		return 0
	}
	if len(configs) == 0 {
		fmt.Fprintf(os.Stderr, "no objects in project %q\n", cfg.Project)
		return 0
	}
	for _, c := range configs {
		fmt.Printf("%s\t%s/%s\n", c.Name, c.Kind, c.Format)
	}
	return 0
}

func requireResolveArgs(cfg Config, env string) int {
	if cfg.Host == "" || cfg.Token == "" {
		fmt.Fprintln(os.Stderr, "not logged in: set --host/--token, OPENTDM_HOST/OPENTDM_TOKEN, or run 'opentdm login'")
		return 2
	}
	if cfg.Project == "" {
		fmt.Fprintln(os.Stderr, "missing project: pass --project or set a default with 'opentdm login --project'")
		return 2
	}
	if env == "" {
		fmt.Fprintln(os.Stderr, "missing --env")
		return 2
	}
	return 0
}

// splitDashDash partitions args around the first "--".
func splitDashDash(args []string) (before, after []string) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}

// mergeEnv overlays resolved vars onto the inherited environment (resolved wins).
func mergeEnv(base []string, vars map[string]string) []string {
	out := make([]string, 0, len(base)+len(vars))
	override := map[string]bool{}
	for k := range vars {
		override[k] = true
	}
	for _, kv := range base {
		key := kv
		if i := indexByte(kv, '='); i >= 0 {
			key = kv[:i]
		}
		if !override[key] {
			out = append(out, kv)
		}
	}
	for k, v := range vars {
		out = append(out, k+"="+v)
	}
	return out
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// writeFileAtomic writes via a temp file + rename, mode 0600.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".opentdm-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
