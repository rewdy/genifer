package main

import (
	"bytes"
	"context"
	"image/png"
	"os"
	"path/filepath"

	"github.com/rewdy/genifer/internal/provider"
	"github.com/rewdy/genifer/internal/tui"
)

type fixtureProvider struct{ png []byte }

func (p fixtureProvider) Models(context.Context) ([]provider.Model, error) {
	return []provider.Model{{ID: "fixture/local", Name: "Local preview fixture", Capabilities: provider.Capabilities{AspectRatios: []string{"8:5"}}}}, nil
}
func (p fixtureProvider) Pricing(context.Context, string) (provider.Price, error) {
	return provider.Price{Unit: provider.PriceFree}, nil
}
func (p fixtureProvider) Generate(ctx context.Context, _ provider.GenerateRequest) (provider.GenerateResult, error) {
	if err := ctx.Err(); err != nil {
		return provider.GenerateResult{}, err
	}
	return provider.GenerateResult{Data: p.png, MediaType: "image/png"}, nil
}
func runApp() error {
	dir, err := os.MkdirTemp("", "genifer-preview-check-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	// Keep fixture pricing-cache reads/writes separate from the user's config.
	oldConfig, hadConfig := os.LookupEnv("XDG_CONFIG_HOME")
	if err := os.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config")); err != nil {
		return err
	}
	defer func() {
		if hadConfig {
			_ = os.Setenv("XDG_CONFIG_HOME", oldConfig)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, fixture()); err != nil {
		return err
	}
	return tui.Run(tui.Deps{Provider: fixtureProvider{png: pngData.Bytes()}, StatePath: filepath.Join(dir, "state.json"), OutputDir: dir, Version: "preview fixture"})
}
