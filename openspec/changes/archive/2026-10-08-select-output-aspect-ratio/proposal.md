# Proposal

## Why

Models expose their selectable aspect ratios (output sizes), but the compose form only lists them as read-only text — the user can see the options yet has no way to pick one, so every generation falls back to the provider's default. Letting the user choose, and remembering that choice, turns a visible-but-dead list into a working control.

## What Changes

- The compose form's aspect-ratio hint becomes a **selectable control**: when the current model offers aspect ratios, the user can cycle through them and the chosen value is carried into the generation request.
- The chosen aspect ratio is **persisted as a last-used selection** in `state.json` (never `config.yaml`).
- On entering compose for a model that offers aspect ratios, the form **defaults the selection** to the last-used value when that model still offers it, otherwise to the model's first offered ratio.
- A model that offers no aspect ratios shows no selector (unchanged behavior), and `composeDraft` now actually sets `AspectRatio` on the draft (today it never does, so the plumbing to the provider is dead).

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `tui`: the "Capability-adaptive generate form" requirement changes so aspect ratio is a user-selectable control with a defined default-selection rule, not a static hint.
- `config`: the "last-used selections" requirement changes to also persist the last-used aspect ratio alongside the last-used model.

## Impact

- `internal/tui/model.go`: compose-phase state for the current aspect-ratio selection; key handling in `handleComposeKey`; `composeDraft` sets `AspectRatio`; default-selection on entering compose; preserve both persisted fields when saving state.
- `internal/tui/view.go`: `composeView` renders the selector (current value + how to change it) in place of the static hint; `footerView` key hint.
- `internal/tui/form.go`: helper to pick the default/last-used ratio (mirrors `preselectModel`).
- `internal/config/state.go`: add a `LastAspectRatio` field to `State`.
- No provider or dependency changes; `gen.Draft.AspectRatio` → `provider.GenerateRequest.AspectRatio` plumbing already exists.
