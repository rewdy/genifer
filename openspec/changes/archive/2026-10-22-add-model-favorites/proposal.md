# Proposal

## Why

The model picker lists every model the provider offers, so the handful a user
actually uses are buried among dozens of others on every launch. Users need a
way to mark a few models as preferred and have them appear at the top of the
list without losing access to the rest.

## What Changes

- Add a per-model **favorite** toggle inside the model picker, bound to a
  single keypress on the highlighted row, with a visible indicator on
  favorited rows.
- Sort favorited models to the top of the picker list, above the remaining
  models, which stay in their existing provider order.
- Persist the favorite set in the app-owned `state.json` (never
  `config.yaml`), so favorites survive restarts and stay separate from
  hand-edited configuration.
- Treat favorites that no longer exist in the provider's model list as inert:
  they are ignored for display and ordering, never an error.
- Keep filtering intact: when a filter query is active, matching favorites
  still sort above other matches.

Terminology decision: the feature is called **favorites**, not "pin". A
favorite is a set membership the user assigns to a model, which is how this
feature behaves; "pin" implies a sticky position or manual ordering that the
app does not offer (the list is re-sorted, not manually arranged), and the
familiar star/favorite affordance communicates a personal preference list
better than a pin metaphor.

## Capabilities

### New Capabilities

- `model-favorites`: Marking models as favorites in the picker, ordering
  favorited models first, and persisting the favorite set in app-owned state.

### Modified Capabilities

- None

## Impact

- `internal/config/state.go`: `State` gains a favorite model-id set, still
  loaded/saved through the existing resilient `state.json` path.
- `internal/tui/model.go`, `internal/tui/delegate.go`: picker item model,
  ordering, favorite-toggle keybinding, and row indicator.
- `internal/tui/view.go` (and any keybinding hints): surface the toggle key.
- `openspec/specs/model-favorites/spec.md`: new capability spec.
- No provider or generation-flow changes; no `config.yaml` changes.
