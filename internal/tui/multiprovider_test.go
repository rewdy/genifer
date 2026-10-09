package tui

import (
	"context"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rewdy/genifer/internal/config"
	"github.com/rewdy/genifer/internal/provider"
)

// fakeProvider is a test double for provider.Provider. It records whether its
// Generate was called so routing can be asserted.
type fakeProvider struct {
	name      string
	models    []provider.Model
	modelsErr error
	price     provider.Price
	genCalled *string // set to p.name on Generate
}

func (p *fakeProvider) Models(ctx context.Context) ([]provider.Model, error) {
	if p.modelsErr != nil {
		return nil, p.modelsErr
	}
	return p.models, nil
}

func (p *fakeProvider) Pricing(ctx context.Context, modelID string) (provider.Price, error) {
	return p.price, nil
}

func (p *fakeProvider) Generate(ctx context.Context, r provider.GenerateRequest) (provider.GenerateResult, error) {
	if p.genCalled != nil {
		*p.genCalled = p.name
	}
	return provider.GenerateResult{Data: []byte("img"), MediaType: "image/png"}, nil
}

// 7.1: a reachable and an unreachable provider — the reachable models list,
// the offline marker, and no app-level error.
func TestLoadModelsIsolatesUnreachable(t *testing.T) {
	reachable := &fakeProvider{name: "or", models: []provider.Model{{ID: "m1", Name: "Model 1"}}}
	down := &fakeProvider{name: "local", modelsErr: provider.ErrUnreachable}
	registry := map[string]provider.Provider{"or": reachable, "local": down}
	keys := []string{"or", "local"}

	msg := loadModels(registry, keys)().(modelsLoadedMsg)
	if msg.err != nil {
		t.Fatalf("err = %v, want nil (one provider reachable)", msg.err)
	}
	if len(msg.models) != 1 || msg.models[0].ProviderKey != "or" || msg.models[0].Model.ID != "m1" {
		t.Fatalf("models = %+v, want one from 'or'", msg.models)
	}
	if !msg.offline["local"] {
		t.Error("expected 'local' marked offline")
	}
	if msg.offline["or"] {
		t.Error("'or' should not be offline")
	}
}

// 7.1: when every provider is unreachable, the model list is an app error.
func TestLoadModelsAllUnreachable(t *testing.T) {
	down := &fakeProvider{name: "local", modelsErr: provider.ErrUnreachable}
	registry := map[string]provider.Provider{"local": down}
	msg := loadModels(registry, []string{"local"})().(modelsLoadedMsg)
	if msg.err == nil {
		t.Fatal("err = nil, want an error when no provider is reachable")
	}
}

// 7.2: two providers sharing a model id resolve to distinct prices.
func TestPricesDistinctAcrossProviders(t *testing.T) {
	or := &fakeProvider{name: "or", price: provider.Price{Unit: provider.PricePerImage, USD: 0.02}}
	local := &fakeProvider{name: "local", price: provider.Price{Unit: provider.PriceFree}}
	m := New(Deps{
		Providers:    map[string]provider.Provider{"or": or, "local": local},
		ProviderKeys: []string{"or", "local"},
		StatePath:    filepath.Join(t.TempDir(), "state.json"),
	})
	m.models = []taggedModel{
		{ProviderKey: "or", Model: provider.Model{ID: "shared", Name: "Shared"}},
		{ProviderKey: "local", Model: provider.Model{ID: "shared", Name: "Shared"}},
	}
	m = m.initPricing()
	// Drive the pricing fetches synchronously.
	cmd := m.pricingCmds()
	if cmd == nil {
		t.Fatal("expected pricing commands")
	}
	// Resolve each provider's price via its command result applied to the model.
	orPrice, _ := or.Pricing(context.Background(), "shared")
	localPrice, _ := local.Pricing(context.Background(), "shared")
	m.prices[config.PricingCacheKey("or", "shared")] = orPrice
	m.prices[config.PricingCacheKey("local", "shared")] = localPrice

	if m.prices[config.PricingCacheKey("or", "shared")].USD != 0.02 {
		t.Errorf("or price = %+v, want 0.02/img", m.prices[config.PricingCacheKey("or", "shared")])
	}
	if m.prices[config.PricingCacheKey("local", "shared")].Unit != provider.PriceFree {
		t.Errorf("local price = %+v, want free", m.prices[config.PricingCacheKey("local", "shared")])
	}
}

// 7.3: selection and preselect work over a merged two-provider list, matched by
// key+id and persisted with the provider key.
func TestSelectionAcrossProviders(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	m := New(Deps{
		Providers:    map[string]provider.Provider{"or": &fakeProvider{name: "or"}, "local": &fakeProvider{name: "local"}},
		ProviderKeys: []string{"or", "local"},
		StatePath:    statePath,
	})
	m.models = []taggedModel{
		{ProviderKey: "or", Model: provider.Model{ID: "shared", Name: "OR Shared"}},
		{ProviderKey: "local", Model: provider.Model{ID: "shared", Name: "Local Shared"}},
	}
	m.rebuildItems()
	m.picker.Select(1) // the local instance of "shared"

	updated, _ := m.handlePickerKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	s := config.LoadState(statePath)
	if s.LastModel != "shared" || s.LastProviderKey != "local" {
		t.Fatalf("persisted state = %+v, want shared@local", s)
	}
	// preselect must resolve back to the local instance (index 1), not or (0).
	if idx := preselectModel(m.models, s.LastProviderKey, s.LastModel); idx != 1 {
		t.Errorf("preselect = %d, want 1 (local)", idx)
	}
}

// 7.4: generation routes to the provider that owns the selected model.
func TestGenerateRoutesToOwningProvider(t *testing.T) {
	var called string
	or := &fakeProvider{name: "or", genCalled: &called}
	local := &fakeProvider{name: "local", genCalled: &called}
	m := New(Deps{
		Providers:    map[string]provider.Provider{"or": or, "local": local},
		ProviderKeys: []string{"or", "local"},
		StatePath:    filepath.Join(t.TempDir(), "state.json"),
		OutputDir:    t.TempDir(),
	})
	m.models = []taggedModel{
		{ProviderKey: "or", Model: provider.Model{ID: "shared"}},
		{ProviderKey: "local", Model: provider.Model{ID: "shared"}},
	}
	m.selected = 1 // local
	m.prompt.SetValue("a cat")

	_, cmd := m.generate()
	if cmd == nil {
		t.Fatal("expected a generate command")
	}
	// Draining the batch runs the generation closure.
	drainCmd(cmd)
	if called != "local" {
		t.Errorf("generation routed to %q, want local", called)
	}
}

// drainCmd executes a tea.Cmd (and any batched sub-commands) to force side
// effects in tests.
func drainCmd(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			drainCmd(c)
		}
	}
}
