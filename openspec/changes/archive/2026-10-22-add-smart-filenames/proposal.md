# Proposal

## Why

Generated images are saved with timestamp-only names (`2026-01-02-150405.png`),
which carry no meaning and make a folder of outputs hard to scan or search.
Users who want descriptive names must rename files by hand. An opt-in step that
asks a cheap LLM to turn the prompt into a short filename gives meaningful names
for pennies, while leaving the default behavior unchanged for everyone else.

## What Changes

- Add an **opt-in** "smart filenames" feature: when enabled, the prompt is sent
  to a low-cost OpenRouter text model that returns a short, descriptive base
  filename, used in place of the timestamp name. When disabled (the default),
  behavior is unchanged.
- Add two top-level `config.yaml` settings:
  - `smart_filenames` (bool, default `false`) — the on/off switch.
  - `filename_model` (string, default empty) — the OpenRouter model id used for
    naming (free-text; e.g. `google/gemini-flash-1.5`).
- Extend the config editor (TUI) with a toggle row for `smart_filenames` and a
  free-text field for `filename_model`, plus a visible link to the OpenRouter
  models page so users can find a model id.
- Extend the provider seam with a text-completion method so the naming call goes
  through the provider interface rather than bypassing it; implement it in the
  OpenRouter provider against the chat-completions endpoint.
- The naming step is **best-effort**: any failure (model unset, request error,
  empty/unusable result, cancellation of the naming call alone) falls back to
  the existing timestamp name. It never fails or blocks a generation.

## Capabilities

### New Capabilities
- `smart-filenames`: Opt-in generation of a meaningful image filename from the
  prompt via a low-cost provider text model, with sanitization, collision
  handling, and a guaranteed timestamp fallback.

### Modified Capabilities
- `image-provider`: Add a provider-interface text-completion operation (used to
  turn a prompt into a filename) and its OpenRouter implementation, keeping the
  rule that callers depend only on the interface.
- `config`: Add the `smart_filenames` and `filename_model` settings, their
  defaults, and their presence in the starter config.
- `config-ui`: The configuration view exposes the smart-filenames toggle and the
  filename-model field, writes them back to `config.yaml`, applies them to the
  running session, and shows the OpenRouter models-page link.
- `generation-workflow`: The save step derives the file's base name from the
  smart-filename step when enabled, falling back to the timestamped name
  otherwise; naming never blocks or fails generation.

## Impact

- **Code**: `internal/provider/provider.go` (new interface method + request
  type), `internal/provider/openrouter/` (chat-completions call reusing
  `authHeader`, HTTP client, error mappers), `internal/gen/save.go` and
  `internal/gen/run.go` (accept and apply a base name; insert the naming step),
  `internal/config/config.go` (new fields + defaults), `internal/config/writer.go`
  (`EditableSettings` + `WriteSettings` for the two top-level keys),
  `internal/config/starter.go` (document the keys), `internal/tui/config.go` and
  `internal/tui/view.go` (editor rows, link, key handling).
- **Config file**: two new top-level keys; `config.yaml` stays user-owned and is
  only touched through the existing targeted writer.
- **Dependencies**: none added; reuses the existing OpenRouter client and
  `net/http`.
- **Cost/privacy**: when enabled, the prompt text is sent to an additional
  (text) model, incurring a small extra cost per generation. Off by default.
