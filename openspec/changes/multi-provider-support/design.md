# Design

## Context

See `proposal.md` for motivation. The relevant current state:

- `provider.Provider` (`internal/provider/provider.go`) is a narrow seam —
  `Models`, `Pricing`, `Generate` — and `AGENTS.md` treats it as the single
  outside-world boundary. Only `openrouter` implements it today.
- `main.buildProvider` builds exactly one provider from `cfg.Provider` (a
  string) and the type-named `cfg.OpenRouter` block.
- The TUI (`internal/tui/model.go`) holds one `provider.Provider` in `Deps`,
  one flat `[]provider.Model`, and a `map[string]provider.Price` keyed by bare
  model id. Selection resolves to a bare model id (`currentModelID`) and
  `gen.Run` receives that single provider plus a `gen.Draft{Model: id}`.
- `state.json` (`config.State`) stores `LastModel string`; the pricing cache
  (`pricing-cache.json`) is keyed by bare model id. Both are app-owned and
  degrade gracefully.
- Config loading is strict (unknown YAML keys error) with a deliberate
  three-way outcome: missing → soft default; valid → load; malformed → hard
  error.

## Goals / Non-Goals

**Goals:**

- Keep `provider.Provider` unchanged; add providers behind the existing seam.
- Keep `internal/gen` provider-agnostic — the TUI resolves which provider owns
  the selected model and passes it in, as it does today.
- Make `(providerKey, modelID)` the model identity across picker, pricing,
  state, and generation routing, so same-id models from two providers coexist.
- Add a local A1111-compatible provider that reaches A1111 / Forge / ComfyUI
  (with the compatible shim) through one HTTP contract.

**Non-Goals:**

- Embedding an inference runtime (torch / HuggingFace model execution). Local
  means "talk to a WebUI the user already runs," never "be the engine."
- A multi-provider onboarding loop. Onboarding writes one instance; additional
  instances are hand-edited (documented).
- Growing the provider capability model (negative prompt, steps, CFG, sampler).
  Local models use WebUI defaults for the first cut; capability growth is a
  separate future change.
- img2img / ControlNet for the local provider (reference images unsupported).

## Decisions

### Config: flat-union list of keyed, typed instances

`Config.Provider string` + `OpenRouterConfig` becomes `Providers
[]ProviderConfig`, where `ProviderConfig` is a flat union:

```yaml
providers:
  - key: or
    type: openrouter
    api_key: "{env:OPENROUTER_API_KEY}"
  - key: local
    type: a1111
    base_url: "http://127.0.0.1:7860"
```

`key` is identity (used in the picker and persisted state); `type` is dispatch.
Fields not relevant to a type are simply unused by it. `base_url` is already
shared (OpenRouter uses it for test overrides; a1111 requires it).

- **Why flat union over nested-by-type or two-pass decode:** both current types
  have few fields and `AGENTS.md` says "simplicity first." A single struct is
  the least code. If types diverge sharply later, migrate to a two-pass
  `UnmarshalYAML`. Trade-off: strict decoding's typo protection weakens, since a
  field valid for one type sits unused on another — acceptable at this field
  count.
- **Validation:** keys must be unique and non-empty; `type` must be recognized
  (unknown type → clear error naming the instance, per the `image-provider`
  spec).

### Legacy config: detect, back up, recover into onboarding

Because strict decoding would already reject the legacy `provider:` /
`openrouter:` keys as unknown, loading must intercept *before* that generic
error to produce the friendly path:

1. On load, peek for legacy top-level keys (`provider:` and/or a type-named
   block) absent a `providers:` list.
2. If detected, rename `config.yaml` → `config.yaml.bak` (non-colliding name if
   `.bak` exists), and signal "needs onboarding" rather than returning parsed
   config.
3. `main` routes that signal into the first-run flow (`FirstRun`-equivalent).

- **Why back-up-then-write, not overwrite:** the user-owned / never-overwrite
  invariant is about not *destroying* their file. Relocating to `.bak` and
  writing fresh honors it; a true clobber would not. `WriteStarter` keeps its
  O_EXCL create-only guarantee for the normal path; the legacy path is the one
  sanctioned replacement, and only after the backup exists.
- **No key prefill** (explored and chosen): the old key stays in `.bak`; the
  rebuild re-collects it. Prefill would mean parsing the very shape we reject —
  more code for little gain.

### Provider registry replaces the singleton

