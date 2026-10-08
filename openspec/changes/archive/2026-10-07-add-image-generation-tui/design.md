# Design

## Context

See proposal.md — Why. Greenfield Go project; the only existing files are the OpenSpec scaffold. The design is grounded in verified OpenRouter behavior (October 2026):

- Image generation uses a dedicated endpoint `POST https://openrouter.ai/api/v1/images` with `Authorization: Bearer <key>`. Request includes `model`, `prompt`, `n`, optional `aspect_ratio`/`size`/`seed`, and `input_references[]` for image-to-image. Response is `{ data: [{ b64_json, media_type }], usage: {cost,...} }` — base64 only, no hosted URLs. Extension derives from `media_type` (png/jpeg/webp/svg+xml).
- Model discovery: `GET /api/v1/images/models` lists image-API models; each carries a `supported_parameters` map (descriptors like `{type:"enum",values:[]}`, `{type:"range",min,max}`, `{type:"boolean"}`) and reference-image support. A per-model endpoints call exists for finer detail but is not needed for MVP.
- Known failure modes: 402 insufficient credit (needs > $1), 502 generation failed (not billed), auth errors. Streaming only with `n=1`; MVP uses `n=1` and no streaming.

Constraints: terminal-only UI (no native image rendering — files are opened via the system viewer); secrets must never be stored in config in plaintext by default; the app must never overwrite the user's hand-edited config.

## Goals / Non-Goals

**Goals:**
- A narrow provider seam so a second provider is purely additive (no caller changes).
- OpenRouter-isms (exact JSON, model-id format, parameter descriptors) stay below the provider boundary.
- Lazy, generic value resolution usable by any config string.
- A responsive TUI whose generate form adapts to the selected model's capabilities.
- Clean separation of hand-edited `config.yaml` from app-written `state.json`.

**Non-Goals (design-level):**
- Chat-completions image models (PATH B) — the OpenRouter provider may add this internally later without changing the interface.
- In-terminal image preview (Sixel/Kitty/ASCII).
- Prompt templates and remembered slot values.
- Batch generation / variations (`n>1`).

## Decisions

### D1: OpenRouter images API only (PATH A)
Use `/api/v1/images` exclusively. It is the dedicated, uniform path (base64 out, consistent capability map). Chat-only image models are excluded from the MVP model list.
- Rationale: one request/response shape; capability discovery is uniform; keeps the provider implementation small.
- Alternative (PATH B, support chat image models too): rejected for MVP — two endpoints, two shapes, merged discovery. If added later it is internal to the OpenRouter provider; the `Provider` interface does not change.

### D2: Narrow provider interface
```
type Provider interface {
    Models(ctx) ([]Model, error)
    Generate(ctx, GenerateRequest) (GenerateResult, error)
}
```
`GenerateRequest` carries provider-agnostic fields: model id, prompt, `[]ReferenceImage`, and a generic parameter bag (aspect ratio, seed, …). `Model` carries id, display name, and a `Capabilities` value describing supported params + reference-image support. `GenerateResult` carries raw bytes + media type.
- Rationale: callers (workflow, TUI) depend only on this; OpenRouter mapping lives in `internal/provider/openrouter`.
- Alternative (expose OpenRouter DTOs directly): rejected — would leak provider specifics upward and make a second provider a rewrite.

### D3: Generic lazy value resolver
A resolver turns a raw string into a resolved value: literal, `{env:NAME}`, or `{cmd:...}`. Resolution is lazy (invoked when a value is needed, e.g. the API key at first request) and single-level (a resolved value is not re-scanned for directives).
- Rationale: one mechanism serves API key, open command, output dir, etc.; lazy avoids shelling out on startup.
- Alternative (dedicated `api_key` / `api_key_env` / `api_key_command` fields): rejected — less reusable, more surface. Escaping a literal `{env:...}` is deferred (noted as a known edge; add `\{` only if a real need appears).

### D4: config.yaml vs state.json split
`config.yaml` is user-owned (never written by the app). `state.json` is app-owned and holds last-used selections (last model). Both live under the platform config dir (`~/.config/genifer` on Unix).
- Rationale: the app can freely rewrite state without clobbering hand edits; a corrupt/absent state file degrades gracefully.
- Alternative (single file the app rewrites): rejected — risks losing user formatting/comments and conflating concerns.

### D5: Bubble Tea architecture — Elm-style with async commands
A root `tea.Model` composes sub-models (header, model picker, generate form, status). Generation runs as a `tea.Cmd` performing the HTTP call off the update loop, returning a completion message. A `context.Context` per generation is cancelled on ESC, aborting the request (D2's `ctx` parameter).
- Rationale: standard Bubble Tea pattern; keeps UI responsive; cancellation is first-class, not retrofitted.
- Alternative (goroutine + manual channel plumbing into Update): rejected — `tea.Cmd` is the idiomatic, less error-prone mechanism.

### D6: Capability-adaptive form
The form reads the selected `Model.Capabilities` to decide which controls to render and whether to offer reference-image attachment. Switching models re-derives the controls.
- Rationale: delivers the "gorgeous, smart" feel and prevents sending unsupported params.

### D7: Save-then-open
Always decode base64 and write a timestamped file (extension from media type) into the output dir (created if missing). Then offer to open, or auto-open when `auto_open` is set. Open command defaults per OS (`open`/`xdg-open`/`start`) and is overridable (resolved via D3).

### Package layout (proposed)
```
main.go
internal/
  config/      # load config.yaml, read/write state.json, resolver (D3,D4)
  provider/    # Provider interface + types (D2)
    openrouter/ # images API client, model mapping, error mapping (D1)
  gen/         # workflow: compose->review->generate->save->open (generation-workflow)
  tui/         # Bubble Tea models, header, picker, form, status (tui)
```

## Risks / Trade-offs

- [Base64-only responses can be large in memory] → MVP is `n=1`, single image; decode-then-write immediately, don't retain. Revisit if batches are added.
- [Model slugs/capabilities churn server-side] → Discover models at runtime (never hardcode); treat an unknown remembered model as "not available" (config spec).
- [`{cmd:...}` executes arbitrary commands from config] → Config is user-owned and local; commands run only lazily and only values the user wrote. Document the behavior; do not auto-run anything not referenced.
- [Terminal cannot render images] → Accepted; save-then-open via system viewer is the design (D7), not a limitation to work around.
- [Paid calls need > $1 credit (402) and failures return 502] → Mapped to distinct, friendly errors (image-provider spec) so the TUI never shows a raw stack trace.
- [Charm API surface may differ across versions] → Pin dependency versions in `go.mod`; keep TUI sub-models small to isolate churn.

## Open Questions

- Exact default output directory (cwd-relative `./genifer-out` vs. a fixed user dir). Does not affect specs, interface, or task breakdown — a configurable default can be finalized during implementation.
