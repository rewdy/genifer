# Design

## Context

See proposal.md for motivation and specs/model-favorites/spec.md for the
behavior contract.

Current relevant state:

- `config.State` (`internal/config/state.go`) is the app-owned `state.json`:
  `LastModel`, `LastAspectRatio`, `LastTransparentBackground`. It is loaded with
  a tolerant `LoadState` (any read/unmarshal failure yields a zero `State`) and
  written with `SaveState`; the TUI repeatedly does load-mutate-save on it
  (`internal/tui/model.go` picker `enter` handler, aspect/transparency
  persistence).
- The picker is a `bubbles/list` whose items are `modelItem` values built in
  `rebuildItems()` from `m.models` (the provider-ordered slice) plus the pricing
  map. Selection is tracked in two ways: `m.selected` (index into `m.models`)
  and the list's own index. On `enter` the handler maps the selected item back
  to `m.models` with `indexOfModel(m.models, it.m.ID)`, so item order and
  `m.models` order are already decoupled by ID.
- `modelsLoadedMsg` pre-selects with `preselectModel(m.models, last)` and then
  `m.picker.Select(idx)` — an index into `m.models` assumed to equal the list
  index.
- Picker keybindings are handled in `handlePickerKey`; `f` is unused there
  (it means "open folder" only on the result screen). Help hints for the picker
  are composed in `internal/tui/view.go`.
- Rendering is a one-line custom `modelDelegate` (`internal/tui/delegate.go`)
  that composes cursor + name + muted detail and fills the row width.

## Goals / Non-Goals

**Goals:**

- Store favorites in `state.json` with no schema version bump and no migration.
- Reorder only the picker's item list; leave `m.models` (provider order) and the
  model→index mapping untouched.
- Keep the favorite toggle a pure state change in `Update`, with file I/O done
  through the existing synchronous `config.SaveState` pattern already used by
  the picker (state writes are tiny and already synchronous there).
- No new dependencies.

**Non-Goals:**

- Manual/reorderable favorites (drag order, pin positions) — favorites are a
  set, ordered by the provider's list.
- A separate config.yaml key for favorites; `config.yaml` stays user-owned.
- A dedicated favorites screen or filter.
- Favoriting from any screen other than the picker.

## Decisions

### Call the feature "favorites", keyed by `f`

A favorite is set membership, which matches the stored model (a set of IDs) and
the star-style indicator. "Pin" would imply ordering semantics we do not
implement. `f` is free in the picker and matches the word. The hint
(`f` favorite) is added to the picker help row; while the filter is active the
key is typed into the filter as usual, since `handlePickerKey` returns early for
the filtering state.

### Persist as `Favorites []string` in `config.State`

```go
// Favorites is the set of model ids the user marked as favorites.
Favorites []string `json:"favorites,omitempty"`
```

A slice (not a map) keeps the JSON readable and diff-friendly, dedupes are
trivial, and `omitempty` means existing `state.json` files stay valid with no
migration. Order in the slice is irrelevant because display order comes from the
provider list; treat it as a set in code via a helper that converts to
`map[string]struct{}`. Writes go through the existing load-mutate-`SaveState`
pattern, which already preserves sibling fields.

Stale IDs are intentionally kept: unlike `LastModel` (which is only consulted
for pre-selection), a favorite reappearing later should light up again. The
ordering helper simply never sees the stale ID because it iterates `m.models`.

### Order in `rebuildItems`, not in `m.models`

`rebuildItems()` builds the item slice; apply a stable partition there:

```go
func favoritesFirst(models []provider.Model, favs map[string]struct{}) []provider.Model
```

returns a new slice with favorites first, each group keeping provider order
(`sort.SliceStable` on a favorite-rank key, or a two-pass append). `m.models`
stays provider-ordered, so:

- `indexOfModel(m.models, …)`, `preselectModel`, and the aspect/transparency
  lookup keep working unchanged;
- toggling a favorite only rebuilds items, no model state changes.

`rebuildItems` must preserve the *highlighted model*, not the numeric index:
capture the currently selected item ID before `SetItems`, then `Select` the item
whose ID matches after rebuild. This is required by the "selection stays on the
toggled model" scenario and also fixes the pre-existing index assumption in
pre-selection: after ordering, `Select` must be done by ID. Add a small helper
`selectModelByID(list, id)`, and use it both for the favorite toggle and for the
`modelsLoadedMsg` pre-selection path (`preselectModel` still picks the default
ID from `m.models`, but selection in the list is by ID).

Filtering needs no extra work: `bubbles/list` filters the items it was given, so
whatever order `SetItems` supplied is preserved among matches, satisfying
"favorites first within filter results".

### Favorite state lives on the model (session snapshot), not read per render

Add `favorites map[string]struct{}` to `Model`, loaded from `state.json` when
models are loaded, and pass a `favorite bool` into each `modelItem` during
`rebuildItems` so the delegate renders the indicator without doing I/O during
`View`. Toggling updates the map, writes state, and calls `rebuildItems`.

### Delegate indicator: yellow `★`

Render a yellow `★` (U+2605) in a fixed-width prefix between the cursor and the
name for favorited rows, and the same width of blank padding for non-favorited
rows so names stay aligned. The palette has no yellow yet, so add one to
`internal/tui/styles.go` alongside `headerBG`/`inputBG`:

```go
// favoriteStar is the warm yellow used for the favorite star in the model
// picker, distinct from the pink selection accent.
const favoriteStar = lipgloss.Color("#f5c542")
```

Apply it with the row's app background and regardless of selection state, so the
star reads as yellow on both normal and highlighted rows. Width accounting stays
inside the existing padding logic (the line is truncated then padded to list
width), so no new layout rules.

## Risks / Trade-offs

- **Index/list drift after reorder** → always select by item ID after
  `SetItems`; document the invariant that `m.models` is provider-ordered and
  only the list is reordered.
- **Glyph availability** → `★` (U+2605) is widely supported in terminals; the
  star is a prefix so the model name remains readable even if a terminal
  substitutes a fallback glyph.
- **Frequent `state.json` writes while toggling several favorites** → each
  toggle is one small synchronous write, same cost as existing selection
  persistence; acceptable and keeps the code simple (no debounce task).
- **Stale favorites accumulate** → bounded by how many distinct models a user
  ever favorites; invisible to the user by design.

## Migration Plan

None. `state.json` gains an optional key; older files load as "no favorites",
newer files are ignored by older builds. No config.yaml change, no data loss.

## Open Questions

None.
