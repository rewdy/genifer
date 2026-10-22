# Design

## Context

See proposal.md — Why. Relevant current state and constraints:

- `gen.Save(dir, res)` (`internal/gen/save.go`) is the only place a filename is
  minted; it is timestamp-only and never sees the prompt. `gen.Run`
  (`internal/gen/run.go`) is the single chokepoint where both the `Draft`
  (prompt) and the `provider.Provider` are in scope — the natural place to
  insert a naming step before `Save`.
- `provider.Provider` is the one seam to the outside world; callers must not see
  provider-specific shapes (AGENTS.md). It is image-only today (`Models`,
  `Pricing`, `Generate`).
- The OpenRouter client already has reusable pieces: `authHeader`, the shared
  `*http.Client`, `defaultBaseURL`, and the `mapTransportError` / `mapHTTPError`
  / `extractMessage` error mappers.
- The in-tool editor is `internal/tui/config.go` backed by
  `config.WriteSettings` / `EditableSettings` in `internal/config/writer.go`,
  which can only edit **top-level** scalar YAML keys (`setScalarKey` skips
  indented lines).
- Project ethos: "pricing/cost is decorative, never fatal." Smart naming follows
  the same rule — best-effort, never blocks a generation.

## Goals / Non-Goals

**Goals:**
- Add the naming step at the `gen.Run` seam with a guaranteed timestamp
  fallback, touching `Save` minimally.
- Keep the provider abstraction intact: a new interface method, not a direct
  OpenRouter call from `gen`.
- Persist both settings as top-level `config.yaml` keys so the existing writer
  manages them with no nested-key changes.
- Expose both settings in the config editor, with an OpenRouter models-page link.

**Non-Goals:**
- No model picker / live model list for the filename model (free-text id only,
  per the request).
- No prompt-engineering sophistication beyond a single short system+user prompt;
  no multi-candidate ranking.
- No caching of names, no renaming of previously saved files.
- No new config domain levels or restructuring of existing specs.

## Decisions

### 1. Insert naming in `gen.Run`, give `Save` an explicit base name

Add an unexported helper and change `Save` to accept a base name, keeping the
timestamp as the fallback value.

- Introduce `gen.Save(dir, res, base string)` where `base == ""` means "use the
  timestamp" — preserving current behavior for all existing callers that pass
  `""`. The collision loop and extension logic are unchanged.
- In `Run`, after a successful `Generate` and before `Save`, compute the base
  name via a new `nameFor(ctx, p, d, cfg)` helper. On any error or empty result
  it returns `""`, so `Save` falls back to the timestamp.
- `Run` needs the smart-filename settings. Thread a small value (e.g. a
  `NameOptions{Enabled bool; Model string}`) into `Run` rather than the whole
  `config.Config`, keeping `gen` free of config-package coupling. The TUI builds
  it from `m.deps.Config`.

*Alternative considered:* compute the name in the TUI and pass it into `Run`.
Rejected — `Run` is already the designated "generation done → saved" seam and
has `p` and `d.Prompt` in hand; threading a name through `generatedMsg` adds a
state round-trip for no benefit.

### 2. New provider method `GenerateText`

Add to the `Provider` interface:

```go
GenerateText(ctx context.Context, req TextRequest) (string, error)
```

with `TextRequest{ Model, Prompt string }`. The OpenRouter implementation POSTs
to `baseURL + "/chat/completions"` with a `messages` array (a system instruction
plus the generation prompt), reusing `authHeader`, `c.http`, and the existing
error mappers. Response is `choices[0].message.content`, trimmed.

The naming prompt (built in the `gen` layer, since the wording is
feature-specific) asks for a dash-separated slug explicitly, for example:

> Generate a short, descriptive filename for an image created from the following
> prompt. Use 3–6 lowercase words separated by single dashes, with no file
> extension, no spaces, and no punctuation. Respond with only the filename.
> Prompt: {prompt}

