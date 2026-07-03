// Package main is the entry point for the ccr-models-usage CLI.
//
// ccr-models-usage reads the Claude Code Router configuration, deduplicates
// provider entries by (hostname, api_key), and reports per-provider usage
// for the providers it knows how to query.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Djarvur/ccr-models-usage/internal/config"
	"github.com/Djarvur/ccr-models-usage/internal/provider"
	"github.com/Djarvur/ccr-models-usage/internal/providers/opencodego"
	"github.com/Djarvur/ccr-models-usage/internal/providers/zai"
	"github.com/Djarvur/ccr-models-usage/internal/render"
)

// exitConfigError is the exit code returned when the CCR config file
// is missing or unreadable. Adapter failures do not change the exit
// code.
const exitConfigError = 2

func main() {
	err := run(context.Background(), "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ccr-models-usage: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, configPath string) error {
	if configPath == "" {
		flagPath := flag.String("config", "", "path to the CCR config (default: ~/.claude-code-router/config.json)")
		flag.Parse()
		configPath = *flagPath
	}

	path := configPath
	if path == "" {
		defaultPath, pathErr := config.DefaultConfigPath()
		if pathErr != nil {
			return fmt.Errorf("default config path: %w", pathErr)
		}
		path = defaultPath
	}

	raw, err := config.Load(path, os.Stderr)
	if err != nil {
		if config.IsUnreadable(err) {
			os.Exit(exitConfigError)
		}

		return fmt.Errorf("load config: %w", err)
	}

	deduped := config.Dedup(raw)
	if len(deduped) == 0 {
		return nil
	}

	registry := provider.NewRegistry()
	registry.Register(zai.New())
	registry.Register(opencodego.New())

	provs := make([]provider.Provider, 0, len(deduped))
	for _, d := range deduped {
		provs = append(provs, provider.Provider{
			Name: d.Name,
			Host: d.Host,
			Key:  d.Key,
		})
	}

	results := provider.FetchAll(ctx, registry, provs)

	rows := buildRows(results, deduped)

	writeErr := render.Write(os.Stdout, rows, time.Now())
	if writeErr != nil {
		return fmt.Errorf("render: %w", writeErr)
	}

	return nil
}

// buildRows converts the parallel-fetch results into render rows,
// preserving the order of providers in the original config.
func buildRows(results []provider.Result, deduped []config.Provider) []render.Row {
	rows := make([]render.Row, 0, len(deduped))
	for idx, d := range deduped {
		header := fmt.Sprintf("%s %s", d.Name, d.Host)
		res := results[idx]
		if res.Level != "" {
			header += " " + res.Level
		}
		row := render.Row{Header: header}
		if res.Err != nil {
			row.Skip = res.Err.Error()
		} else {
			row.Limits = res.Limits
		}
		rows = append(rows, row)
	}

	return rows
}
