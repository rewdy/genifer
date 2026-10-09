# Proposal

## Why

genifer supports exactly one provider at a time, and only a hosted one
(OpenRouter). People who run image models locally (ComfyUI, Forge, A1111) have
no option, and the single `provider` string cannot express "OpenRouter *and* a
local backend, pick per model." This closes both gaps: a local provider and a
configuration shape that holds several providers at once.

## What Changes

- **BREAKING** Config moves from a single `provider:` string plus a type-named
  block to a `providers:` **list** of keyed instances, each with a `key`
  (identity), a `type` (how to treat it), and type-specific fields. This lets
  two instances of the same type coexist (e.g. a laptop and a GPU-box A1111).
- **BREAKING** An existing single-provider config (`provider:` / `openrouter:`)
  no longer loads. It fails *recoverably*: genifer detects the legacy shape,
  backs the file up to `config.yaml.bak`, and routes the user into first-run
  onboarding to rebuild it. The old file is preserved, never silently clobbered.
- Add a **local image provider** that speaks the A1111-compatible HTTP contract
  (`/sdapi/v1/txt2img` for generation, `/sdapi/v1/sd-models` for discovery).
  This one contract reaches A1111, Forge, and ComfyUI-with-shim. Local models
  are always free and report hardcoded capabilities (seed yes, a fixed
  aspect-ratio-to-dimensions map, no reference images for the first cut).
- The model picker **merges models from every configured provider**, grouped by
  provider key. A model is now identified by `(providerKey, modelID)`, not a
  bare id, so two providers may offer the same model id without collision.
- A provider that is unreachable at startup (e.g. a local WebUI that is not
  running) is **isolated**: its models are omitted and it is shown as offline,
  without blanking the rest of the picker or erroring the app.
- Per-model pricing (in-memory and the `pricing-cache.json` file) is **re-keyed
  by `(providerKey, modelID)`** so prices from two providers do not overwrite
  each other.
- Last-used selection in `state.json` records the **provider key alongside the
  model id**, so pre-selection resolves to the right provider's model.
- First-run onboarding gains a **provider-type choice** (OpenRouter or local)
  before collecting type-specific detail, and writes a single list-shaped entry.
  Configuring more than one provider is done by hand-editing `config.yaml`
  (documented); a multi-provider onboarding loop is out of scope here.

## Capabilities

### New Capabilities
- `local-image-provider`: An A1111-compatible local provider — model discovery
  via `/sdapi/v1/sd-models`, generation via `/sdapi/v1/txt2img`, always-free
  pricing, hardcoded capabilities, and graceful handling of an unreachable
  endpoint.

### Modified Capabilities
- `config`: Configuration becomes a list of keyed, typed provider instances;
  the legacy single-provider shape is detected and recovered (backup + rebuild)
  rather than loaded; the starter config emits the list shape; last-used state
  records a provider key alongside the model; pricing cache is keyed per
  provider instance.
- `image-provider`: The provider abstraction is selected and dispatched by a
  per-instance `type`, and the app holds a registry of providers keyed by their
  configured `key` rather than a single active provider.
- `tui`: The model picker merges models across all configured providers, groups
  and identifies them by provider key, isolates unreachable providers, and
  routes a generation to the provider that owns the selected model; onboarding
  gains a provider-type choice and the legacy-config recovery entry point.

## Impact

- **Code**: `internal/config` (config shape, loading/legacy detection, starter,
  state, pricing cache), `internal/provider` (new `a1111` package;
  `provider.go` interface unchanged), `internal/tui` (model/picker/pricing
  state, onboarding, row rendering), `main.go` (`buildProviders` registry).
- **The provider seam (`provider.Provider`) does not change** — the app holds N
  implementations instead of one; `internal/gen` stays provider-agnostic and
  receives the resolved provider for the selected model.
- **Config format**: breaking, with a recoverable migration path (backup +
  onboarding rebuild). `docs/config.md` rewritten for the list shape.
- **App-owned files**: `state.json` and `pricing-cache.json` formats bump;
  both degrade gracefully (reset / re-fetch), so no migration is required.
- **Dependencies**: none added — the local provider is plain HTTP + base64,
  the same shape the OpenRouter client already uses.
