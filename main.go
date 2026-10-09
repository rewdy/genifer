// Command genifer is a full-screen terminal UI for generating images via
// OpenRouter image models.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"

	"github.com/rewdy/genifer/internal/config"
	"github.com/rewdy/genifer/internal/provider"
	"github.com/rewdy/genifer/internal/provider/a1111"
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
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "--version", "-v", "version":
			fmt.Println(buildVersion())
			return nil
		case "config":
			return configCommand(args[1:])
		}
	}

	cfgPath, err := config.ConfigPath()
	if err != nil {
		return err
	}
	cfg, outcome, err := loadConfig(cfgPath)
	if err != nil {
		return err
	}
	firstRun := outcome != loadValid

	statePath, err := config.StatePath()
	if err != nil {
		return err
	}

	registry, providerKeys, err := buildProviders(cfg)
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
		Providers:    registry,
		ProviderKeys: providerKeys,
		Config:       cfg,
		StatePath:    statePath,
		OutputDir:    outputDir,
		OpenCommand:  openCmd,
		Version:      buildVersion(),
		FirstRun:     firstRun,
		ConfigPath:   cfgPath,
		Paster:       tui.NewClipboardPaster(),
	}
	return tui.Run(deps)
}

// loadOutcome classifies how config loading resolved, so run can decide between
// a normal start and the first-run onboarding flow.
type loadOutcome int

const (
	loadValid   loadOutcome = iota // a valid current-shape config loaded
	loadMissing                    // no config.yaml present
	loadLegacy                     // a legacy-shape config was detected and backed up
)

// loadConfig loads the config, resolving the three non-fatal outcomes:
//   - missing   → built-in defaults, enter first-run onboarding
//   - legacy    → back the file up, enter first-run onboarding on defaults
//   - valid     → the loaded config
//
// A malformed file remains a hard error.
func loadConfig(cfgPath string) (config.Config, loadOutcome, error) {
	cfg, err := config.Load(cfgPath)
	switch {
	case err == nil:
		return cfg, loadValid, nil
	case errors.Is(err, config.ErrConfigNotFound):
		return config.Default(), loadMissing, nil
	case errors.Is(err, config.ErrLegacyConfig):
		if _, bErr := config.BackupConfig(cfgPath); bErr != nil {
			return config.Config{}, loadLegacy, bErr
		}
		return config.Default(), loadLegacy, nil
	default:
		return config.Config{}, loadValid, err
	}
}

// configCommand implements `genifer config` and `genifer config path`.
//
//   - `genifer config path` prints the resolved config path and exits, with no
//     side effects (no file creation, no editor, no TUI).
//   - `genifer config` creates a commented starter config.yaml if none exists,
//     then opens it in the resolved editor ($VISUAL → $EDITOR → OS default),
//     returning before the TUI is launched.
func configCommand(args []string) error {
	cfgPath, err := config.ConfigPath()
	if err != nil {
		return err
	}

	if len(args) > 0 && args[0] == "path" {
		fmt.Println(cfgPath)
		return nil
	}

	if _, err := config.WriteStarter(cfgPath, config.StarterSpec{}); err != nil {
		return err
	}

	argv, err := config.ResolveEditor(os.Getenv, runtime.GOOS)
	if err != nil {
		return fmt.Errorf("%w\nedit it directly at: %s", err, cfgPath)
	}

	argv = append(argv, cfgPath)
	c := exec.Command(argv[0], argv[1:]...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("opening %s with %q: %w", cfgPath, argv[0], err)
	}
	return nil
}

// buildProviders constructs the configured provider registry: a map keyed by
// each instance's configured key, plus the ordered list of keys for stable
// picker grouping. Each instance is dispatched on its Type; an unrecognized
// type is a clear error naming the offending instance. API keys are resolved
// lazily by the provider on first request.
func buildProviders(cfg config.Config) (map[string]provider.Provider, []string, error) {
	reg := make(map[string]provider.Provider, len(cfg.Providers))
	keys := make([]string, 0, len(cfg.Providers))
	for _, pc := range cfg.Providers {
		pc := pc // capture per iteration for the key closure below
		var p provider.Provider
		switch pc.Type {
		case config.TypeOpenRouter:
			opts := []openrouter.Option{}
			if pc.BaseURL != "" {
				opts = append(opts, openrouter.WithBaseURL(pc.BaseURL))
			}
			keyFn := func(ctx context.Context) (string, error) {
				return pc.APIKey.Resolve(ctx)
			}
			p = openrouter.New(keyFn, opts...)
		case config.TypeA1111:
			p = a1111.New(pc.BaseURL)
		default:
			return nil, nil, fmt.Errorf("provider %q: unknown type %q", pc.Key, pc.Type)
		}
		reg[pc.Key] = p
		keys = append(keys, pc.Key)
	}
	return reg, keys, nil
}

// resolveOutputDir resolves the configured output directory. See
// config.ResolveOutputDir for the rules ("." / "pwd" / "~" / relative / absolute).
func resolveOutputDir(cfg config.Config) (string, error) {
	raw, err := cfg.OutputDir.Resolve(context.Background())
	if err != nil {
		return "", err
	}
	def, err := config.DefaultOutputDir()
	if err != nil {
		return "", err
	}
	return config.ResolveOutputDir(raw, os.Getwd, os.UserHomeDir, def)
}