The prompt wording is only a hint. `sanitizeBaseName` (Decision 5) is the
contract that guarantees the dash slug regardless of what the model returns, so
a non-compliant response (`"A Red Fox!"`) still becomes `a-red-fox`.

*Alternative considered:* a free function in the `openrouter` package called
directly from `gen`. Rejected — bypasses the one-seam rule; every outside call
goes through `Provider`.

*Alternative considered:* a narrowly named `SuggestFilename`. Rejected — a
generic `GenerateText` is a smaller, more reusable surface and keeps
filename-specific prompt wording in the `gen` layer where the feature lives.

### 3. Two top-level config keys, both user-editable

`smart_filenames bool` and `filename_model Value` added to `config.Config`.
Reasons for top-level (not nested under `openrouter:`): `WriteSettings` only
rewrites top-level keys, so top-level placement lets the editor own them with no
writer changes. `filename_model` is a `Value` for consistency with other string
settings (supports `{env:}`/`{cmd:}`), resolved lazily only when naming runs.

`smart_filenames` is a plain `bool` (default `false`) rather than the `*bool`
tri-state used by `history`, because its default is "off" and absent already
means off — no need to distinguish unset from explicit false.

Both keys are added to `EditableSettings` + `WriteSettings` and documented in
`starter.go`'s template (the starter test enforces valid YAML).

### 4. Editor UX: toggle + text field + link

In `internal/tui/config.go`: add two `configField` enum entries
(`fieldSmartFilenames`, `fieldFilenameModel`) before `configFieldCount`; add a
`bool` and a `textinput.Model` to `configState`; seed in `newConfigState` /
`openConfig`; handle `space` for the toggle and route text to the model field;
persist in `confirmConfig` via `WriteSettings` and apply live to
`m.deps.Config`. The models-page link
(`https://openrouter.ai/models`) is rendered as a faint helper line near the
filename-model row in `configView`. No validation on the model id beyond
trimming — an unusable id simply causes a best-effort naming failure at run
time, which falls back to the timestamp.

### 5. Sanitization (dash slug) lives in `gen`

A `sanitizeBaseName(string) string` in `gen` turns the model's response into a
dash-separated slug: lowercase, strip any extension and path separators, replace
every run of non-alphanumeric characters with a single `-`, trim leading/
trailing dashes, and bound length (e.g. 60 chars). An empty result → `""` →
timestamp fallback. Keeping it in `gen` co-locates it with `Save`'s collision
handling.

Collision handling is **not new code**: `Save`'s existing `-N` loop is reused
verbatim, only its `base` source changes (timestamp → slug when naming is on).
So re-generating with the same prompt yields `a-red-fox.png`, then
`a-red-fox-1.png`, `a-red-fox-2.png`, and so on — the exact `-1`/`-2` suffixing
that timestamped names already get within the same second.

## Risks / Trade-offs

- **Extra latency and cost per generation when enabled** → Off by default;
  single short completion to a cheap model; failure/cancellation falls back to
  timestamp so it never blocks. Consider a short independent timeout on the
  naming call so a slow text model does not stall saving.
- **Model returns junk / unsafe / overlong text** → Sanitization + empty-check +
  timestamp fallback; collision loop prevents overwrite.
- **Prompt is sent to a second model (privacy)** → Documented in the starter
  config and proposal; strictly opt-in.
- **Interface growth on `Provider`** → One method; keeps the single-seam rule
  intact, which is cheaper than the alternative of leaking OpenRouter into `gen`.
- **Naming-call cancellation vs generation cancellation** → The image is already
  produced before naming runs; cancelling the naming call (or the parent ctx)
  must still save the image under the timestamp, not discard it. The helper
  treats any ctx error as "no name".

## Open Questions

- Exact default/placeholder model id to suggest in the editor and starter config
  (e.g. `google/gemini-flash-1.5` vs `openai/gpt-4o-mini`). Deferrable: it does
  not change the specs, the approach, or the task breakdown — only the suggested
  text. Default empty (feature off) regardless.
