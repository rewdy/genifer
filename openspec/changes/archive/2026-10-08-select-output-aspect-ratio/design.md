# Design

## Context

See proposal.md — Why. Relevant current state:

- `provider.Capabilities.AspectRatios []string` carries the model's offered ratios; `openrouter.capsFrom` copies them straight from the API's `supported_parameters.aspect_ratio.values` in that order. The provider imposes no "default".
- The compose form (`internal/tui/view.go` `composeView`) renders the ratios only as a faint read-only hint. The one interactive compose widget is the prompt `textarea`; `handleComposeKey` intercepts a few keys and forwards the rest to it.
- `gen.Draft.AspectRatio` → `Draft.Request()` → `provider.GenerateRequest.AspectRatio` → `openrouter.generatePayload` (with `omitempty`) is fully wired, but `Model.composeDraft()` never sets `AspectRatio`, so it is always empty today.
- `config.State` holds only `LastModel`. `LoadState`/`SaveState` round-trip the whole struct. Both save sites construct a fresh `State{LastModel: ...}` — not read-modify-write.

## Goals / Non-Goals

**Goals:**
- A keyboard-driven, inline aspect-ratio selector in the compose form, gated on the model offering ratios.
- Persist the last-used ratio and prefer it as the default when the current model offers it, else the model's first offered ratio.
- Fix `composeDraft` so the selection actually reaches the provider.

**Non-Goals:**
- No new selector for seed or any other parameter (aspect ratio only).
- No free-form / custom aspect ratios — selection is limited to the model's offered values.
- No provider-side validation that a chosen ratio is in the model's list (the UI only ever offers valid values).
- No change to how `openrouter` fetches or orders ratios.

## Decisions

### Selection dialog, not a cycle

The user needs to *see* all offered ratios, not blind-cycle through them. So the aspect-ratio control is a **selection dialog**: `ctrl+a` in compose opens a small overlay (a new `phaseAspect`) that lists every offered ratio with the current selection highlighted; `up`/`down` move the cursor, `enter` confirms the choice and returns to compose, `esc` cancels without changing the selection. The compose form shows the current value with a hint (e.g. `aspect: 16:9 (ctrl+a to change)`); the footer hint appears only when the model offers ratios.

- Why not a full `bubbles/list`: the ratio set is tiny (a handful of fixed strings) and needs no filtering, pricing, or scrolling. A plain indexed list rendered inline — a cursor index over `[]string` — is enough, keeps the dialog self-contained, and is trivial to unit-test without list plumbing.
- Why a phase rather than an inline focus toggle: the app already models modal interaction as `phase`, and the dialog is modal (keys drive the list, not the textarea). A dedicated `phaseAspect` reuses the existing `handleKey` dispatch and `bodyView` switch, and keeps `Update`/`View` free of extra focus state on the compose form itself.
- Alternative considered (and rejected): a cycle selector that advances the value on each keypress. Rejected because it hides the full set — a user who just wants to know what's available would have to cycle through everything.

### Selection state lives on the Model, defaulted on entering compose

Add `aspectRatio string` (the current selection) to the compose section of `Model`, plus a transient `aspectCursor int` for the dialog's highlighted row while it is open. `aspectRatio` is (re)computed whenever a model is chosen, via a helper that mirrors `preselectModel`:

```
pickAspectRatio(offered []string, last string) string
  // "" if offered is empty
  // last if last is in offered
  // offered[0] otherwise
```

Opening the dialog initializes `aspectCursor` to the index of the current `aspectRatio`. `enter` sets `aspectRatio = offered[aspectCursor]`; `esc` discards the cursor. `composeDraft` sets `AspectRatio: m.aspectRatio`. Recomputing on model-change ensures switching models never leaves a stale value that the new model doesn't offer.

- Alternative considered: deriving the selection lazily in `composeDraft` instead of storing it. Rejected — the view needs to display the current selection and the dialog needs to seed its cursor, so it must be stored state.

### Persist via read-modify-write to avoid clobbering

`State` gains `LastAspectRatio string json:"last_aspect_ratio,omitempty"`. The current save sites build a fresh `State{LastModel: ...}`, which would erase a second field. The fix: both the model-pick save and the new aspect-ratio save **load the current state, set the one field, and save** (or save both known fields together). The spec requires that persisting one selection preserves the other.

- When to save the ratio: on generate (consistent with "the last-used model is saved when chosen/generated"). Saving on every dialog confirm is unnecessary churn.
- Alternative considered: a combined `SaveState` call that always writes both model and ratio from current Model state. Viable too; the read-modify-write framing is chosen because the two save sites fire at different moments (model on pick, ratio on generate) and each should preserve whatever else is there.

## Risks / Trade-offs

- [`ctrl+a` is a common "move to line start" readline binding in the textarea] → In compose, `ctrl+a` is intercepted to open the dialog before the key reaches the textarea, so line-start navigation is unavailable via that chord. Acceptable: the compose prompt is a free-form image prompt, not a code editor, and the chord is only shadowed when the model offers ratios. Revisit with a different chord if it proves annoying.
- [Last-used ratio is global, not per-model] → A single remembered ratio is shared across models. A ratio like `21:9` remembered from one model simply won't apply to a model that doesn't offer it (falls back to first). This is the intended, simple behavior per the proposal; per-model memory is out of scope.
- [Stale selection on model switch] → Mitigated by recomputing `aspectRatio` via `pickAspectRatio` whenever the active model changes.