`main.buildProvider` → `buildProviders(cfg) (map[string]provider.Provider,
error)`, one entry per instance, dispatched on `.Type`. `tui.Deps.Provider
provider.Provider` → a registry (`map[string]provider.Provider`, plus the
ordered instance list for stable picker grouping). `provider.go` is untouched —
the app simply holds N implementations.

### Model identity becomes (providerKey, modelID) in the TUI

The ripple the exploration identified as the real heart of the work:

- Startup fans out `Models(ctx)` across instances concurrently, tags each
  returned model with its provider key, and merges into one list. A failing
  instance contributes zero models and a recorded "offline" marker (failure
  isolation) instead of aborting the merge.
- `m.prices` is re-keyed by `providerKey + "/" + modelID`; the on-disk pricing
  cache gets the same composite key and a version bump (stale/old-version →
  ignored and re-fetched, so no migration).
- `currentModelID` → carries the provider key; the picker row (`delegate.go`)
  shows a provider-key group/tag.
- `generate()` resolves the owning provider from the registry by key and passes
  that single `provider.Provider` into `gen.Run` — `gen.Draft` and `gen.Run`
  keep their current provider-agnostic shape.
- `state.State` gains `LastProviderKey`; `preselectModel` matches on
  `(key, id)`.

- **Why key `gen` stays agnostic:** preserves the one-way dependency
  (`tui → gen, provider`). `gen` never learns about multiple providers; it
  receives whichever one owns the selected model, exactly as today.

### Local provider = A1111-compatible HTTP client

New `internal/provider/a1111` package implementing the three methods:

- `Models` → `GET /sdapi/v1/sd-models` (real catalog of installed checkpoints).
- `Generate` → `POST /sdapi/v1/txt2img`, mapping the selected aspect ratio to
  `width`/`height` via a fixed table; decode the base64 image from the
  `images` array. Whatever checkpoint is currently loaded is what generates
  (no checkpoint-switch call in the first cut).
- `Pricing` → always `PriceFree`.
- `Capabilities` → hardcoded: `SupportsSeed: true`, a fixed `AspectRatios`
  list with a key→dimensions map, `AcceptsReferenceImages: false`.
- Unreachable endpoint → a distinct error (reuse/extend the sentinel errors in
  `provider.go`, e.g. a connection-refused mapping) so the TUI can mark it
  offline, kept separate from `ErrAuth`.

- **Why A1111-compatible over ComfyUI-native or embedding:** `/sdapi/v1/*` is a
  de-facto standard that Forge speaks natively and ComfyUI speaks via a
  community shim, and it matches genifer's existing HTTP+base64 shape, so it
  reaches the most runtimes for the least code. ComfyUI's native graph API would
  fight the flat `GenerateRequest`; embedding torch is a different product.

### Provider key naming

The config `type` for the local provider is `a1111` — it names the *wire
contract*, which Forge and the ComfyUI shim also honor, rather than a specific
app the user runs. Instance `key` is user-chosen (e.g. `local`, `gpu-box`).

## Risks / Trade-offs

- **Flat-union config weakens strict typo detection** → acceptable at the
  current small field count; revisit with a typed decode if types diverge.
- **ComfyUI `/sdapi/v1/*` compatibility is community-maintained**, not
  first-party → scope the spec to the A1111 contract; ComfyUI "works if you
  install the shim," not a guarantee we test against ComfyUI directly.
- **Startup fan-out adds latency equal to the slowest reachable provider** →
  run `Models` calls concurrently with a bounded timeout; an unreachable local
  endpoint must fail fast (short dial timeout) so it never stalls the picker.
- **Breaking config change affects existing users** → mitigated by recoverable
  detection (backup + onboarding rebuild) and a `docs/config.md` rewrite; the
  old file is preserved in `.bak`.
- **App-owned file format bumps** (`state.json` key, `pricing-cache.json` key +
  version) → both already degrade gracefully; worst case is one launch without
  a pre-selected model and a one-time pricing re-fetch.

## Migration Plan

1. Ship the new config shape, registry, local provider, and TUI changes
   together (one change).
2. On first launch after upgrade: a legacy `config.yaml` is backed up to `.bak`
   and the user is walked through onboarding to rebuild it; `state.json` and
   `pricing-cache.json` are silently superseded (reset / re-fetch).
3. Rollback: a user reverting to the prior genifer version still has their
   `config.yaml.bak` to restore by hand. No destructive, irreversible step.
