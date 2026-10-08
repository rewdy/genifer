// Command genifer is a full-screen terminal UI for generating images via
// OpenRouter image models.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rewdy/genifer/internal/config"
	"github.com/rewdy/genifer/internal/provider"
	"github.com/rewdy/genifer/internal/provider/openrouter"
	"github.com/rewdy/genifer/internal/tui"
)

// version is the display version shown in the header.
const version = "v0.1.0"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "genifer:", err)
		os.Exit(1)
	}
}

func run() error {
	cfgPath, err := config.ConfigPath()
	if err != nil {
		return err
	}
	cfg, err := config.Load(cfgPath)
	if err != nil && err != config.ErrConfigNotFound {
		return err
	}
	noConfig := err == config.ErrConfigNotFound

	statePath, err := config.StatePath()
	if err != nil {
		return err
	}

	p, err := buildProvider(cfg)
	if err != nil {
		return err
	}

	outputDir, err := resolveOutputDir(cfg)
	if err != nil {
		return err
	}

	openCmd, err := cfg.OpenCommand.Resolve(context.Background())
	if err != nil {
		// A bad open_command shouldn't prevent generating; fall back to default.
		openCmd = ""
	}

	deps := tui.Deps{
		Provider:    p,
		Config:      cfg,
		StatePath:   statePath,
		OutputDir:   outputDir,
		OpenCommand: openCmd,
		Version:     version,
	}
	if noConfig {
		fmt.Fprintf(os.Stderr, "No config found at %s — using defaults.\n", cfgPath)
	}
	return tui.Run(deps)
}

// buildProvider constructs the configured provider. The API key is resolved
// lazily by the provider on first request.
func buildProvider(cfg config.Config) (provider.Provider, error) {
	switch cfg.Provider {
	case "", "openrouter":
		keyFn := func(ctx context.Context) (string, error) {
			return cfg.OpenRouter.APIKey.Resolve(ctx)
		}
		opts := []openrouter.Option{}
		if cfg.OpenRouter.BaseURL != "" {
			opts = append(opts, openrouter.WithBaseURL(cfg.OpenRouter.BaseURL))
		}
		return openrouter.New(keyFn, opts...), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", cfg.Provider)
	}
}

// resolveOutputDir resolves the configured output directory. See
// config.ResolveOutputDir for the rules ("." / "pwd" / relative / absolute).
func resolveOutputDir(cfg config.Config) (string, error) {
	raw, err := cfg.OutputDir.Resolve(context.Background())
	if err != nil {
		return "", err
	}
	def, err := config.DefaultOutputDir()
	if err != nil {
		return "", err
	}
	return config.ResolveOutputDir(raw, os.Getwd, def)
}
