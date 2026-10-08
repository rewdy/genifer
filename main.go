// Command genifer is a full-screen terminal UI for generating images via
// OpenRouter image models.
package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/rewdy/genifer/internal/config"
	"github.com/rewdy/genifer/internal/provider"
	"github.com/rewdy/genifer/internal/provider/openrouter"
	"github.com/rewdy/genifer/internal/tui"
)

// version is the display version. It defaults to "dev" and may be overridden at
// build time via -ldflags "-X main.version=...". When unset, buildVersion
// derives it from the embedded module/VCS build info.
var version = "dev"

// buildVersion resolves the version to display. A build-time override (via
// -ldflags) wins; otherwise it falls back to the Go-embedded build info: the
// module version for `go install module@tag` builds, or the VCS revision for
// plain `go build`.
func buildVersion() string {
	if version != "dev" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	var rev, dirty string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	if rev != "" {
		if len(rev) > 12 {
			rev = rev[:12]
		}
		return rev + dirty
	}
	return version
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "genifer:", err)
		os.Exit(1)
	}
}

func run() error {
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--version", "-v", "version":
			fmt.Println(buildVersion())
			return nil
		}
	}

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
		Version:     buildVersion(),
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
