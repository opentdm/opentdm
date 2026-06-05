package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/opentdm/opentdm/apiclient"
)

// cmdConfigs dispatches `opentdm configs <sub>`.
func cmdConfigs(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: opentdm configs set --env ENV CONFIG KEY=VAL [KEY=VAL...]")
		return 2
	}
	switch args[0] {
	case "set":
		return cmdConfigsSet(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown configs subcommand %q\n", args[0])
		return 2
	}
}

// cmdConfigsSet upserts variables: read-modify-write (the items endpoint
// replaces a whole layer, so we merge into the current set).
func cmdConfigsSet(args []string) int {
	fs := flag.NewFlagSet("configs set", flag.ContinueOnError)
	host := fs.String("host", "", "server base URL")
	token := fs.String("token", "", "user PAT (otdmu_...)")
	project := fs.String("project", "", "project slug")
	env := fs.String("env", "", "environment slug (required)")
	secret := fs.Bool("secret", false, "mark the values as secret")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args() // CONFIG KEY=VAL...
	if len(rest) < 2 {
		fmt.Fprintln(os.Stderr, "usage: opentdm configs set --env ENV CONFIG KEY=VAL [KEY=VAL...]")
		return 2
	}
	configName, pairs := rest[0], rest[1:]

	cfg := effective(*host, *token, *project)
	if code := requireWrite(cfg, *env); code != 0 {
		return code
	}
	ctx := context.Background()
	client := apiclient.New(cfg.Host, cfg.Token)
	info, err := client.FindConfig(ctx, cfg.Project, configName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	current, err := client.GetItems(ctx, cfg.Project, info.ID, *env)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	merged := map[string]apiclient.ItemKV{}
	for _, it := range current {
		merged[it.Key] = it
	}
	for _, p := range pairs {
		i := strings.IndexByte(p, '=')
		if i < 0 {
			fmt.Fprintf(os.Stderr, "invalid assignment %q (want KEY=VALUE)\n", p)
			return 2
		}
		key := p[:i]
		merged[key] = apiclient.ItemKV{Key: key, Value: p[i+1:], IsSecret: *secret}
	}
	out := make([]apiclient.ItemKV, 0, len(merged))
	for _, v := range merged {
		out = append(out, v)
	}
	if err := client.SetItems(ctx, cfg.Project, info.ID, *env, out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "set %d key(s) in %s/%s\n", len(pairs), configName, *env)
	return 0
}

// cmdPushFile uploads a file's content to a file config at an environment.
func cmdPushFile(args []string) int {
	fs := flag.NewFlagSet("push-file", flag.ContinueOnError)
	host := fs.String("host", "", "server base URL")
	token := fs.String("token", "", "user PAT (otdmu_...)")
	project := fs.String("project", "", "project slug")
	env := fs.String("env", "", "environment slug (required)")
	file := fs.String("file", "", "path to the file to upload (required)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	rest := fs.Args()
	if len(rest) < 1 || *file == "" {
		fmt.Fprintln(os.Stderr, "usage: opentdm push-file --env ENV --file PATH CONFIG")
		return 2
	}
	configName := rest[0]

	cfg := effective(*host, *token, *project)
	if code := requireWrite(cfg, *env); code != 0 {
		return code
	}
	content, err := os.ReadFile(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read file:", err)
		return 1
	}
	ctx := context.Background()
	client := apiclient.New(cfg.Host, cfg.Token)
	info, err := client.FindConfig(ctx, cfg.Project, configName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if info.Kind != "file" {
		fmt.Fprintf(os.Stderr, "%q is not a file config\n", configName)
		return 2
	}
	if err := client.PutBlob(ctx, cfg.Project, info.ID, *env, contentTypeFor(info.Format), content); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "pushed %s to %s/%s\n", *file, configName, *env)
	return 0
}

func contentTypeFor(format string) string {
	switch format {
	case "json":
		return "application/json"
	case "csv":
		return "text/csv"
	case "xml":
		return "application/xml"
	default:
		return "application/octet-stream"
	}
}

// requireWrite validates resolve args and that the token is a user PAT (writes
// require user auth, not a read-only service token).
func requireWrite(cfg Config, env string) int {
	if code := requireResolveArgs(cfg, env); code != 0 {
		return code
	}
	if !strings.HasPrefix(cfg.Token, "otdmu_") {
		fmt.Fprintln(os.Stderr, "writes require a user PAT (otdmu_...); create one in the UI and run 'opentdm login --token otdmu_...'")
		return 2
	}
	return 0
}
